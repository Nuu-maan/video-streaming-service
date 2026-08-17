"use client";

import { useCallback, useEffect, useRef, useState } from "react";

import { pollVideoStatus } from "@/features/upload/actions";
import type { UploadProgress, UploadState } from "@/features/upload/types";
import type { UploadDetails } from "@/features/upload/schemas";
import type { Video } from "@/types/common";

/**
 * Drives one upload end to end: transfer → transcode → ready.
 *
 * The transfer goes through XMLHttpRequest, not fetch, deliberately: fetch
 * cannot report request-body progress (its Response streaming covers the
 * download side only), and a multi-gigabyte upload with no progress bar is
 * unusable. XHR's `upload.onprogress` is still the only browser API that
 * reports bytes *sent*.
 *
 * It posts to our own `/api/upload` route handler rather than the Go API:
 * the bearer token lives in an httpOnly cookie this JavaScript cannot read
 * (by design), so the handler attaches it server-side and streams the body
 * through. Same-origin also means the cookie rides along automatically.
 */

/** Speed is measured over a sliding window so it reflects "now", not the average since the start. */
const SPEED_WINDOW_MS = 8_000;
/** Transcoding progress poll cadence. */
const POLL_INTERVAL_MS = 2_500;

interface Sample {
  at: number;
  loaded: number;
}

function computeProgress(loaded: number, total: number, samples: Sample[]): UploadProgress {
  const now = performance.now();
  samples.push({ at: now, loaded });
  while (samples.length > 2 && now - samples[0].at > SPEED_WINDOW_MS) {
    samples.shift();
  }

  const first = samples[0];
  const elapsedMs = now - first.at;
  const bytesPerSecond = elapsedMs > 500 ? ((loaded - first.loaded) / elapsedMs) * 1000 : 0;

  const remaining = total - loaded;
  const etaSeconds = bytesPerSecond > 0 ? Math.ceil(remaining / bytesPerSecond) : null;

  return {
    loadedBytes: loaded,
    totalBytes: total,
    percent: total > 0 ? Math.min(100, Math.floor((loaded / total) * 100)) : 0,
    bytesPerSecond,
    etaSeconds,
  };
}

/** The API's error envelope, as XHR hands it back. */
function messageFromResponse(status: number, responseText: string): string {
  let serverMessage: string | null = null;
  try {
    const body: unknown = JSON.parse(responseText);
    serverMessage = (body as { error?: { message?: string } }).error?.message ?? null;
  } catch {
    // A proxy's HTML error page; the status code is all we have.
  }

  // Uploads carry their own rate limit: 5 per hour. Name it — "slow down" as a
  // generic error would read as a bug.
  if (status === 429) return "You've hit the upload limit — 5 uploads per hour. Try again in a while.";
  if (status === 413) return "The server refused the file as too large (the limit is 2 GB).";
  if (status === 415) return serverMessage ?? "That video format isn't supported.";
  if (status === 401) return "Your session has expired. Sign in again, then retry the upload.";
  return serverMessage ?? "The upload failed. Try again.";
}

export function useUpload() {
  const [state, setState] = useState<UploadState>({ phase: "idle" });
  const xhrRef = useRef<XMLHttpRequest | null>(null);
  const samplesRef = useRef<Sample[]>([]);

  const start = useCallback((file: File, details: UploadDetails) => {
    const form = new FormData();
    form.append("video", file);
    form.append("title", details.title);
    form.append("description", details.description);
    form.append("visibility", details.visibility);

    const xhr = new XMLHttpRequest();
    xhrRef.current = xhr;
    samplesRef.current = [{ at: performance.now(), loaded: 0 }];

    xhr.upload.onprogress = (event) => {
      if (!event.lengthComputable) return;
      setState({
        phase: "uploading",
        progress: computeProgress(event.loaded, event.total, samplesRef.current),
      });
    };

    xhr.onload = () => {
      xhrRef.current = null;
      // Any 2xx: the API answers 201 today, but a success is a success and an
      // upload that completed must never be reported as failed.
      if (xhr.status >= 200 && xhr.status < 300) {
        try {
          const body: unknown = JSON.parse(xhr.responseText);
          const video = (body as { data: Video }).data;
          setState({ phase: "processing", video, transcodingProgress: video.transcoding_progress ?? 0 });
          return;
        } catch {
          // 201 with an unparsable body — the video exists; the studio list will show it.
          setState({ phase: "failed", message: "Upload finished but the response was unreadable. Check your studio — the video is likely there." });
          return;
        }
      }
      setState({ phase: "failed", message: messageFromResponse(xhr.status, xhr.responseText) });
    };

    xhr.onerror = () => {
      xhrRef.current = null;
      setState({ phase: "failed", message: "The connection dropped mid-upload. Check your network and try again." });
    };

    xhr.onabort = () => {
      xhrRef.current = null;
      setState({ phase: "idle" });
    };

    xhr.open("POST", "/api/upload");
    xhr.send(form);

    setState({
      phase: "uploading",
      progress: {
        loadedBytes: 0,
        totalBytes: file.size,
        percent: 0,
        bytesPerSecond: 0,
        etaSeconds: null,
      },
    });
  }, []);

  /** Aborts the in-flight XHR — a real cancel, not a hidden progress bar. */
  const cancel = useCallback(() => {
    xhrRef.current?.abort();
  }, []);

  const reset = useCallback(() => {
    xhrRef.current?.abort();
    setState({ phase: "idle" });
  }, []);

  /**
   * Transcoding poll. Only the transfer needs the tab open; once the phase is
   * "processing" the worker owns the job, and this is purely observational —
   * closing the tab loses nothing.
   */
  const videoId = state.phase === "processing" ? state.video.id : null;
  useEffect(() => {
    if (!videoId) return;
    // Pinned to a local so the narrowing survives into the hoisted `tick`
    // below — TypeScript will not carry it across a function declaration.
    const id = videoId;

    let stopped = false;
    let timer: ReturnType<typeof setTimeout>;

    async function tick() {
      const result = await pollVideoStatus(id);
      if (stopped) return;

      if (result.ok) {
        const { status, progress } = result.report;
        if (status === "ready") {
          setState((current) =>
            current.phase === "processing" ? { phase: "ready", video: current.video } : current,
          );
          return;
        }
        if (status === "failed") {
          setState({
            phase: "failed",
            message: result.report.message ?? "Processing failed. You can retry from your studio, or upload again.",
          });
          return;
        }
        setState((current) =>
          current.phase === "processing"
            ? { ...current, transcodingProgress: progress ?? current.transcodingProgress }
            : current,
        );
      }
      // A failed poll is a blip, not a failed upload — keep the last known
      // progress on screen and try again.
      timer = setTimeout(tick, POLL_INTERVAL_MS);
    }

    timer = setTimeout(tick, POLL_INTERVAL_MS);
    return () => {
      stopped = true;
      clearTimeout(timer);
    };
  }, [videoId]);

  /**
   * Leaving the page kills an in-flight transfer (it is this tab doing the
   * sending), so warn — but only during "uploading". During "processing" the
   * user is free to go; the server keeps working.
   */
  const isUploading = state.phase === "uploading";
  useEffect(() => {
    if (!isUploading) return;
    const warn = (event: BeforeUnloadEvent) => {
      event.preventDefault();
    };
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [isUploading]);

  // Abort on unmount so a navigation away does not leave a zombie transfer.
  useEffect(() => () => xhrRef.current?.abort(), []);

  return { state, start, cancel, reset };
}

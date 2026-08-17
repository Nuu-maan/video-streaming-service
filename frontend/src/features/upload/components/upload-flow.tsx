"use client";

import { CircleCheck, Clapperboard, Play, RotateCcw } from "lucide-react";
import Link from "next/link";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { ErrorState } from "@/components/common/error-state";
import { routes } from "@/config/routes";
import { UploadDropzone } from "@/features/upload/components/upload-dropzone";
import { UploadForm } from "@/features/upload/components/upload-form";
import { UploadProgress } from "@/features/upload/components/upload-progress";
import { UploadStages } from "@/features/upload/components/upload-stages";
import { useUpload } from "@/features/upload/hooks/use-upload";
import type { UploadDetails } from "@/features/upload/schemas";

/**
 * The whole upload experience on one screen: pick a file, describe it, watch
 * it transfer, watch it transcode, get the link. One client component owns the
 * state so a cancel or a failure can land the user exactly where they left
 * off — file still chosen, description still typed.
 */
export function UploadFlow() {
  const { state, start, cancel, reset } = useUpload();
  const [file, setFile] = useState<File | null>(null);
  const [details, setDetails] = useState<UploadDetails | undefined>(undefined);

  function handleSubmit(submitted: UploadDetails) {
    if (!file) return;
    setDetails(submitted);
    start(file, submitted);
  }

  function startOver() {
    reset();
    setFile(null);
    setDetails(undefined);
  }

  if (state.phase === "idle") {
    return file ? (
      <UploadForm
        file={file}
        initialDetails={details}
        onSubmit={handleSubmit}
        onChangeFile={() => setFile(null)}
      />
    ) : (
      <UploadDropzone onSelect={setFile} />
    );
  }

  if (state.phase === "failed") {
    return (
      <ErrorState
        title="The upload didn't make it"
        description={state.message}
        action={
          <div className="flex items-center gap-2">
            {file ? (
              <Button onClick={reset}>
                <RotateCcw aria-hidden data-icon="inline-start" />
                Try again
              </Button>
            ) : null}
            <Button variant="outline" onClick={startOver}>
              Start over
            </Button>
          </div>
        }
      />
    );
  }

  return (
    <div className="flex flex-col gap-6">
      <UploadStages state={state} />

      {state.phase === "uploading" && file ? (
        <UploadProgress fileName={file.name} progress={state.progress} onCancel={cancel} />
      ) : null}

      {state.phase === "processing" ? (
        <div className="flex flex-col gap-3 rounded-xl bg-card p-4 shadow-border">
          <div className="flex items-baseline justify-between gap-4">
            <p className="min-w-0 truncate text-sm font-medium">{state.video.title}</p>
            <p className="shrink-0 text-sm font-medium tabular-nums" aria-hidden>
              {Math.round(state.transcodingProgress)}%
            </p>
          </div>
          <Progress
            value={state.transcodingProgress}
            aria-label="Transcoding progress"
            aria-valuetext={`${Math.round(state.transcodingProgress)}% processed`}
          />
          <div className="flex items-center justify-end">
            <Button asChild variant="ghost" size="sm">
              <Link href={routes.studio}>Go to your studio</Link>
            </Button>
          </div>
        </div>
      ) : null}

      {state.phase === "ready" ? (
        <div className="flex flex-col items-center gap-4 rounded-xl bg-card p-8 text-center shadow-border">
          <span className="flex size-14 items-center justify-center rounded-2xl bg-brand-500/10 text-brand-500 ring-1 ring-brand-500/20 ring-inset">
            <CircleCheck aria-hidden className="size-7" />
          </span>
          <div>
            <h2 className="text-heading text-balance">&ldquo;{state.video.title}&rdquo; is live</h2>
            <p className="mt-1 text-sm text-muted-foreground">
              Transcoded and ready to stream
              {state.video.available_qualities.length > 0
                ? ` in ${state.video.available_qualities.join(", ")}`
                : ""}
              .
            </p>
          </div>
          <div className="flex flex-wrap items-center justify-center gap-2">
            <Button asChild>
              <Link href={routes.video(state.video.id)}>
                <Play aria-hidden data-icon="inline-start" />
                Watch it
              </Link>
            </Button>
            <Button asChild variant="outline">
              <Link href={routes.studio}>
                <Clapperboard aria-hidden data-icon="inline-start" />
                Go to your studio
              </Link>
            </Button>
            <Button variant="ghost" onClick={startOver}>
              Upload another
            </Button>
          </div>
        </div>
      ) : null}
    </div>
  );
}

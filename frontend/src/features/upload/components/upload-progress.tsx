"use client";

import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import type { UploadProgress as UploadProgressData } from "@/features/upload/types";
import { formatBytes } from "@/lib/format";

interface UploadProgressProps {
  fileName: string;
  progress: UploadProgressData;
  /** Aborts the XHR — an actual cancel, not a dismissed progress bar. */
  onCancel: () => void;
}

/** "2 h 5 m left" reads like a hostage note; people think in rough time. */
function formatEta(seconds: number | null): string | null {
  if (seconds === null) return null;
  if (seconds < 5) return "almost done";
  if (seconds < 60) return "under a minute left";
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) return `about ${minutes} min left`;
  const hours = Math.floor(minutes / 60);
  const rest = minutes % 60;
  return rest > 0 ? `about ${hours} h ${rest} min left` : `about ${hours} h left`;
}

export function UploadProgress({ fileName, progress, onCancel }: UploadProgressProps) {
  const { loadedBytes, totalBytes, percent, bytesPerSecond, etaSeconds } = progress;
  // 100% with the request still open means the server is writing the file to
  // disk — say so, or the stuck bar reads as a hang.
  const finishing = percent >= 100;
  const eta = formatEta(etaSeconds);

  return (
    <div className="flex flex-col gap-3 rounded-xl bg-card p-4 shadow-border">
      <div className="flex items-baseline justify-between gap-4">
        <p className="min-w-0 truncate text-sm font-medium">{fileName}</p>
        <p className="shrink-0 text-sm font-medium tabular-nums" aria-hidden>
          {percent}%
        </p>
      </div>

      <Progress
        value={percent}
        aria-label={`Uploading ${fileName}`}
        aria-valuetext={`${percent}% uploaded`}
      />

      <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
        <p className="text-xs text-muted-foreground tabular-nums">
          {finishing ? (
            "Finishing up — the server is storing your file…"
          ) : (
            <>
              {formatBytes(loadedBytes)} of {formatBytes(totalBytes)}
              {bytesPerSecond > 0 ? <> · {formatBytes(bytesPerSecond)}/s</> : null}
              {eta ? <> · {eta}</> : null}
            </>
          )}
        </p>
        <Button type="button" variant="ghost" size="sm" onClick={onCancel} disabled={finishing}>
          Cancel
        </Button>
      </div>
    </div>
  );
}

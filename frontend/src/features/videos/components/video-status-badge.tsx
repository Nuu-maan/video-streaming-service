import { CircleAlert, LoaderCircle } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type { VideoStatus } from "@/types/common";

interface VideoStatusBadgeProps {
  status: VideoStatus;
  /** 0–100, shown while processing. */
  progress?: number;
  className?: string;
}

/**
 * Lifecycle badge for a video that is not (yet) watchable. Renders nothing for
 * "ready" — an unremarkable state needs no label. Pure and server-compatible;
 * ProcessingPoller re-renders it with live values.
 */
export function VideoStatusBadge({ status, progress, className }: VideoStatusBadgeProps) {
  if (status === "ready") return null;

  if (status === "failed") {
    return (
      <Badge variant="destructive" className={className}>
        <CircleAlert aria-hidden />
        Processing failed
      </Badge>
    );
  }

  const percent =
    status === "processing" && typeof progress === "number"
      ? Math.min(100, Math.max(0, Math.round(progress)))
      : null;

  return (
    <Badge variant="secondary" className={cn("tabular-nums", className)}>
      <LoaderCircle aria-hidden className="animate-spin" />
      {status === "uploading" ? "Uploading" : percent === null ? "Processing" : `Processing ${percent}%`}
    </Badge>
  );
}

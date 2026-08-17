"use client";

import { FileVideo, Upload } from "lucide-react";
import { useRef, useState } from "react";

import { formatExtensionList, validateVideoFile } from "@/features/upload/schemas";
import { limits } from "@/config/site";
import { cn } from "@/lib/utils";

interface UploadDropzoneProps {
  onSelect: (file: File) => void;
  className?: string;
}

/**
 * Drag-and-drop with click-to-browse. Validation (extension allowlist, 2 GB
 * cap) runs the instant a file is picked — before a byte is uploaded — because
 * rejecting a 2 GB file *after* uploading it is unforgivable.
 */
export function UploadDropzone({ onSelect, className }: UploadDropzoneProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [dragging, setDragging] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // dragenter/dragleave fire for every child crossed; a depth counter is the
  // only way "still inside the zone" is knowable without flicker.
  const dragDepth = useRef(0);

  function accept(file: File | undefined) {
    if (!file) return;
    const problem = validateVideoFile(file);
    if (problem) {
      setError(problem);
      return;
    }
    setError(null);
    onSelect(file);
  }

  return (
    <div className={className}>
      <button
        type="button"
        onClick={() => inputRef.current?.click()}
        onDragEnter={(event) => {
          event.preventDefault();
          dragDepth.current += 1;
          setDragging(true);
        }}
        onDragOver={(event) => event.preventDefault()}
        onDragLeave={() => {
          dragDepth.current -= 1;
          if (dragDepth.current <= 0) {
            dragDepth.current = 0;
            setDragging(false);
          }
        }}
        onDrop={(event) => {
          event.preventDefault();
          dragDepth.current = 0;
          setDragging(false);
          accept(event.dataTransfer.files[0]);
        }}
        aria-describedby={error ? "dropzone-error" : undefined}
        className={cn(
          "flex min-h-72 w-full flex-col items-center justify-center gap-1 rounded-xl border border-dashed px-6 py-12 text-center outline-none",
          "transition-[border-color,background-color,transform] duration-(--motion-fast) ease-(--ease-out-quart)",
          "focus-visible:ring-3 focus-visible:ring-ring/50",
          dragging
            ? "scale-[1.01] border-brand-500 bg-brand-500/5"
            : "border-border hover:border-muted-foreground/40 hover:bg-muted/30 active:scale-[0.99]",
        )}
      >
        <span
          className={cn(
            "mb-3 flex size-14 items-center justify-center rounded-2xl ring-1 ring-inset",
            "transition-colors duration-(--motion-fast)",
            dragging
              ? "bg-brand-500/10 text-brand-500 ring-brand-500/30"
              : "bg-muted text-muted-foreground ring-border/60",
          )}
        >
          {dragging ? <FileVideo aria-hidden className="size-6" /> : <Upload aria-hidden className="size-6" />}
        </span>

        <span className="text-heading text-balance">
          {dragging ? "Drop it here" : "Drag and drop a video"}
        </span>
        <span className="text-sm text-muted-foreground">
          or <span className="font-medium text-foreground underline underline-offset-4">browse your files</span>
        </span>
        <span className="mt-3 text-xs text-muted-foreground">
          {formatExtensionList()} · up to 2 GB
        </span>
      </button>

      <input
        ref={inputRef}
        type="file"
        accept={[...limits.acceptedVideoExtensions, ...limits.acceptedVideoTypes].join(",")}
        className="sr-only"
        tabIndex={-1}
        onChange={(event) => {
          accept(event.target.files?.[0]);
          // Same file picked twice must fire change again.
          event.target.value = "";
        }}
      />

      {error ? (
        <p id="dropzone-error" role="alert" className="mt-3 text-sm text-destructive">
          {error}
        </p>
      ) : null}
    </div>
  );
}

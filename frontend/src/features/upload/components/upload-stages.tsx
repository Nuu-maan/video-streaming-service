"use client";

import { Check, LoaderCircle } from "lucide-react";

import type { UploadState } from "@/features/upload/types";
import { cn } from "@/lib/utils";

interface UploadStagesProps {
  state: UploadState;
}

type StepStatus = "done" | "active" | "pending";

interface Step {
  label: string;
  detail: string | null;
  status: StepStatus;
}

/**
 * The honest lifecycle: Uploading → Processing → Ready. The transfer needs
 * this tab open; the transcode does not — the copy says so, because "can I
 * close this?" is the question everyone staring at a progress bar has.
 *
 * `detail` is STATIC per phase. It used to carry the live percentage inside the
 * sentence, and the sentence lived in an `aria-live` paragraph — so every poll
 * re-announced the whole of "Transcoding for streaming — 34%. You can leave this
 * page; it keeps going on the server." The guidance and the number now live in
 * different elements: the guidance is said once, and the number is said on a
 * throttled beat (see `announce`).
 */
function deriveSteps(state: UploadState): Step[] {
  const phase = state.phase;

  const uploading: StepStatus = phase === "uploading" ? "active" : "done";
  const processing: StepStatus =
    phase === "uploading" ? "pending" : phase === "processing" ? "active" : "done";
  const ready: StepStatus = phase === "ready" ? "done" : "pending";

  return [
    {
      label: "Upload",
      detail: phase === "uploading" ? "Sending your file — keep this tab open." : null,
      status: uploading,
    },
    {
      label: "Process",
      detail:
        phase === "processing"
          ? "Transcoding for streaming. You can leave this page; it keeps going on the server."
          : null,
      status: processing,
    },
    {
      label: "Ready",
      detail: phase === "ready" ? "Your video is live." : null,
      status: ready,
    },
  ];
}

/**
 * The one live region for the whole upload — transfer AND transcode.
 *
 * Quantised to 10% for the transfer and 25% for the transcode, so it speaks a
 * dozen short strings across an upload rather than re-reading a paragraph on
 * every progress event. The percentage the EYE reads stays continuous (see the
 * progress bar); only the spoken one is stepped.
 *
 * The transfer had the opposite bug to the transcode: it was announced nowhere
 * at all. upload-progress.tsx renders its percentage `aria-hidden` and the bar
 * is not in a live region, so the byte transfer — the thing the person is
 * actually waiting on — was silent from beginning to end.
 */
function announce(state: UploadState): string {
  switch (state.phase) {
    case "uploading": {
      const step = Math.floor(Math.min(100, Math.max(0, state.progress.percent)) / 10) * 10;
      return step >= 100
        ? "Upload complete. The server is storing your file."
        : `Uploading, ${step} percent.`;
    }
    case "processing": {
      const step = Math.floor(Math.min(100, Math.max(0, state.transcodingProgress)) / 25) * 25;
      return step === 0
        ? "Upload complete. Processing your video."
        : `Processing, ${step} percent.`;
    }
    case "ready":
      return "Your video is ready.";
    default:
      return "";
  }
}

function StepIndicator({ status }: { status: StepStatus }) {
  if (status === "done") {
    return (
      <span className="flex size-7 items-center justify-center rounded-full bg-brand-500 text-white">
        <Check aria-hidden className="size-4" strokeWidth={3} />
      </span>
    );
  }
  if (status === "active") {
    return (
      <span className="flex size-7 items-center justify-center rounded-full border-2 border-brand-500 text-brand-500">
        <LoaderCircle aria-hidden className="size-4 animate-spin" />
      </span>
    );
  }
  return <span className="size-7 rounded-full border-2 border-border" />;
}

export function UploadStages({ state }: UploadStagesProps) {
  if (state.phase === "idle" || state.phase === "failed") return null;

  const steps = deriveSteps(state);
  const activeStep = steps.find((step) => step.status === "active") ?? steps[steps.length - 1];

  return (
    <div className="flex flex-col gap-3">
      <ol className="flex items-center gap-2">
        {steps.map((step, index) => (
          <li
            key={step.label}
            aria-current={step.status === "active" ? "step" : undefined}
            className={cn("flex items-center gap-2", index < steps.length - 1 && "flex-1")}
          >
            <StepIndicator status={step.status} />
            <span
              className={cn(
                "text-sm",
                step.status === "pending" ? "text-muted-foreground" : "font-medium text-foreground",
              )}
            >
              {step.label}
            </span>
            {index < steps.length - 1 ? (
              <span
                aria-hidden
                className={cn(
                  "h-px flex-1 rounded-full transition-colors duration-(--motion-medium)",
                  step.status === "done" ? "bg-brand-500/60" : "bg-border",
                )}
              />
            ) : null}
          </li>
        ))}
      </ol>

      {/* Static guidance. NOT a live region — it is the same sentence for the
          whole of a phase, and re-reading it on every tick is what made this
          unusable. */}
      <p className="min-h-5 text-sm text-pretty text-muted-foreground">{activeStep.detail}</p>

      {/* The only live region in the upload, covering both phases, throttled. */}
      <span role="status" aria-live="polite" className="sr-only">
        {announce(state)}
      </span>
    </div>
  );
}

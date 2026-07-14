"use client";

import { RotateCcw, TriangleAlert } from "lucide-react";

import { Button } from "@/components/ui/button";
import type { PlayerFailure } from "@/features/player/types";

interface PlayerErrorProps {
  failure: PlayerFailure;
  onRetry: () => void;
}

/**
 * What a broken stream looks like. Not a black rectangle: a black rectangle is
 * indistinguishable from a video that has not started, and the viewer will sit
 * there waiting for it.
 *
 * The copy never speculates about permissions. A video the API will not serve
 * answers 404 whether it is missing or merely private — that is deliberate, and
 * saying "you don't have access" would both leak and, half the time, lie.
 */
export function PlayerError({ failure, onRetry }: PlayerErrorProps) {
  return (
    <div
      role="alert"
      className="absolute inset-0 z-20 flex flex-col items-center justify-center gap-3 bg-black/85 px-6 text-center"
    >
      <div className="flex size-11 items-center justify-center rounded-2xl bg-white/10 text-white ring-1 ring-white/15 ring-inset">
        <TriangleAlert aria-hidden className="size-5" />
      </div>
      <div className="space-y-1">
        <p className="text-heading text-white">{failure.title}</p>
        <p className="max-w-sm text-sm text-pretty text-white/70">{failure.description}</p>
      </div>
      <Button
        variant="secondary"
        size="sm"
        onClick={onRetry}
        className="mt-1 bg-white/15 text-white hover:bg-white/25"
      >
        <RotateCcw aria-hidden />
        Try again
      </Button>
    </div>
  );
}

"use client";

import { LoaderCircle, RefreshCw } from "lucide-react";
import { useId, useState, useTransition } from "react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { FieldError } from "@/features/admin/components/field-error";
import { retryVideo } from "@/features/admin/actions";

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

/**
 * Retry a transcode by ID.
 *
 * This exists because the list above it is incomplete and cannot be made
 * complete: failed videos are found through `GET /videos?status=failed`, which
 * only returns public videos, so a *private* video that failed to transcode is
 * invisible to it. Its owner can see it in their studio; an admin cannot see it
 * anywhere. Rather than pretend the list is the whole truth, the page admits the
 * gap and gives the one way through it.
 *
 * The API is the real validator here — it 400s anything that is not in status
 * `failed`, and passes that sentence straight back — so this only checks the
 * shape, which is the part worth catching before a round trip.
 */
export function RetryVideoForm() {
  const fieldId = useId();
  const [videoId, setVideoId] = useState("");
  const [error, setError] = useState<string | undefined>();
  const [pending, startTransition] = useTransition();

  function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const id = videoId.trim();

    if (!UUID.test(id)) {
      setError("That isn't a valid video ID.");
      return;
    }
    setError(undefined);

    startTransition(async () => {
      const result = await retryVideo(id);
      if (result.ok) {
        toast.success(result.message);
        setVideoId("");
        return;
      }
      toast.error("Couldn't retry that transcode", { description: result.message });
    });
  }

  return (
    <form onSubmit={submit} className="flex flex-wrap items-start gap-3" noValidate>
      <div className="grid min-w-64 flex-1 gap-1.5">
        <Label htmlFor={fieldId} className="sr-only">
          Video ID
        </Label>
        <Input
          id={fieldId}
          value={videoId}
          onChange={(event) => setVideoId(event.target.value)}
          placeholder="00000000-0000-0000-0000-000000000000"
          className="font-mono text-sm"
          aria-invalid={Boolean(error)}
          autoComplete="off"
          spellCheck={false}
        />
        <FieldError message={error} />
      </div>

      <Button type="submit" variant="outline" disabled={pending}>
        {pending ? (
          <LoaderCircle aria-hidden data-icon="inline-start" className="animate-spin" />
        ) : (
          <RefreshCw aria-hidden data-icon="inline-start" />
        )}
        Retry
      </Button>
    </form>
  );
}

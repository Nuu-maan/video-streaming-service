"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { FileVideo, Globe, Link2, Lock } from "lucide-react";
import { useForm, useWatch } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { limits } from "@/config/site";
import { uploadDetailsSchema, type UploadDetails } from "@/features/upload/schemas";
import { formatBytes } from "@/lib/format";
import { cn } from "@/lib/utils";
import type { VideoVisibility } from "@/types/common";

interface UploadFormProps {
  file: File;
  /** Restores what was typed before a cancelled upload — losing a written description to a cancel is rude. */
  initialDetails?: UploadDetails;
  onSubmit: (details: UploadDetails) => void;
  onChangeFile: () => void;
}

/** Plain-English visibility choices — a select would hide exactly the text people need to make this call. */
const VISIBILITY_OPTIONS: Array<{
  value: VideoVisibility;
  label: string;
  description: string;
  icon: typeof Globe;
}> = [
  { value: "public", label: "Public", description: "Anyone can find and watch it.", icon: Globe },
  {
    value: "unlisted",
    label: "Unlisted",
    description: "Anyone with the link can watch. Hidden from search and listings.",
    icon: Link2,
  },
  { value: "private", label: "Private", description: "Only you can watch it.", icon: Lock },
];

/** "family-trip-2026.mp4" → "family-trip-2026" as a starting title. */
function titleFromFilename(name: string): string {
  const dot = name.lastIndexOf(".");
  return (dot > 0 ? name.slice(0, dot) : name).slice(0, limits.maxTitleLength);
}

export function UploadForm({ file, initialDetails, onSubmit, onChangeFile }: UploadFormProps) {
  const form = useForm<UploadDetails>({
    resolver: zodResolver(uploadDetailsSchema),
    defaultValues: initialDetails ?? {
      title: titleFromFilename(file.name),
      description: "",
      visibility: "public",
    },
  });

  // `useWatch`, not `form.watch`: the latter hands back a function the React
  // Compiler cannot memoize safely, so it bails out of optimising this whole
  // component. `useWatch` subscribes to one field and re-renders on it alone —
  // which is all the character counters below need.
  const title = useWatch({ control: form.control, name: "title" });
  const description = useWatch({ control: form.control, name: "description" });
  const errors = form.formState.errors;

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col gap-6">
      {/* The chosen file, with a way out that costs nothing. */}
      <div className="flex items-center gap-3 rounded-xl bg-card p-3 shadow-border">
        <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
          <FileVideo aria-hidden className="size-5" />
        </div>
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium">{file.name}</p>
          <p className="text-xs text-muted-foreground tabular-nums">{formatBytes(file.size)}</p>
        </div>
        <Button type="button" variant="ghost" size="sm" onClick={onChangeFile}>
          Change file
        </Button>
      </div>

      <div className="flex flex-col gap-2">
        <div className="flex items-baseline justify-between">
          <Label htmlFor="upload-title">Title</Label>
          <span
            className={cn(
              "text-xs tabular-nums",
              title.length > limits.maxTitleLength ? "text-destructive" : "text-muted-foreground",
            )}
          >
            {title.length}/{limits.maxTitleLength}
          </span>
        </div>
        <Input
          id="upload-title"
          autoComplete="off"
          aria-invalid={errors.title ? true : undefined}
          aria-describedby={errors.title ? "upload-title-error" : undefined}
          {...form.register("title")}
        />
        {errors.title ? (
          <p id="upload-title-error" role="alert" className="text-sm text-destructive">
            {errors.title.message}
          </p>
        ) : null}
      </div>

      <div className="flex flex-col gap-2">
        <div className="flex items-baseline justify-between">
          <Label htmlFor="upload-description">
            Description <span className="font-normal text-muted-foreground">(optional)</span>
          </Label>
          <span
            className={cn(
              "text-xs tabular-nums",
              description.length > limits.maxDescriptionLength ? "text-destructive" : "text-muted-foreground",
            )}
          >
            {description.length}/{limits.maxDescriptionLength.toLocaleString("en")}
          </span>
        </div>
        <Textarea
          id="upload-description"
          rows={4}
          placeholder="Tell viewers what this video is about"
          aria-invalid={errors.description ? true : undefined}
          aria-describedby={errors.description ? "upload-description-error" : undefined}
          {...form.register("description")}
        />
        {errors.description ? (
          <p id="upload-description-error" role="alert" className="text-sm text-destructive">
            {errors.description.message}
          </p>
        ) : null}
      </div>

      <fieldset className="flex flex-col gap-2">
        <legend className="mb-2 text-sm font-medium">Visibility</legend>
        <div className="grid gap-2 sm:grid-cols-3">
          {VISIBILITY_OPTIONS.map((option) => {
            const Icon = option.icon;
            return (
              <label
                key={option.value}
                className={cn(
                  "flex cursor-pointer flex-col gap-1.5 rounded-xl border border-border p-3 outline-none",
                  "transition-colors duration-(--motion-fast)",
                  "hover:bg-muted/40",
                  "has-checked:border-brand-500/50 has-checked:bg-brand-500/5",
                  "has-focus-visible:ring-3 has-focus-visible:ring-ring/50",
                )}
              >
                <input
                  type="radio"
                  value={option.value}
                  className="sr-only"
                  {...form.register("visibility")}
                />
                <span className="flex items-center gap-2 text-sm font-medium">
                  <Icon aria-hidden className="size-4 text-muted-foreground" />
                  {option.label}
                </span>
                <span className="text-xs text-pretty text-muted-foreground">{option.description}</span>
              </label>
            );
          })}
        </div>
      </fieldset>

      <div className="flex justify-end">
        <Button type="submit" size="lg" className="active:scale-[0.98]">
          Upload video
        </Button>
      </div>
    </form>
  );
}

"use client";

import { LoaderCircle } from "lucide-react";
import { useState, useTransition } from "react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { createPlaylist, updatePlaylist } from "@/features/playlists/actions";
import {
  MAX_PLAYLIST_DESCRIPTION,
  MAX_PLAYLIST_TITLE,
  visibilityOptions,
} from "@/features/playlists/schemas";
import type { Playlist, VideoVisibility } from "@/types/common";

interface PlaylistFormDialogProps {
  /** Provide to edit; omit to create. */
  playlist?: Playlist;
  trigger?: React.ReactNode;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
}

/**
 * One dialog for create and edit — the fields, the bounds and the validation are
 * identical, and two near-identical forms is how they drift apart.
 */
export function PlaylistFormDialog({
  playlist,
  trigger,
  open,
  onOpenChange,
}: PlaylistFormDialogProps) {
  const editing = Boolean(playlist);
  const [internalOpen, setInternalOpen] = useState(false);
  const [title, setTitle] = useState(playlist?.title ?? "");
  const [description, setDescription] = useState(playlist?.description ?? "");
  const [visibility, setVisibility] = useState<VideoVisibility>(playlist?.visibility ?? "private");
  const [pending, startTransition] = useTransition();

  const isOpen = open ?? internalOpen;
  const setOpen = onOpenChange ?? setInternalOpen;

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!title.trim()) return;

    startTransition(async () => {
      const input = { title: title.trim(), description: description.trim(), visibility };
      const result = playlist
        ? await updatePlaylist(playlist.id, input)
        : await createPlaylist(input);

      if (!result.ok) {
        toast.error(result.message);
        return;
      }

      setOpen(false);
      toast.success(editing ? "Playlist updated." : `Created “${result.playlist.title}”.`);
      if (!editing) {
        setTitle("");
        setDescription("");
        setVisibility("private");
      }
    });
  }

  return (
    <Dialog open={isOpen} onOpenChange={(next) => (pending ? undefined : setOpen(next))}>
      {trigger ? <DialogTrigger asChild>{trigger}</DialogTrigger> : null}
      <DialogContent className="sm:max-w-md">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>{editing ? "Edit playlist" : "New playlist"}</DialogTitle>
            <DialogDescription className="text-pretty">
              {editing
                ? "Change the name, the description, or who can see it."
                : "Collect videos to watch together. You can change any of this later."}
            </DialogDescription>
          </DialogHeader>

          <div className="my-5 flex flex-col gap-4">
            <div className="flex flex-col gap-2">
              <Label htmlFor="playlist-title">Name</Label>
              <Input
                id="playlist-title"
                value={title}
                maxLength={MAX_PLAYLIST_TITLE}
                onChange={(event) => setTitle(event.target.value)}
                disabled={pending}
                placeholder="Watch on the train"
                autoComplete="off"
                required
              />
            </div>

            <div className="flex flex-col gap-2">
              <Label htmlFor="playlist-description">
                Description <span className="font-normal text-muted-foreground">(optional)</span>
              </Label>
              <Textarea
                id="playlist-description"
                value={description}
                maxLength={MAX_PLAYLIST_DESCRIPTION}
                onChange={(event) => setDescription(event.target.value)}
                disabled={pending}
                rows={2}
                className="resize-none"
              />
            </div>

            <div className="flex flex-col gap-2">
              <Label htmlFor="playlist-visibility">Visibility</Label>
              <Select
                value={visibility}
                onValueChange={(value) => setVisibility(value as VideoVisibility)}
                disabled={pending}
              >
                <SelectTrigger id="playlist-visibility" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {visibilityOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      <span className="font-medium">{option.label}</span>
                      <span className="text-muted-foreground">{option.hint}</span>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" disabled={pending} onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={pending || !title.trim()}>
              {pending ? <LoaderCircle aria-hidden className="animate-spin" /> : null}
              {editing ? "Save changes" : "Create playlist"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

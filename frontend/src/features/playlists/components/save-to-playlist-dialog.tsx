"use client";

import { ListPlus, LoaderCircle, Plus, X } from "lucide-react";
import { useId, useState, useTransition } from "react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import {
  addVideoToPlaylist,
  createPlaylist,
  loadSaveTargets,
  toggleVideoInPlaylist,
} from "@/features/playlists/actions";
import { MAX_PLAYLIST_TITLE } from "@/features/playlists/schemas";
import type { SaveTarget } from "@/features/playlists/types";
import { useSignInPrompt } from "@/hooks/use-sign-in-prompt";

interface SaveToPlaylistDialogProps {
  videoId: string;
  isAuthenticated: boolean;
  /** Opens the popover. Defaults to a "Save" button. */
  trigger?: React.ReactNode;
}

/**
 * The playlists are fetched when the popover opens, not on page load: most
 * viewers never save a video, and none of them should pay for the request.
 *
 * A checkbox flips immediately and the row is disabled until the server agrees.
 * Both directions are forgiving server-side — adding a video that is already in
 * the playlist is a swallowed 409, removing one that is not is a swallowed 404 —
 * so a double-click cannot corrupt the list.
 */
/**
 * One playlist in the list.
 *
 * A Radix Checkbox renders a `<button role="checkbox">`, and an ARIA checkbox is
 * NOT reliably named by a wrapping `<label>` — which is what this used to be. A
 * signed-in user with a screen reader heard "checkbox, not checked" and was
 * never told which playlist. Worse, clicking the label forwarded a synthetic
 * click to the button, so the toggle could fire twice from one press.
 *
 * So: no wrapping label. The row is a plain flex container, the title has an id,
 * and the checkbox is named by `aria-labelledby`. One control, one click, one
 * name.
 *
 * `checked="indeterminate"` is the honest rendering of `contains: "unknown"` —
 * we do not know, so we do not draw a tick and we do not draw an empty box
 * either. Clicking it still adds.
 */
function SaveTargetRow({
  target,
  busy,
  onToggle,
}: {
  target: SaveTarget;
  busy: boolean;
  onToggle: () => void;
}) {
  const titleId = useId();
  const unknown = target.contains === "unknown";

  return (
    <li className="flex items-center gap-3 rounded-md px-2 py-2 transition-colors duration-(--motion-fast) hover:bg-muted/70 has-disabled:opacity-60">
      <Checkbox
        aria-labelledby={titleId}
        aria-describedby={unknown ? `${titleId}-unknown` : undefined}
        checked={target.contains === "unknown" ? "indeterminate" : target.contains}
        disabled={busy}
        onCheckedChange={onToggle}
      />
      <span id={titleId} className="min-w-0 flex-1 truncate text-sm">
        {target.title}
      </span>
      {unknown ? (
        <span id={`${titleId}-unknown`} className="sr-only">
          We couldn&apos;t check whether this video is already in this playlist. Selecting it will add
          the video.
        </span>
      ) : null}
      {busy ? (
        <LoaderCircle aria-hidden className="size-3.5 animate-spin text-muted-foreground" />
      ) : (
        <span className="text-xs text-muted-foreground tabular-nums">{target.videoCount}</span>
      )}
    </li>
  );
}

export function SaveToPlaylistDialog({
  videoId,
  isAuthenticated,
  trigger,
}: SaveToPlaylistDialogProps) {
  const [open, setOpen] = useState(false);
  const [targets, setTargets] = useState<SaveTarget[] | null>(null);
  const [loading, setLoading] = useState(false);
  const [creating, setCreating] = useState(false);
  const [newTitle, setNewTitle] = useState("");
  const [busyId, setBusyId] = useState<string | null>(null);
  const [savingNew, startSaveNew] = useTransition();
  const promptSignIn = useSignInPrompt();

  async function handleOpenChange(next: boolean) {
    if (next && !isAuthenticated) {
      promptSignIn("Sign in to save videos to a playlist.");
      return;
    }

    setOpen(next);
    if (!next || targets || loading) return;

    setLoading(true);
    const result = await loadSaveTargets(videoId);
    setLoading(false);

    if (!result.ok) {
      toast.error(result.message);
      setOpen(false);
      return;
    }
    setTargets(result.targets);
  }

  async function handleToggle(target: SaveTarget) {
    // Only a CONFIRMED membership removes. "unknown" adds — see the note on
    // `Membership`: we never destroy something on the strength of a guess.
    const next = target.contains !== true;
    setBusyId(target.id);
    setTargets(
      (current) =>
        current?.map((item) =>
          item.id === target.id
            ? {
                ...item,
                contains: next,
                videoCount: Math.max(0, item.videoCount + (next ? 1 : -1)),
              }
            : item,
        ) ?? null,
    );

    const result = await toggleVideoInPlaylist(target.id, videoId, target.contains);
    setBusyId(null);

    if (!result.ok) {
      // Put the row back the way it was; the checkbox was a promise we could not keep.
      setTargets(
        (current) =>
          current?.map((item) =>
            item.id === target.id
              ? { ...item, contains: target.contains, videoCount: target.videoCount }
              : item,
          ) ?? null,
      );
      toast.error(result.message);
      return;
    }
    toast.success(next ? `Saved to ${target.title}.` : `Removed from ${target.title}.`);
  }

  function handleCreate(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const title = newTitle.trim();
    if (!title) return;

    startSaveNew(async () => {
      const created = await createPlaylist({ title, visibility: "private" });
      if (!created.ok) {
        toast.error(created.message);
        return;
      }

      const added = await addVideoToPlaylist(created.playlist.id, videoId);
      if (!added.ok) {
        toast.error(added.message);
        return;
      }

      setTargets((current) => [
        {
          id: created.playlist.id,
          title: created.playlist.title,
          videoCount: 1,
          visibility: created.playlist.visibility,
          contains: true,
        },
        ...(current ?? []),
      ]);
      setNewTitle("");
      setCreating(false);
      toast.success(`Saved to ${created.playlist.title}.`);
    });
  }

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger asChild>
        {trigger ?? (
          <Button type="button" variant="outline" size="sm" className="rounded-full">
            <ListPlus aria-hidden />
            Save
          </Button>
        )}
      </PopoverTrigger>

      <PopoverContent align="end" className="w-72 p-0">
        <div className="flex items-center justify-between border-b border-border/60 px-4 py-3">
          <p className="text-sm font-medium">Save to…</p>
          <Button variant="ghost" size="icon-sm" aria-label="Close" onClick={() => setOpen(false)}>
            <X aria-hidden className="size-4" />
          </Button>
        </div>

        <div className="px-2 py-2">
          {loading || !targets ? (
            <div className="flex flex-col gap-3 px-2 py-2">
              {Array.from({ length: 3 }, (_, index) => (
                <div key={index} className="flex items-center gap-3">
                  <Skeleton className="size-4 rounded" />
                  <Skeleton className="h-4 flex-1 rounded-md" />
                </div>
              ))}
            </div>
          ) : targets.length === 0 ? (
            <p className="px-2 py-3 text-sm text-pretty text-muted-foreground">
              You don&apos;t have any playlists yet. Make one below.
            </p>
          ) : (
            <ScrollArea className="max-h-56">
              <ul className="flex flex-col">
                {targets.map((target) => (
                  <SaveTargetRow
                    key={target.id}
                    target={target}
                    busy={busyId === target.id}
                    onToggle={() => void handleToggle(target)}
                  />
                ))}
              </ul>
            </ScrollArea>
          )}
        </div>

        <div className="border-t border-border/60 p-2">
          {creating ? (
            <form onSubmit={handleCreate} className="flex flex-col gap-2 p-1">
              <Label htmlFor="new-playlist-title" className="sr-only">
                Playlist name
              </Label>
              <Input
                id="new-playlist-title"
                value={newTitle}
                onChange={(event) => setNewTitle(event.target.value)}
                maxLength={MAX_PLAYLIST_TITLE}
                placeholder="Playlist name"
                autoComplete="off"
                autoFocus
                disabled={savingNew}
                className="h-8"
              />
              <div className="flex justify-end gap-2">
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  disabled={savingNew}
                  onClick={() => {
                    setCreating(false);
                    setNewTitle("");
                  }}
                >
                  Cancel
                </Button>
                <Button type="submit" size="sm" disabled={savingNew || !newTitle.trim()}>
                  {savingNew ? <LoaderCircle aria-hidden className="animate-spin" /> : null}
                  Create
                </Button>
              </div>
              <p className="text-xs text-muted-foreground">New playlists start out private.</p>
            </form>
          ) : (
            <Button
              variant="ghost"
              size="sm"
              className="w-full justify-start"
              onClick={() => setCreating(true)}
            >
              <Plus aria-hidden />
              New playlist
            </Button>
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
}

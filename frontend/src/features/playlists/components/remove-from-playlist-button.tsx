"use client";

import { LoaderCircle, X } from "lucide-react";
import { useRouter } from "next/navigation";
import { useTransition } from "react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { removeVideoFromPlaylist } from "@/features/playlists/actions";

interface RemoveFromPlaylistButtonProps {
  playlistId: string;
  videoId: string;
  videoTitle: string;
}

/**
 * Removal is addressed by video id, never by position: the API leaves gaps in
 * the sequence when a video is removed and does not renumber, so position is an
 * ordering key, not an index.
 */
export function RemoveFromPlaylistButton({
  playlistId,
  videoId,
  videoTitle,
}: RemoveFromPlaylistButtonProps) {
  const router = useRouter();
  const [pending, startTransition] = useTransition();

  function handle() {
    startTransition(async () => {
      const result = await removeVideoFromPlaylist(playlistId, videoId);
      if (!result.ok) {
        toast.error(result.message);
        return;
      }
      // The row is server-rendered; refresh rather than pretend it is gone.
      router.refresh();
      toast.success(`Removed “${videoTitle}”.`);
    });
  }

  return (
    <Button
      variant="ghost"
      size="icon"
      disabled={pending}
      onClick={handle}
      aria-label={`Remove ${videoTitle} from this playlist`}
      className="size-8 shrink-0 rounded-full text-muted-foreground opacity-0 transition-opacity duration-(--motion-fast) group-hover/row:opacity-100 focus-visible:opacity-100 max-md:opacity-100"
    >
      {pending ? (
        <LoaderCircle aria-hidden className="size-4 animate-spin" />
      ) : (
        <X aria-hidden className="size-4" />
      )}
    </Button>
  );
}

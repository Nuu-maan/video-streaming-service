"use client";

import { Ellipsis, Pencil, Trash2 } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { toast } from "sonner";

import { ConfirmDialog } from "@/components/common/confirm-dialog";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { deletePlaylist } from "@/features/playlists/actions";
import { PlaylistFormDialog } from "@/features/playlists/components/playlist-form-dialog";
import { routes } from "@/config/routes";
import type { Playlist } from "@/types/common";

/** Owner-only controls on the playlist page: rename/re-scope, or delete outright. */
export function PlaylistActions({ playlist }: { playlist: Playlist }) {
  const router = useRouter();
  const [editOpen, setEditOpen] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);

  async function handleDelete() {
    const result = await deletePlaylist(playlist.id);
    if (!result.ok) {
      toast.error(result.message);
      return;
    }
    toast.success(`Deleted “${playlist.title}”.`);
    router.push(routes.playlists);
  }

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="icon" aria-label="Playlist options">
            <Ellipsis aria-hidden />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-40">
          <DropdownMenuItem onSelect={() => setEditOpen(true)}>
            <Pencil aria-hidden />
            Edit
          </DropdownMenuItem>
          <DropdownMenuItem variant="destructive" onSelect={() => setConfirmOpen(true)}>
            <Trash2 aria-hidden />
            Delete
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <PlaylistFormDialog playlist={playlist} open={editOpen} onOpenChange={setEditOpen} />

      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title={`Delete “${playlist.title}”?`}
        description="The playlist goes away. The videos in it stay exactly where they are."
        confirmLabel="Delete playlist"
        destructive
        onConfirm={handleDelete}
      />
    </>
  );
}

"use client";

import { Ellipsis, Flag, Pencil, Trash2 } from "lucide-react";
import { Children } from "react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

/**
 * The per-comment overflow menu: a shell, and the items that may go in it.
 *
 * This replaces a component that took `canEdit`, `canDelete`, `canReport` and
 * `isAuthenticated` — four booleans whose entire job was to pick which menu
 * items to render, plus a `!canEdit && !canDelete && !canReport` early return to
 * catch the case where they picked none. The permission fan-out is a rendering
 * decision the CALLER already knows how to make (CommentItem computes all three),
 * so it should make it; every new capability under the old shape meant another
 * boolean, another branch, and another term in that early return.
 *
 * Composed instead:
 *
 *   <CommentMenu>
 *     {isAuthor  ? <CommentMenu.Edit onSelect={…} />   : null}
 *     {canDelete ? <CommentMenu.Delete onSelect={…} /> : null}
 *     {canReport ? <CommentMenu.Report onSelect={…} /> : null}
 *   </CommentMenu>
 *
 * The shell renders nothing when it has no children, which is the early return
 * for free and one that cannot fall out of sync with the items.
 *
 * The items only ever RAISE INTENT — they call `onSelect` and nothing else. The
 * dialogs they lead to (delete confirmation, report) belong to the caller, and
 * that is not a stylistic preference: a Radix dialog rendered inside
 * DropdownMenuContent unmounts together with the menu the instant the menu
 * closes, which is the instant the item is chosen. It would never be seen.
 */
export function CommentMenu({ children }: { children: React.ReactNode }) {
  // `Children.toArray` drops null/undefined/booleans, so this is exactly "did
  // the caller give me any real items?" — no capabilities, no menu, no trigger.
  if (Children.toArray(children).length === 0) return null;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          aria-label="Comment options"
          className="shrink-0 rounded-full text-muted-foreground opacity-0 transition-opacity duration-(--motion-fast) group-hover/comment:opacity-100 focus-visible:opacity-100 data-[state=open]:opacity-100 max-md:opacity-100"
        >
          <Ellipsis aria-hidden className="size-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-40">
        {children}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function CommentMenuEdit({ onSelect }: { onSelect: () => void }) {
  return (
    <DropdownMenuItem onSelect={onSelect}>
      <Pencil aria-hidden />
      Edit
    </DropdownMenuItem>
  );
}

function CommentMenuDelete({ onSelect }: { onSelect: () => void }) {
  return (
    <DropdownMenuItem variant="destructive" onSelect={onSelect}>
      <Trash2 aria-hidden />
      Delete
    </DropdownMenuItem>
  );
}

function CommentMenuReport({ onSelect }: { onSelect: () => void }) {
  return (
    <DropdownMenuItem onSelect={onSelect}>
      <Flag aria-hidden />
      Report
    </DropdownMenuItem>
  );
}

CommentMenu.Edit = CommentMenuEdit;
CommentMenu.Delete = CommentMenuDelete;
CommentMenu.Report = CommentMenuReport;

"use client";

import { LoaderCircle } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";

interface ConfirmDialogProps {
  /** The element that opens the dialog (rendered via asChild). Omit to control externally. */
  trigger?: React.ReactNode;
  title: string;
  description?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  /** Styles the confirm button for irreversible actions. */
  destructive?: boolean;
  /** Runs on confirm; the dialog shows a pending state and closes when it resolves. */
  onConfirm: () => void | Promise<void>;
  /** Controlled mode. */
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
}

/**
 * Confirmation for genuinely destructive, irreversible actions only —
 * overusing it trains people to click through. Everything else should be
 * undoable instead.
 */
export function ConfirmDialog({
  trigger,
  title,
  description,
  confirmLabel = "Confirm",
  cancelLabel = "Cancel",
  destructive = false,
  onConfirm,
  open,
  onOpenChange,
}: ConfirmDialogProps) {
  const [internalOpen, setInternalOpen] = useState(false);
  const [pending, setPending] = useState(false);
  const isOpen = open ?? internalOpen;
  const setOpen = onOpenChange ?? setInternalOpen;

  async function handleConfirm() {
    setPending(true);
    try {
      await onConfirm();
      setOpen(false);
    } finally {
      setPending(false);
    }
  }

  return (
    <Dialog open={isOpen} onOpenChange={(next) => (pending ? undefined : setOpen(next))}>
      {trigger ? <DialogTrigger asChild>{trigger}</DialogTrigger> : null}
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle className="text-balance">{title}</DialogTitle>
          {description ? (
            <DialogDescription className="text-pretty">{description}</DialogDescription>
          ) : null}
        </DialogHeader>
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline" disabled={pending}>
              {cancelLabel}
            </Button>
          </DialogClose>
          <Button
            variant={destructive ? "destructive" : "default"}
            disabled={pending}
            onClick={handleConfirm}
          >
            {pending ? <LoaderCircle aria-hidden className="animate-spin" /> : null}
            {confirmLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

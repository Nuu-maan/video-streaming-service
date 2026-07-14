"use client";

import { Flag, LoaderCircle } from "lucide-react";
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
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { submitReport } from "@/features/reports/actions";
import { MAX_REPORT_DESCRIPTION, reportReasons, type ReportType } from "@/features/reports/schemas";
import type { ReportTarget } from "@/features/reports/types";
import { useSignInPrompt } from "@/hooks/use-sign-in-prompt";

interface ReportDialogProps {
  target: ReportTarget;
  isAuthenticated: boolean;
  /** Opens the dialog. Omit when driving `open` from a menu item. */
  trigger?: React.ReactNode;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
}

const targetNoun: Record<ReportTarget["kind"], string> = {
  video: "video",
  comment: "comment",
  user: "channel",
};

/**
 * Reporting is a considered act, not a reflex, so it is a dialog with a reason
 * the reporter must choose — never a one-click flag. The description is
 * optional: demanding an essay is how reports go unfiled.
 */
export function ReportDialog({
  target,
  isAuthenticated,
  trigger,
  open,
  onOpenChange,
}: ReportDialogProps) {
  const [internalOpen, setInternalOpen] = useState(false);
  const [reportType, setReportType] = useState<ReportType | "">("");
  const [description, setDescription] = useState("");
  const [pending, startTransition] = useTransition();
  const promptSignIn = useSignInPrompt();

  const isOpen = open ?? internalOpen;
  const setOpen = onOpenChange ?? setInternalOpen;
  const noun = targetNoun[target.kind];

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!reportType) return;

    if (!isAuthenticated) {
      setOpen(false);
      promptSignIn("Sign in to report content.");
      return;
    }

    startTransition(async () => {
      const result = await submitReport({
        target,
        report_type: reportType,
        description: description.trim() || undefined,
      });

      if (result.ok) {
        setOpen(false);
        setReportType("");
        setDescription("");
        toast.success("Report submitted. Thanks — our moderators will take a look.");
        return;
      }
      // A duplicate is not an error the reporter caused; it reads as information.
      if (result.code === "DUPLICATE") {
        setOpen(false);
        toast.info(result.message);
        return;
      }
      toast.error(result.message);
    });
  }

  return (
    <Dialog open={isOpen} onOpenChange={(next) => (pending ? undefined : setOpen(next))}>
      {trigger ? <DialogTrigger asChild>{trigger}</DialogTrigger> : null}
      <DialogContent className="sm:max-w-md">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Flag aria-hidden className="size-4 text-destructive" />
              Report this {noun}
            </DialogTitle>
            <DialogDescription className="text-pretty">
              Tell us what is wrong with this {noun}. Reports are private and reviewed by a
              moderator.
            </DialogDescription>
          </DialogHeader>

          <div className="my-5 flex flex-col gap-4">
            <div className="flex flex-col gap-2">
              <Label htmlFor="report-reason">Reason</Label>
              <Select
                value={reportType}
                onValueChange={(value) => setReportType(value as ReportType)}
                disabled={pending}
              >
                <SelectTrigger id="report-reason" className="w-full">
                  <SelectValue placeholder="Choose a reason" />
                </SelectTrigger>
                <SelectContent>
                  {reportReasons.map((reason) => (
                    <SelectItem key={reason.value} value={reason.value}>
                      {reason.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="flex flex-col gap-2">
              <Label htmlFor="report-description">
                Details <span className="font-normal text-muted-foreground">(optional)</span>
              </Label>
              <Textarea
                id="report-description"
                value={description}
                onChange={(event) => setDescription(event.target.value.slice(0, MAX_REPORT_DESCRIPTION))}
                disabled={pending}
                rows={3}
                placeholder="Anything a moderator should know"
                className="resize-none"
              />
            </div>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              disabled={pending}
              onClick={() => setOpen(false)}
            >
              Cancel
            </Button>
            <Button type="submit" variant="destructive" disabled={pending || !reportType}>
              {pending ? <LoaderCircle aria-hidden className="animate-spin" /> : null}
              Submit report
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

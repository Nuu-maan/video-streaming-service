"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { Ban } from "lucide-react";
import { useId, useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";

import { ConfirmDialog } from "@/components/common/confirm-dialog";
import { Button } from "@/components/ui/button";
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
import { banUser } from "@/features/admin/actions";
import { FieldError } from "@/features/admin/components/field-error";
import { banDurations, banSchema, MAX_BAN_REASON, type BanInput } from "@/features/admin/schemas";

/**
 * Two sentinels the API never sees.
 *
 * `CUSTOM` opens the free-text field. `PERMANENT` exists because the API encodes
 * a permanent ban as an *empty* duration, and an empty string is not a legal
 * Radix `SelectItem` value — so the option carries this instead and it is
 * translated back to `""` before it reaches the schema.
 */
const CUSTOM = "custom";
const PERMANENT = "permanent";

interface BanUserFormProps {
  /** Prefilled from `?user=`, so the reports queue can deep-link a moderator straight here. */
  defaultUserId?: string;
}

/**
 * Ban an account.
 *
 * The duration is a Go duration string, which is a genuinely surprising API: the
 * units stop at hours, so there is no `7d` — seven days is `168h`. The presets
 * exist so that the common cases never require knowing that, and the custom
 * field validates against Go's actual grammar so the moderator finds out here
 * rather than from a 400.
 *
 * Validation runs client-side for the immediate answer and the API validates
 * again regardless — it is the only party that can tell you that you are trying
 * to ban yourself, or that the account does not exist.
 */
export function BanUserForm({ defaultUserId }: BanUserFormProps) {
  const userFieldId = useId();
  const reasonFieldId = useId();
  const customFieldId = useId();

  // `undefined`, not "": an empty string is a *selected* value as far as Radix is
  // concerned, and it would render the trigger as chosen-but-blank instead of
  // showing the placeholder.
  const [preset, setPreset] = useState<string | undefined>(undefined);
  const [confirming, setConfirming] = useState<BanInput | null>(null);

  const {
    register,
    handleSubmit,
    reset,
    setValue,
    formState: { errors },
  } = useForm<BanInput>({
    resolver: zodResolver(banSchema),
    defaultValues: { user_id: defaultUserId ?? "", reason: "", duration: "" },
  });

  function choosePreset(value: string) {
    setPreset(value);
    // A preset writes straight into the field the schema validates. Both
    // sentinels resolve to the empty string: "custom" because the moderator is
    // about to type their own, "permanent" because empty *is* the API's
    // encoding of a permanent ban.
    const duration = value === CUSTOM || value === PERMANENT ? "" : value;
    setValue("duration", duration, { shouldValidate: false });
  }

  async function confirmBan(values: BanInput) {
    const result = await banUser(values.user_id, values.reason, values.duration);
    if (result.ok) {
      toast.success(result.message);
      reset({ user_id: "", reason: "", duration: "" });
      setPreset(undefined);
      return;
    }
    toast.error("Couldn't ban that user", { description: result.message });
  }

  const permanent = !confirming?.duration;

  return (
    <>
      <form onSubmit={handleSubmit((values) => setConfirming(values))} className="grid gap-4" noValidate>
        <div className="grid gap-1.5">
          <Label htmlFor={userFieldId}>User ID</Label>
          <Input
            id={userFieldId}
            {...register("user_id")}
            placeholder="00000000-0000-0000-0000-000000000000"
            className="font-mono text-sm"
            aria-invalid={Boolean(errors.user_id)}
            autoComplete="off"
            spellCheck={false}
          />
          <FieldError message={errors.user_id?.message} />
        </div>

        <div className="grid gap-1.5">
          <Label htmlFor={reasonFieldId}>Reason</Label>
          <Textarea
            id={reasonFieldId}
            {...register("reason")}
            rows={3}
            maxLength={MAX_BAN_REASON}
            placeholder="What they did, and where. This is recorded against the account."
            className="resize-none"
            aria-invalid={Boolean(errors.reason)}
          />
          <FieldError message={errors.reason?.message} />
        </div>

        <div className="grid gap-1.5">
          <Label>Duration</Label>
          <Select value={preset} onValueChange={choosePreset}>
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Choose how long" />
            </SelectTrigger>
            <SelectContent>
              {banDurations.map((duration) => (
                <SelectItem key={duration.label} value={duration.value || PERMANENT}>
                  {duration.label}
                </SelectItem>
              ))}
              <SelectItem value={CUSTOM}>Custom…</SelectItem>
            </SelectContent>
          </Select>

          {preset === CUSTOM ? (
            <div className="mt-1.5 grid gap-1.5">
              <Label htmlFor={customFieldId} className="text-xs text-muted-foreground">
                Go duration — hours are the largest unit, so 7 days is <code>168h</code>
              </Label>
              <Input
                id={customFieldId}
                {...register("duration")}
                placeholder="72h"
                className="font-mono text-sm"
                aria-invalid={Boolean(errors.duration)}
                autoComplete="off"
                spellCheck={false}
              />
            </div>
          ) : (
            <input type="hidden" {...register("duration")} />
          )}
          <FieldError message={errors.duration?.message} />
        </div>

        <Button type="submit" variant="destructive" className="justify-self-start">
          <Ban aria-hidden data-icon="inline-start" />
          Ban user
        </Button>
      </form>

      <ConfirmDialog
        open={confirming !== null}
        onOpenChange={(open) => (open ? undefined : setConfirming(null))}
        title={permanent ? "Ban this account permanently?" : `Ban this account for ${confirming?.duration}?`}
        description={
          confirming
            ? `${confirming.user_id} will be signed out of every session and blocked from signing in${
                permanent ? " indefinitely" : ` for ${confirming.duration}`
              }. The reason is recorded against the account. You can lift a ban afterwards.`
            : undefined
        }
        confirmLabel={permanent ? "Ban permanently" : "Ban"}
        destructive
        onConfirm={async () => {
          if (confirming) await confirmBan(confirming);
        }}
      />
    </>
  );
}

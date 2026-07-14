"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { RotateCcw } from "lucide-react";
import { useId, useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";

import { ConfirmDialog } from "@/components/common/confirm-dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { unbanUser } from "@/features/admin/actions";
import { FieldError } from "@/features/admin/components/field-error";
import { unbanSchema } from "@/features/admin/schemas";

type UnbanInput = { user_id: string };

/**
 * Lift a ban.
 *
 * Restoring someone's access is not destructive, but it is not nothing either —
 * it is the reversal of a moderation decision somebody made on purpose — so it
 * still names what it does before it does it. The dialog just isn't styled red.
 */
export function UnbanUserForm({ defaultUserId }: { defaultUserId?: string }) {
  const fieldId = useId();
  const [confirming, setConfirming] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<UnbanInput>({
    resolver: zodResolver(unbanSchema),
    defaultValues: { user_id: defaultUserId ?? "" },
  });

  async function confirmUnban(userId: string) {
    const result = await unbanUser(userId);
    if (result.ok) {
      toast.success(result.message);
      reset({ user_id: "" });
      return;
    }
    toast.error("Couldn't lift that ban", { description: result.message });
  }

  return (
    <>
      <form
        onSubmit={handleSubmit((values) => setConfirming(values.user_id))}
        className="grid gap-4"
        noValidate
      >
        <div className="grid gap-1.5">
          <Label htmlFor={fieldId}>User ID</Label>
          <Input
            id={fieldId}
            {...register("user_id")}
            placeholder="00000000-0000-0000-0000-000000000000"
            className="font-mono text-sm"
            aria-invalid={Boolean(errors.user_id)}
            autoComplete="off"
            spellCheck={false}
          />
          <FieldError message={errors.user_id?.message} />
        </div>

        <Button type="submit" variant="outline" className="justify-self-start">
          <RotateCcw aria-hidden data-icon="inline-start" />
          Lift ban
        </Button>
      </form>

      <ConfirmDialog
        open={confirming !== null}
        onOpenChange={(open) => (open ? undefined : setConfirming(null))}
        title="Lift this ban?"
        description={
          confirming
            ? `${confirming} will be able to sign in again immediately. The original ban reason stays on the record.`
            : undefined
        }
        confirmLabel="Lift ban"
        onConfirm={async () => {
          if (confirming) await confirmUnban(confirming);
        }}
      />
    </>
  );
}

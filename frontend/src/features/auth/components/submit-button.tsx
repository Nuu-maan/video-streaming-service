"use client";

import { Loader2 } from "lucide-react";

import { Button } from "@/components/ui/button";

/**
 * The one submit button every auth form uses.
 *
 * `pending` comes from the third slot of `useActionState` — the form owns it, so
 * this button does not need `useFormStatus` and does not care that it is not a
 * child of the <form> element in the React tree.
 *
 * While pending it is disabled (a double-submitted login is a wasted request
 * against a 10/min budget) but the label does not move by so much as a pixel.
 * That took an actual fix: the spinner used to be mounted INTO a
 * `justify-center` flex row, so the moment it appeared it inserted a 16px icon
 * plus an 8px gap and shoved the label ~12px to the left — a jitter on the
 * highest-stakes press in the product, in the exact spot the docblock claimed
 * nothing moved. It is now absolutely positioned: out of flow entirely, so the
 * label stays geometrically centred in the button whether it is there or not.
 */
export function SubmitButton({ pending, children }: { pending: boolean; children: React.ReactNode }) {
  return (
    <Button
      type="submit"
      size="lg"
      disabled={pending}
      aria-busy={pending}
      className="mt-1 h-10 w-full disabled:opacity-100"
    >
      {pending ? <Loader2 aria-hidden className="absolute left-3 size-4 animate-spin" /> : null}
      <span className={pending ? "opacity-70" : undefined}>{children}</span>
    </Button>
  );
}

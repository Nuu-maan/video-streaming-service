import { Globe, Link2, Lock } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import type { VideoVisibility } from "@/types/common";

const VISIBILITY = {
  public: { label: "Public", icon: Globe, hint: "Anyone can find and watch it." },
  unlisted: { label: "Unlisted", icon: Link2, hint: "Anyone with the link can watch it." },
  private: { label: "Private", icon: Lock, hint: "Only you can watch it." },
} as const satisfies Record<VideoVisibility, { label: string; icon: typeof Globe; hint: string }>;

/**
 * Who can see this video. Unlike the status badge, this always renders — the
 * whole point of the column is that "public" is a fact worth confirming at a
 * glance, not an absence.
 */
export function VisibilityBadge({ visibility }: { visibility: VideoVisibility }) {
  const { label, icon: Icon, hint } = VISIBILITY[visibility];

  return (
    <Badge variant="outline" className="gap-1.5 border" title={hint}>
      <Icon aria-hidden className="text-muted-foreground" />
      {label}
    </Badge>
  );
}

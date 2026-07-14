import {
  Table,
  TableBody,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { StudioVideoRow } from "@/features/studio/components/studio-video-row";
import type { StudioVideoRow as Row } from "@/features/studio/types";

interface StudioVideoTableProps {
  videos: Row[];
}

/**
 * The creator's library. Every video they own, at every visibility and every
 * point in the lifecycle — this is the only screen in the app that shows a
 * private video or a failed transcode, which is precisely why status and
 * visibility get columns of their own rather than a hover tooltip.
 *
 * The counts are right-aligned and tabular so the digits line up in a column
 * instead of drifting; the table scrolls horizontally inside its own container
 * on narrow screens rather than pushing the page sideways.
 */
export function StudioVideoTable({ videos }: StudioVideoTableProps) {
  return (
    <div className="rounded-xl bg-card shadow-border">
      <Table>
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            <TableHead className="min-w-64">Video</TableHead>
            <TableHead className="w-40">Status</TableHead>
            <TableHead className="w-32">Visibility</TableHead>
            <TableHead className="w-20 text-right">Views</TableHead>
            <TableHead className="w-20 text-right">Likes</TableHead>
            <TableHead className="w-24 text-right">Comments</TableHead>
            <TableHead className="w-32">Uploaded</TableHead>
            {/* The actions column is self-evident from its menu button, but a
                blank <th> is unlabelled to a screen reader. */}
            <TableHead className="w-12">
              <span className="sr-only">Actions</span>
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {videos.map((video) => (
            <StudioVideoRow key={video.id} video={video} />
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

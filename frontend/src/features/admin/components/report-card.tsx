import { MessageSquare, UserRound } from "lucide-react";
import Link from "next/link";
import { Suspense } from "react";

import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { routes } from "@/config/routes";
import { ReportActions } from "@/features/admin/components/report-actions";
import { ReportedVideo } from "@/features/admin/components/reported-video";
import { labelForReportType } from "@/features/reports/schemas";
import { formatDate, formatRelativeTime } from "@/lib/format";
import type { ContentReport } from "@/types/common";

/**
 * One report, with the thing it is about rendered inline.
 *
 * What can be shown inline depends entirely on what the API will hand over. A
 * reported *video* can be fetched and previewed. A reported *comment* or
 * *account* cannot: this API has no `GET /comments/{id}` and no
 * `GET /users/{id}`, so there is no honest way to render their contents here.
 * Rather than fake it, those show the identifier and the reporter's own words —
 * which, for a harassment report, is the substance of the complaint anyway.
 *
 * The video preview sits behind its own Suspense boundary so that ten reports
 * fetch their ten videos concurrently and the queue paints as they arrive,
 * instead of the whole page waiting on the slowest one.
 */
export function ReportCard({ report }: { report: ContentReport }) {
  return (
    <li className="rounded-xl bg-card p-5 shadow-border">
      <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-2">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant="outline" className="border">
              {labelForReportType(report.report_type)}
            </Badge>
            <span className="text-xs text-muted-foreground">
              reported{" "}
              <time dateTime={report.created_at} title={formatDate(report.created_at)}>
                {formatRelativeTime(report.created_at)}
              </time>
            </span>
          </div>

          {/* `reason` is required by the API and is the reporter's headline;
              `description` is the optional detail underneath it. */}
          <p className="mt-2 text-sm font-medium text-pretty">{report.reason}</p>
          {report.description ? (
            <p className="mt-1 text-sm text-pretty text-muted-foreground">{report.description}</p>
          ) : null}
        </div>
      </div>

      <div className="mt-4 space-y-3">
        {report.video_id ? (
          <Suspense fallback={<ReportedVideoSkeleton />}>
            <ReportedVideo videoId={report.video_id} />
          </Suspense>
        ) : null}

        {report.comment_id ? (
          <div className="flex items-center gap-3 rounded-lg bg-muted/40 p-3 text-sm text-muted-foreground">
            <MessageSquare aria-hidden className="size-4 shrink-0" />
            <span>
              A comment was reported. This API has no endpoint to read one back, so only its ID is
              available: <code className="font-mono text-xs">{report.comment_id}</code>
            </span>
          </div>
        ) : null}

        {report.user_id ? (
          <div className="flex flex-wrap items-center gap-x-3 gap-y-2 rounded-lg bg-muted/40 p-3 text-sm text-muted-foreground">
            <UserRound aria-hidden className="size-4 shrink-0" />
            <span>
              Reported account: <code className="font-mono text-xs">{report.user_id}</code>
            </span>
            {/* Deep-links into the ban console with the ID already filled in —
                the alternative is a moderator copying a UUID by hand. */}
            <Link
              href={`${routes.adminUsers}?user=${report.user_id}`}
              className="rounded-sm font-medium text-foreground underline underline-offset-2 outline-none hover:text-brand-700 focus-visible:ring-3 focus-visible:ring-ring/50 dark:hover:text-brand-400"
            >
              Open in ban console
            </Link>
          </div>
        ) : null}
      </div>

      <div className="mt-4 border-t border-border pt-4">
        <ReportActions reportId={report.id} hasVideo={Boolean(report.video_id)} />
      </div>
    </li>
  );
}

function ReportedVideoSkeleton() {
  return (
    <div className="flex items-start gap-3 rounded-lg bg-muted/40 p-3">
      <Skeleton className="aspect-video w-28 shrink-0 rounded-md" />
      <div className="w-full space-y-2">
        <Skeleton className="h-4 w-2/3" />
        <Skeleton className="h-3 w-1/3" />
      </div>
    </div>
  );
}

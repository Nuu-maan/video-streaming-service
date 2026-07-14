import { ShieldCheck } from "lucide-react";
import { Suspense } from "react";

import { EmptyState } from "@/components/common/empty-state";
import { ErrorState } from "@/components/common/error-state";
import { PageHeader } from "@/components/common/page-header";
import { Pagination } from "@/components/common/pagination";
import { getPendingReports } from "@/features/admin/api";
import { ReportCard } from "@/features/admin/components/report-card";
import { errorCopy } from "@/features/admin/error-copy";
import { formatCount } from "@/lib/format";
import type { ContentReport, Page } from "@/types/common";

export const metadata = { title: "Reports" };

/** `?page=` is attacker-writable. Anything that is not a positive integer is page 1. */
function pageNumber(value: string | string[] | undefined): number {
  const raw = Array.isArray(value) ? value[0] : value;
  const parsed = Number(raw);
  return Number.isInteger(parsed) && parsed > 0 ? parsed : 1;
}

export default async function AdminReportsPage(props: PageProps<"/admin/reports">) {
  const searchParams = await props.searchParams;
  const page = pageNumber(searchParams.page);

  let result: Page<ContentReport>;
  try {
    result = await getPendingReports({ page });
  } catch (error) {
    return (
      <>
        <PageHeader title="Reports" description="Content the community has flagged for review." />
        <ErrorState {...errorCopy(error, "the moderation queue", "moderate_content")} />
      </>
    );
  }

  const { items, pagination } = result;

  return (
    <>
      <PageHeader
        title="Reports"
        description={
          pagination.total === 0
            ? "Content the community has flagged for review."
            : `${formatCount(pagination.total, "report")} awaiting review. Oldest first.`
        }
      />

      {items.length === 0 ? (
        <EmptyState
          icon={ShieldCheck}
          title="The queue is clear"
          description="Nothing is waiting for review. New reports land here the moment they're filed."
        />
      ) : (
        <>
          <ul className="flex flex-col gap-4">
            {items.map((report) => (
              <ReportCard key={report.id} report={report} />
            ))}
          </ul>
          <Suspense>
            <Pagination pagination={pagination} />
          </Suspense>
        </>
      )}
    </>
  );
}

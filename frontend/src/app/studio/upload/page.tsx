import type { Metadata } from "next";

import { PageHeader } from "@/components/common/page-header";
import { UploadFlow } from "@/features/upload/components/upload-flow";

export const metadata: Metadata = {
  title: "Upload",
  description: "Upload a video and we'll transcode it for adaptive streaming.",
};

/**
 * A thin route: the entire upload experience is one client component, because
 * the transfer is driven by XMLHttpRequest in the browser (it is the only API
 * that reports request-body progress) and every stage after it is local state.
 */
export default function UploadPage() {
  return (
    <div className="mx-auto w-full max-w-2xl">
      <PageHeader
        title="Upload a video"
        description="MP4, MOV, AVI, MKV or WebM, up to 2 GB. We'll transcode it into 360p through 1080p and stream it adaptively."
        className="mb-8"
      />
      <UploadFlow />
    </div>
  );
}

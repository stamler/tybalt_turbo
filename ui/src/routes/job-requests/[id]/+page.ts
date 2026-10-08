import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
import type { RequestDetail } from "$lib/jobRequests";
export const load: PageLoad = async ({ params }) => {
  const detail = await pb.send<RequestDetail>(`/api/job_requests/${params.id}`, { method: "GET" });
  const [options, duplicates] = await Promise.all([
    detail.permissions.can_edit
      ? pb.send("/api/job_requests/options", { method: "GET" })
      : Promise.resolve({}),
    pb.send(`/api/job_requests/${params.id}/duplicates`, { method: "GET" }),
  ]);
  return { detail, options, duplicates };
};

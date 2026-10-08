import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
import type { RequestDetail } from "$lib/jobRequests";
export const load: PageLoad = async ({ params }) => ({
  detail: await pb.send<RequestDetail>(`/api/job_requests/${params.id}/print`, { method: "GET" }),
  generated: new Date().toLocaleDateString("en-CA"),
});

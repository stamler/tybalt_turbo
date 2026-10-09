import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
import type { RequestDetail } from "$lib/jobRequests";
export const load: PageLoad = async ({ params, url }) => ({
  detail: await pb.send<RequestDetail>(`/api/job_requests/${params.id}/print`, { method: "GET" }),
  generated: new Date().toLocaleDateString("en-CA"),
  // Staff choose on the request page whether the client sees the notes.
  includeNotes: url.searchParams.get("notes") === "1",
});

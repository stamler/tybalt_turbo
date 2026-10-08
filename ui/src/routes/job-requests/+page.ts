import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
export const load: PageLoad = async ({ url }) => {
  const queue = url.searchParams.get("queue") || "mine";
  const page = Math.max(1, Number(url.searchParams.get("page")) || 1);
  const result = await pb.send(
    `/api/job_requests?queue=${encodeURIComponent(queue)}&page=${page}&perPage=50`,
    { method: "GET" },
  );
  return { ...result, queue };
};

import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";

const priorities = new Set(["in_use", "recent", "dormant", "all"]);

function normalizedPriority(raw: string | null) {
  return raw && priorities.has(raw) ? raw : "in_use";
}

function normalizedPage(raw: string | null) {
  const parsed = Number.parseInt(raw ?? "", 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 1;
}

export const load: PageLoad = async ({ url }) => {
  const priority = normalizedPriority(url.searchParams.get("priority"));
  const page = normalizedPage(url.searchParams.get("page"));
  const missing = await pb.send(
    `/api/jobs/project_authorization/missing?priority=${encodeURIComponent(priority)}&page=${page}&limit=50`,
    { method: "GET" },
  );

  return {
    missing,
    priority,
  };
};

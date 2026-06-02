import { pb } from "$lib/pocketbase";

export async function load() {
  const response = await pb.send("/api/jobs/project_authorization/pending", { method: "GET" });
  return {
    items: response.items ?? [],
  };
}

import { pb } from "$lib/pocketbase";
import { pocketBaseFileHref } from "$lib/utilities";

export async function load() {
  const response = await pb.send("/api/jobs/project_authorization/pending", { method: "GET" });
  return {
    items: (response.items ?? []).map((item: any) => ({
      ...item,
      project_authorization_doc_url: item.project_authorization_doc
        ? pocketBaseFileHref("jobs", item.id, item.project_authorization_doc)
        : "",
    })),
  };
}

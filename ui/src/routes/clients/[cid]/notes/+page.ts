import type { PageLoad } from "./$types";
import { loadClientWorkspace } from "$lib/clientWorkspaceLoad";

export const load: PageLoad = (event) => loadClientWorkspace(event, "notes");

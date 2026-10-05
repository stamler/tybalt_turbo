import type { PageLoad } from "./$types";
import { loadClientEditor } from "$lib/clientEditorLoad";
export const load: PageLoad = ({ params, url }) =>
  loadClientEditor(params.cid, url, "invoicing", params.iid);

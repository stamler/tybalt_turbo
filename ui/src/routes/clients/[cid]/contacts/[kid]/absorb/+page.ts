import type { PageLoad } from "./$types";
import { loadClientMerge } from "$lib/clientEditorLoad";
export const load: PageLoad = ({ params, url }) => loadClientMerge(params.cid, params.kid, url);

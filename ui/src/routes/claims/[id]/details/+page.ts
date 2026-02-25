import type { PageLoad } from "./$types";
import { pb } from "$lib/pocketbase";

export interface ClaimHolder {
  id: string;
  admin_profile_id: string;
  given_name: string;
  surname: string;
}

interface ApiClaimHolder {
  admin_profile_id: string;
  given_name: string;
  surname: string;
}

interface ApiClaimDetails {
  id: string;
  name: string;
  description: string;
  holders: ApiClaimHolder[];
}

export interface ClaimDetails {
  id: string;
  name: string;
  description: string;
  holders: ClaimHolder[];
}

export const load: PageLoad = async ({ params }) => {
  try {
    const raw = (await pb.send(`/api/claims/${params.id}`, {
      method: "GET",
    })) as ApiClaimDetails;
    const item: ClaimDetails = {
      ...raw,
      holders: raw.holders.map((h) => ({ ...h, id: h.admin_profile_id })),
    };
    return { item, error: null };
  } catch (error) {
    console.error(`loading claim details: ${error}`);
    return { item: null, error: "You do not have permission to view claim details." };
  }
};

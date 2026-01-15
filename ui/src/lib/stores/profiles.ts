import type { ProfilesResponse } from "$lib/pocketbase-types";
import { createCollectionStore } from "./collectionStore";

export const profiles = createCollectionStore<ProfilesResponse>(
  "profiles",
  {
    sort: "surname,given_name",
    requestKey: "profiles",
  },
  {
    idField: "uid",
    fields: ["surname", "given_name", "uid"],
    storeFields: ["id", "surname", "given_name", "uid"],
    extractField: (doc, field) => (doc as ProfilesResponse)[field as keyof ProfilesResponse],
  },
);

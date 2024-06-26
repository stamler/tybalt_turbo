import type { ProfilesRecord } from "$lib/pocketbase-types";
import { authStore } from "$lib/stores/auth";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";

export const load: PageLoad = async ({ params }) => {
  let uid = "";
  authStore.subscribe((value) => {
    uid = value?.model?.id ?? "";
  });
  const defaultItem = {
    given_name: "",
    surname: "",
    manager: "",
    alternate_manager: "",
    default_division: "",
    uid: uid,
  };
  let item: ProfilesRecord;
  try {
    item = await pb.collection("profiles").getFirstListItem(`uid = '${params.uid}'` );
    return { item, editing: true, id: item.id };
  } catch (error) {
    console.error(`error loading data, returning default item: ${error}`);
    return { item: { ...defaultItem } as ProfilesRecord, editing: false, id: null };
  }
};

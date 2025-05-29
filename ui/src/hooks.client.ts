import { pb } from "$lib/pocketbase";
import { authStore } from "$lib/stores/auth";

// TOOD: consider whether having an authStore is necessary. It appears that
// pb.authStore.<function> may be just as good.

// pb.authStore.loadFromCookie(document.cookie)
pb.authStore.onChange(() => {
  authStore.set(pb.authStore);
  // document.cookie = pb.authStore.exportToCookie({ httpOnly: false })
}, true);

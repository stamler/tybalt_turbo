import { pb } from '$lib/pocketbase'
import { authStore } from '$lib/stores/auth'

pb.authStore.loadFromCookie(document.cookie)
pb.authStore.onChange(() => {
  authStore.set(pb.authStore)
  document.cookie = pb.authStore.exportToCookie({ httpOnly: false })
}, true)
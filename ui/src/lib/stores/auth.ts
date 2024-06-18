import type { BaseAuthStore } from 'pocketbase'
import { writable } from 'svelte/store'
import { pb } from '$lib/pocketbase'

const { subscribe, set, update } = writable<BaseAuthStore | null>(null)


// The loginWithMicrosoft function is now a method of the AuthStore class
async function loginWithMicrosoft () {
  const authData = await pb.collection('users').authWithOAuth2({ provider: 'microsoft' });
  // if the user is logged in, set the authStore
  // console.log(authData.meta)
  if (authData.meta?.authStore.isValid) {
    set(authData.meta.authStore)
  }
}

function logout() {
  update((authStore) => { authStore?.clear(); return null })
}

export const authStore = {
  set,
  update,
  subscribe,
  loginWithMicrosoft,
  logout,
}
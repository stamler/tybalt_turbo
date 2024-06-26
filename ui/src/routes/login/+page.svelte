<script lang="ts">
  import { authStore } from "$lib/stores/auth";
</script>

<!-- The login box, centered vertically and horizontally using flex -->
<div class="flex h-screen flex-col items-center justify-center">
  <div class="flex w-96 flex-col items-center rounded border bg-neutral-100 p-4 align-middle">
    <h1 class="text-4xl font-bold">Welcome to Tybalt</h1>
    <p class="text-lg">Login with Microsoft to get started</p>
    <button
      class="m-2 rounded bg-blue-500 px-4 py-2 font-bold text-white hover:bg-blue-700"
      onclick={authStore.loginWithMicrosoft}
    >
      Login
    </button>
  </div>
</div>

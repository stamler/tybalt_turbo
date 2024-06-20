<script lang="ts">
  import { authStore } from "$lib/stores/auth";
</script>

<!-- The login box, centered vertically and horizontally using flex -->
<div class="flex flex-col justify-center items-center h-screen">
  <div class="flex flex-col items-center align-middle rounded w-96 p-4 bg-neutral-100 border">
    <h1 class="text-4xl font-bold">Welcome to Tybalt</h1>
    <p class="text-lg">Login with Microsoft to get started</p>
    <button
      class="bg-blue-500 hover:bg-blue-700 text-white font-bold m-2 py-2 px-4 rounded"
      onclick={authStore.loginWithMicrosoft}
    >
      Login
    </button>
  </div>
</div>

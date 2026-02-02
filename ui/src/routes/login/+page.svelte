<script lang="ts">
  import { authStore } from "$lib/stores/auth";
  import { page } from "$app/stores";
  import { onMount } from "svelte";

  onMount(() => {
    const redirectUrl = $page.url.searchParams.get("redirect");
    if (redirectUrl) {
      sessionStorage.setItem("redirectUrl", redirectUrl);
    }
  });
</script>

<!-- The login box, top-aligned on mobile and centered on larger screens -->
<div class="flex min-h-screen flex-col items-center justify-start md:justify-center">
  <!-- Logo -->
  <img src="/logo.svg" alt="Tybalt 𝕋𝕌ℝ𝔹𝕆 Logo" class="m-4 h-32 w-32 select-none md:h-40 md:w-40" />
  <div
    class="flex w-full max-w-sm flex-col items-center rounded-sm border bg-neutral-100 p-4 align-middle sm:max-w-md"
  >
    <h1 class="text-4xl font-bold">Welcome to 𝕋𝕌ℝ𝔹𝕆</h1>
    <p class="text-lg">Login with Microsoft to get started</p>
    <button
      class="m-2 rounded-sm bg-blue-500 px-4 py-2 font-bold text-white hover:bg-blue-700"
      onclick={authStore.loginWithMicrosoft}
    >
      Login
    </button>
  </div>
</div>

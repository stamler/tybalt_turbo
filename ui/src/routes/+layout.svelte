<script>
  import "../app.css";
</script>

<header class="w-screen h-10 bg-neutral-700 text-white px-3 flex justify-between">
  <h1 class="h-full flex flex-col justify-center">Tybalt</h1>
  <!-- <template x-if="loggedIn"> -->
    <div class="h-full">
    <!-- user is logged in -->
    <span class="flex items-center h-full gap-2">
      <span>authStore.model.email</span>
      <button class="bg-blue-500 hover:bg-blue-700 text-white text-sm py-1 px-2 rounded" on:click={ logout() }>
        <svg class="w-6 h-6" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
          <path fill-rule="evenodd" d="M16.5 3.75a1.5 1.5 0 0 1 1.5 1.5v13.5a1.5 1.5 0 0 1-1.5 1.5h-6a1.5 1.5 0 0 1-1.5-1.5V15a.75.75 0 0 0-1.5 0v3.75a3 3 0 0 0 3 3h6a3 3 0 0 0 3-3V5.25a3 3 0 0 0-3-3h-6a3 3 0 0 0-3 3V9A.75.75 0 1 0 9 9V5.25a1.5 1.5 0 0 1 1.5-1.5h6ZM5.78 8.47a.75.75 0 0 0-1.06 0l-3 3a.75.75 0 0 0 0 1.06l3 3a.75.75 0 0 0 1.06-1.06l-1.72-1.72H15a.75.75 0 0 0 0-1.5H4.06l1.72-1.72a.75.75 0 0 0 0-1.06Z" clip-rule="evenodd" />
        </svg>    
      </button>
    </span>
  </div>
</header>
<main>
  <slot />
</main>
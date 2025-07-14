import { sveltekit } from "@sveltejs/kit/vite";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [sveltekit()],
  server: {
    host: true, // or '0.0.0.0'
    port: 5173, // optional: specify port
  },
  build: {
    target: "es2020",
  },
});

/** @type {import('tailwindcss').Config} */
export default {
  content: ["./src/**/*.{html,js,svelte,ts}"],
  safelist: [
    {
      // DsLabel border colors
      pattern: /border-(cyan|teal)-300/,
    },
    {
      // DsLabel background colors
      pattern: /bg-(cyan|teal)-100/,
    },
    {
      // DsLabel text colors
      pattern: /text-(cyan|teal)-500/,
    },
    {
      // DsHoverIcon colors
      pattern: /text-(green|red|blue|yellow|purple|orange|gray)-(500|800)/,
      variants: ["hover", "active"],
    },
    "active:shadow-inner",
    "shadow-none",
  ],
  theme: {
    extend: {},
  },
  plugins: [],
};

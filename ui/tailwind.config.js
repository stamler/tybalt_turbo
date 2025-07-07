/** @type {import('tailwindcss').Config} */
export default {
  content: ["./src/**/*.{html,js,svelte,ts}"],
  safelist: [
    {
      // DsLabel border colors
      pattern: /border-(cyan|teal|red|green|orange|yellow|blue|purple|gray)-300/,
    },
    {
      // DsLabel background colors
      pattern: /bg-(cyan|teal|red|green|orange|yellow|blue|purple|gray)-100/,
    },
    {
      // DsLabel text colors
      pattern: /text-(cyan|teal|red|green|orange|yellow|blue|purple|gray)-500/,
    },
    {
      // DsHoverIcon colors
      pattern: /(text|bg)-(cyan|teal|red|green|orange|yellow|blue|purple|gray)-(200|300|500|800)/,
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

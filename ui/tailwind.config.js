/** @type {import('tailwindcss').Config} */
export default {
  content: ["./src/**/*.{html,js,svelte,ts}"],
  safelist: [
    {
      // DsLabel border colors
      pattern: /border-(cyan|teal|red|green)-300/,
    },
    {
      // DsLabel background colors
      pattern: /bg-(cyan|teal|red|green)-100/,
    },
    {
      // DsLabel text colors
      pattern: /text-(cyan|teal|red|green)-500/,
    },
    {
      // DsHoverIcon colors
      pattern: /(text|bg)-(green|red|blue|yellow|purple|orange|gray)-(200|300|500|800)/,
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

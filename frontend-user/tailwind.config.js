/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{vue,ts}"],
  theme: {
    extend: {
      colors: {
        ink: "#07090c",
        panel: "#10151c",
        line: "#1d2836",
        phos: "#9dff6b",
        amber: "#ffb347",
        cyan: "#5ee0ff",
        mute: "#8aa0b5",
      },
      fontFamily: {
        display: ["Oxanium", "sans-serif"],
        mono: ["IBM Plex Mono", "ui-monospace", "monospace"],
      },
    },
  },
  plugins: [],
};

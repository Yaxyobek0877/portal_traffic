import type { Config } from "tailwindcss";

export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        bg: "#0a0e1a",
        bg2: "#0d1322",
        surface: "rgba(255,255,255,0.04)",
        surface2: "rgba(255,255,255,0.06)",
        accent: "#8b5cf6",
        accent2: "#6366f1",
        accent3: "#22d3ee",
      },
      fontFamily: {
        sans: ["Inter", "system-ui", "sans-serif"],
        mono: ["JetBrains Mono", "ui-monospace", "monospace"],
      },
      borderRadius: {
        card: "12px",
        btn: "8px",
        input: "6px",
      },
      backgroundImage: {
        gradient: "linear-gradient(135deg, #8b5cf6 0%, #6366f1 50%, #22d3ee 100%)",
      },
      boxShadow: {
        glow: "0 0 60px -15px rgba(139,92,246,.5)",
        btn: "0 8px 28px -8px rgba(139,92,246,.55)",
      },
    },
  },
  plugins: [],
} satisfies Config;

import { defineConfig, type Plugin } from "vite";
import react from "@vitejs/plugin-react";

// stripCrossOrigin removes the `crossorigin` attribute Vite always
// adds to <script> and <link rel="stylesheet"> tags. Wails serves
// frontend assets through a custom URL scheme (wails://) that the
// browser treats as cross-origin to the document, so the crossorigin
// attribute requires CORS headers Wails doesn't emit. The result on
// the user side is a silently-blank page. Stripping it makes the
// browser load the resources same-origin which is how Wails treats
// them anyway.
function stripCrossOrigin(): Plugin {
  return {
    name: "portal-strip-crossorigin",
    enforce: "post",
    transformIndexHtml(html) {
      return html.replace(/\s+crossorigin/g, "");
    },
  };
}

// `base: "./"` makes asset URLs relative so Wails' embedded asset
// server doesn't get tripped up by leading-slash absolute paths.
// `modulePreload: false` removes the <link rel="modulepreload"> tags
// Vite adds by default.
export default defineConfig({
  plugins: [react(), stripCrossOrigin()],
  base: "./",
  build: {
    outDir: "dist",
    emptyOutDir: true,
    target: "es2020",
    modulePreload: false,
  },
  server: {
    port: 5173,
    strictPort: true,
  },
});

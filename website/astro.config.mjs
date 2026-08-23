import { defineConfig } from "astro/config";

export default defineConfig({
  site: "https://timfewi.github.io",
  base: "/",
  output: "static",
  devToolbar: {
    enabled: false,
  },
  build: {
    format: "directory",
  },
});

import type { Config } from "@react-router/dev/config";

export default {
  // Configure for static export mode
  ssr: false,

  // Build output directory (relative to client/)
  buildDirectory: "../dist",

  // Server build configuration
  serverBuildFile: "index.js",

  // Vite configuration
  vite: {
    server: {
      port: 5173,
      strictPort: true,
    },
  },
} satisfies Config;

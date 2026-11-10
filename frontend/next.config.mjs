import { fileURLToPath } from "node:url";

// Limitamos Next.js a este frontend. Así no confunde otros proyectos cercanos.
const configuracion = {
  agentRules: false,
  // AWS sirve la exportación con Caddy; el desarrollo local conserva Next.js.
  ...(process.env.EXPORTAR_ESTATICO === "1" ? { output: "export" } : {}),
  ...(process.env.DIRECTORIO_NEXT ? { distDir: process.env.DIRECTORIO_NEXT } : {}),
  devIndicators: false,
  turbopack: {
    root: fileURLToPath(new URL(".", import.meta.url)),
  },
};

export default configuracion;

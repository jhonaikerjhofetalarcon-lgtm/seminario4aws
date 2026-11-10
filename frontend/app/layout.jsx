import "./globals.css";

// Next.js utiliza este archivo para dibujar la estructura de todas las páginas.
export const metadata = {
  title: "Aula · Gestión Académica",
  description:
    "Administración de alumnos, matrículas y modalidades de estudio.",
};

export default function EstructuraPrincipal({ children }) {
  return (
    <html lang="es">
      <body>{children}</body>
    </html>
  );
}

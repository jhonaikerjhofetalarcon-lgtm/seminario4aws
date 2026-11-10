// Este componente dibuja iconos SVG. No necesitamos una librería de iconos.
const trazos = {
  alumnos: (
    <>
      <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
      <circle cx="9" cy="7" r="4" />
      <path d="M22 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75" />
    </>
  ),
  libro: (
    <>
      <path d="M4 3h6a3 3 0 0 1 3 3v15a4 4 0 0 0-4-2H4z" />
      <path d="M13 6a3 3 0 0 1 3-3h5v16h-5a3 3 0 0 0-3 2" />
    </>
  ),
  matriculas: (
    <>
      <rect x="5" y="4" width="15" height="17" rx="2" />
      <path d="M9 4V2m7 2V2M9 10h7m-7 4h4m-4 4h7" />
    </>
  ),
  flecha: <path d="M5 12h14m-6-6 6 6-6 6" />,
  buscar: (
    <>
      <circle cx="10" cy="10" r="6" />
      <path d="m15 15 5 5" />
    </>
  ),
  editar: (
    <>
      <path d="m16 3 5 5-12 12H4v-5zM13 6l5 5" />
    </>
  ),
  eliminar: (
    <>
      <path d="M3 6h18M9 6V3h6v3M6 6l1 15h10l1-15M10 10v7m4-7v7" />
    </>
  ),
  mas: <path d="M12 5v14M5 12h14" />,
  cerrar: <path d="m6 6 12 12M6 18 18 6" />,
  verificar: <path d="m5 12 4 4L19 6" />,
};

export default function Icono({ nombre, tamano = 20 }) {
  return (
    <svg
      width={tamano}
      height={tamano}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.7"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {trazos[nombre] || trazos.libro}
    </svg>
  );
}

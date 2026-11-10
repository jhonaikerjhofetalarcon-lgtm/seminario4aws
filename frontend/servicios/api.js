// fetch ya viene en el navegador: no necesitamos instalar Axios u otra librería.
const URL_API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api";

// Esta función reúne la comunicación HTTP para no repetirla en los componentes.
async function solicitar(ruta, metodo = "GET", datos) {
  let respuesta;
  try {
    respuesta = await fetch(`${URL_API}${ruta}`, {
      method: metodo,
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: datos === undefined ? undefined : JSON.stringify(datos),
      cache: "no-store",
    });
  } catch {
    throw new Error(
      "No se pudo conectar con el servicio. Inténtalo nuevamente en unos momentos.",
    );
  }

  if (respuesta.status === 204) return;
  const contenido = await respuesta.json();
  if (!respuesta.ok) {
    const error = new Error(contenido.error || "No se pudo completar la operación.");
    error.estado = respuesta.status;
    if (respuesta.status === 401 && !ruta.startsWith("/auth/") && typeof window !== "undefined") {
      window.dispatchEvent(new Event("aula:sesion-vencida"));
    }
    throw error;
  }
  return contenido;
}

// Este objeto es el cliente HTTP. Los patrones del trabajo están en el backend:
// CRUD 1: aplicacion/fachada_alumnos.go; CRUD 2: elementos_matricula.go + visitante_indicaciones.go.
export const apiAcademica = {
  consultar: () => solicitar("/resumen"),
  registrarAlumno: (datos) => solicitar("/alumnos", "POST", datos),
  editarAlumno: (id, datos) => solicitar(`/alumnos/${id}`, "PUT", datos),
  eliminarAlumno: (id) => solicitar(`/alumnos/${id}`, "DELETE"),
  registrarMatricula: (datos) => solicitar("/matriculas", "POST", datos),
  editarMatricula: (id, datos) => solicitar(`/matriculas/${id}`, "PUT", datos),
  eliminarMatricula: (id) => solicitar(`/matriculas/${id}`, "DELETE"),
};

export const autenticacionAcademica = {
  sesion: () => solicitar("/auth/sesion"),
  entrar: (correo, clave) => solicitar("/auth/login", "POST", { correo, clave }),
  salir: () => solicitar("/auth/logout", "POST"),
};

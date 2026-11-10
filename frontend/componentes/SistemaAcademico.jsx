"use client";

// Los estados conservan los datos de la pantalla; el efecto consulta la API al abrirla.
import { useEffect, useState } from "react";
import { apiAcademica } from "../servicios/api";
import CrudAlumnos from "./CrudAlumnos";
import CrudMatriculas from "./CrudMatriculas";
import Icono from "./Icono";

export default function SistemaAcademico({ usuario, onSalir }) {
  const [vista, establecerVista] = useState("alumnos");
  const [datos, establecerDatos] = useState(null);
  const [cargando, establecerCargando] = useState(true);
  const [ocupado, establecerOcupado] = useState(false);
  const [mensaje, establecerMensaje] = useState(null);
  const [confirmacion, establecerConfirmacion] = useState(null);
  const [revision, establecerRevision] = useState(0);

  useEffect(() => {
    // La limpieza impide actualizar el componente si ya se cerró.
    let activo = true;
    apiAcademica
      .consultar()
      .then((resultado) => {
        if (activo) establecerDatos(resultado);
      })
      .catch((error) => {
        if (activo) establecerMensaje({ tipo: "error", texto: error.message });
      })
      .finally(() => {
        if (activo) establecerCargando(false);
      });
    return () => {
      activo = false;
    };
  }, []);

  async function recargar() {
    establecerCargando(true);
    establecerMensaje(null);
    try {
      establecerDatos(await apiAcademica.consultar());
    } catch (error) {
      establecerMensaje({ tipo: "error", texto: error.message });
    } finally {
      establecerCargando(false);
    }
  }

  async function cerrarSesion() {
    establecerOcupado(true);
    try {
      await onSalir();
    } catch (error) {
      establecerMensaje({ tipo: "error", texto: error.message });
    } finally {
      establecerOcupado(false);
    }
  }

  // Unificamos la espera, los mensajes y la actualización de las listas al guardar.
  async function ejecutar(accion, textoExito) {
    establecerOcupado(true);
    establecerMensaje(null);
    let guardado = false;
    try {
      await accion();
      guardado = true;
      establecerDatos(await apiAcademica.consultar());
      establecerMensaje({ tipo: "exito", texto: textoExito });
      return true;
    } catch (error) {
      if (guardado) {
        establecerDatos(null);
        establecerConfirmacion(null);
      }
      establecerMensaje({
        tipo: "error",
        texto: guardado
          ? `La operación se guardó, pero no pudimos actualizar la lista. ${error.message}`
          : error.message,
      });
      return false;
    } finally {
      establecerOcupado(false);
    }
  }

  function cambiarVista(nuevaVista) {
    establecerVista(nuevaVista);
    establecerConfirmacion(null);
  }

  async function confirmarAccion() {
    const { tipo, registro } = confirmacion;
    const correcto =
      tipo === "alumno"
        ? await ejecutar(
            () => apiAcademica.eliminarAlumno(registro.id),
            "Alumno y matrículas relacionadas eliminados.",
          )
        : await ejecutar(
            () => apiAcademica.eliminarMatricula(registro.id),
            "Matrícula eliminada.",
          );
    if (correcto) {
      establecerConfirmacion(null);
      // La nueva key limpia cualquier formulario que apuntaba al registro eliminado.
      establecerRevision((anterior) => anterior + 1);
    }
  }

  const cantidadRelacionadas =
    confirmacion?.tipo === "alumno"
      ? datos?.matriculas.filter(
          (matricula) => matricula.alumno_id === confirmacion.registro.id,
        ).length
      : 0;

  return (
    <div className="aplicacion">
      <aside className="barra-lateral">
        <a className="marca" href="/" aria-label="Aula, inicio">
          <span className="simbolo-marca">
            <Icono nombre="libro" tamano={25} />
          </span>
          <span>
            aula<span className="punto-marca">.</span>
          </span>
        </a>
        <div className="espacio-instituto">
          <span className="insignia-instituto">
            <Icono nombre="libro" tamano={18} />
          </span>
          <div>
            <strong>Administración</strong>
            <small>Sistema académico</small>
          </div>
        </div>
        <p className="titulo-menu">GESTIÓN ACADÉMICA</p>
        <nav aria-label="Navegación principal">
          <button
            aria-label="Alumnos"
            disabled={ocupado}
            aria-current={vista === "alumnos" ? "page" : undefined}
            className={`opcion-menu ${vista === "alumnos" ? "seleccionada" : ""}`}
            onClick={() => cambiarVista("alumnos")}
          >
            <Icono nombre="alumnos" />
            <span>Alumnos</span>
            <span className="numero-menu">{datos?.alumnos.length ?? "—"}</span>
          </button>
          <button
            aria-label="Matrículas"
            disabled={ocupado}
            aria-current={vista === "matriculas" ? "page" : undefined}
            className={`opcion-menu ${vista === "matriculas" ? "seleccionada" : ""}`}
            onClick={() => cambiarVista("matriculas")}
          >
            <Icono nombre="matriculas" />
            <span>Matrículas</span>
            <span className="numero-menu">
              {datos?.matriculas.length ?? "—"}
            </span>
          </button>
        </nav>
        <div className="pie-barra">
          <span className="avatar-perfil">
            <Icono nombre="libro" tamano={17} />
          </span>
          <div>
            <strong>Administración</strong>
            <small className="correo-perfil" title={usuario.email}>{usuario.email}</small>
          </div>
        </div>
      </aside>

      <div className="contenido-principal">
        <header className="barra-superior">
          <div className="ruta">
            Gestión académica <span>/</span>
            <strong>{vista === "alumnos" ? "Alumnos" : "Matrículas"}</strong>
          </div>
          <div className="acciones-sesion"><span className="estado-local">
            <span className={`punto-verde ${!datos ? "punto-inactivo" : ""}`} />
            {cargando ? "Conectando" : datos ? "Conectado" : "Sin conexión"}
          </span><button className="boton boton-secundario" onClick={cerrarSesion} disabled={ocupado}>Cerrar sesión</button></div>
        </header>
        <main>
          <div className="encabezado-pagina">
            <div>
              <p className="sobre-titulo">ADMINISTRACIÓN ACADÉMICA</p>
              <h1>
                {vista === "alumnos"
                  ? "Gestión de alumnos"
                  : "Gestión de matrículas"}
              </h1>
              <p className="descripcion-pagina">
                {vista === "alumnos"
                  ? "Registra y actualiza la información de los alumnos."
                  : "Administra los cursos y modalidades de cada alumno."}
              </p>
            </div>
            <span className="sello">
              <Icono
                nombre={vista === "alumnos" ? "alumnos" : "matriculas"}
                tamano={27}
              />
            </span>
          </div>

          {mensaje && (
            <div
              className={`mensaje mensaje-${mensaje.tipo}`}
              role={mensaje.tipo === "error" ? "alert" : "status"}
            >
              <span>{mensaje.texto}</span>
              <button
                className="accion"
                onClick={() => establecerMensaje(null)}
                aria-label="Cerrar mensaje"
              >
                <Icono nombre="cerrar" tamano={17} />
              </button>
            </div>
          )}

          <div className="estadisticas">
            <article className="tarjeta-estadistica">
              <span className="icono-estadistica verde">
                <Icono nombre="alumnos" tamano={23} />
              </span>
              <div>
                <p>Alumnos registrados</p>
                <strong>{datos?.alumnos.length ?? "—"}</strong>
                <span>Total de alumnos</span>
              </div>
            </article>
            <article className="tarjeta-estadistica">
              <span className="icono-estadistica durazno">
                <Icono nombre="matriculas" tamano={23} />
              </span>
              <div>
                <p>Matrículas registradas</p>
                <strong>{datos?.matriculas.length ?? "—"}</strong>
                <span>Inscripciones en cursos</span>
              </div>
            </article>
            <article className="tarjeta-estadistica">
              <span className="icono-estadistica lavanda">
                <Icono nombre="libro" tamano={23} />
              </span>
              <div>
                <p>Cursos disponibles</p>
                <strong>{datos?.cursos.length ?? "—"}</strong>
                <span>Oferta académica</span>
              </div>
            </article>
          </div>

          <div className="navegacion-crud">
            <div className="pestanas" aria-label="Secciones de gestión">
              <button
                aria-label="Ver alumnos"
                aria-pressed={vista === "alumnos"}
                disabled={ocupado}
                className={vista === "alumnos" ? "activa" : ""}
                onClick={() => cambiarVista("alumnos")}
              >
                Alumnos {datos && <b>{datos.alumnos.length}</b>}
              </button>
              <button
                aria-label="Ver matrículas"
                aria-pressed={vista === "matriculas"}
                disabled={ocupado}
                className={vista === "matriculas" ? "activa" : ""}
                onClick={() => cambiarVista("matriculas")}
              >
                Matrículas {datos && <b>{datos.matriculas.length}</b>}
              </button>
            </div>
          </div>

          {confirmacion && (
            <section
              className="confirmacion"
              role="alert"
              aria-labelledby="titulo-confirmacion"
            >
              <div>
                <h2 id="titulo-confirmacion">
                  {confirmacion.tipo === "alumno"
                    ? `¿Eliminar a ${confirmacion.registro.nombre}?`
                    : "¿Eliminar esta matrícula?"}
                </h2>
                <p>
                  {confirmacion.tipo === "alumno"
                    ? `También se eliminarán sus ${cantidadRelacionadas} ${cantidadRelacionadas === 1 ? "matrícula" : "matrículas"}. Esta acción no se puede deshacer.`
                    : `Se quitará la matrícula en ${confirmacion.registro.curso}. El alumno seguirá registrado.`}
                </p>
              </div>
              <div className="botones-confirmacion">
                <button
                  className="boton boton-secundario"
                  disabled={ocupado}
                  onClick={() => establecerConfirmacion(null)}
                >
                  Cancelar
                </button>
                <button
                  className="boton boton-peligro"
                  disabled={ocupado}
                  onClick={confirmarAccion}
                >
                  {ocupado ? "Eliminando…" : "Confirmar eliminación"}
                </button>
              </div>
            </section>
          )}

          {cargando ? (
            <div className="panel estado-vacio" role="status">
              <span className="cargando" />
              <h2>Cargando información…</h2>
              <p>Espera un momento.</p>
            </div>
          ) : !datos ? (
            <div className="panel estado-vacio">
              <Icono nombre="libro" tamano={32} />
              <h2>El servicio no está disponible</h2>
              <p>
                No pudimos cargar la información. Intenta conectarte nuevamente.
              </p>
              <button className="boton boton-principal" onClick={recargar}>
                Volver a conectar
              </button>
            </div>
          ) : vista === "alumnos" ? (
            <CrudAlumnos
              key={`alumnos-${revision}`}
              alumnos={datos.alumnos}
              matriculas={datos.matriculas}
              ocupado={ocupado}
              guardar={(id, entrada) =>
                ejecutar(
                  () =>
                    id
                      ? apiAcademica.editarAlumno(id, entrada)
                      : apiAcademica.registrarAlumno(entrada),
                  id
                    ? "Datos del alumno actualizados."
                    : "Alumno registrado correctamente.",
                )
              }
              solicitarEliminar={(registro) =>
                establecerConfirmacion({ tipo: "alumno", registro })
              }
              irAMatriculas={() => cambiarVista("matriculas")}
            />
          ) : (
            <CrudMatriculas
              key={`matriculas-${revision}`}
              alumnos={datos.alumnos}
              matriculas={datos.matriculas}
              cursos={datos.cursos}
              ocupado={ocupado}
              guardar={(id, entrada) =>
                ejecutar(
                  () =>
                    id
                      ? apiAcademica.editarMatricula(id, entrada)
                      : apiAcademica.registrarMatricula(entrada),
                  id
                    ? "Matrícula actualizada."
                    : "Matrícula registrada correctamente.",
                )
              }
              solicitarEliminar={(registro) =>
                establecerConfirmacion({ tipo: "matricula", registro })
              }
              irAAlumnos={() => cambiarVista("alumnos")}
            />
          )}

          <footer className="pie-pagina">
            <span>Aula · Sistema de Gestión Académica</span>
            <span>Alumnos y matrículas</span>
          </footer>
        </main>
      </div>
    </div>
  );
}

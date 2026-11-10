"use client";

import { useRef, useState } from "react";
import Icono from "./Icono";

// CRUD 2 recibe los alumnos del CRUD 1. No guarda una copia del nombre del alumno.
// Visitor se ejecuta en Go y devuelve una indicación según el tipo de matrícula.
export default function CrudMatriculas({
  alumnos,
  matriculas,
  cursos,
  ocupado,
  guardar,
  solicitarEliminar,
  irAAlumnos,
}) {
  const [alumnoId, establecerAlumnoId] = useState("");
  const [curso, establecerCurso] = useState("");
  // Este estado conserva la modalidad que el usuario selecciona en el formulario.
  const [modalidad, establecerModalidad] = useState("Presencial");
  const [idEnEdicion, establecerIdEnEdicion] = useState(null);
  const [busqueda, establecerBusqueda] = useState("");
  const campoAlumno = useRef(null);

  function limpiarFormulario() {
    establecerAlumnoId("");
    establecerCurso("");
    establecerModalidad("Presencial");
    establecerIdEnEdicion(null);
  }

  async function enviarFormulario(evento) {
    evento.preventDefault();
    const correcto = await guardar(idEnEdicion, {
      alumno_id: Number(alumnoId),
      curso,
      modalidad,
    });
    if (correcto) limpiarFormulario();
  }

  function editar(matricula) {
    establecerIdEnEdicion(matricula.id);
    establecerAlumnoId(String(matricula.alumno_id));
    establecerCurso(matricula.curso);
    establecerModalidad(matricula.modalidad || "Presencial");
    campoAlumno.current?.focus();
  }

  // find busca al alumno relacionado. Si editas su nombre, aquí aparece actualizado.
  function nombreDelAlumno(id) {
    return (
      alumnos.find((alumno) => alumno.id === id)?.nombre ||
      "Alumno no disponible"
    );
  }

  const visibles = matriculas.filter((matricula) =>
    `${nombreDelAlumno(matricula.alumno_id)} ${matricula.curso} ${matricula.modalidad} ${matricula.id}`
      .toLocaleLowerCase("es")
      .includes(busqueda.trim().toLocaleLowerCase("es")),
  );

  return (
    <div className="area-crud">
      <section
        className="panel panel-formulario"
        aria-labelledby="titulo-formulario-matricula"
      >
        <h2 id="titulo-formulario-matricula">
          {idEnEdicion ? "Editar matrícula" : "Registrar matrícula"}
        </h2>
        <p className="texto-suave">
          {idEnEdicion
            ? `Estás modificando la matrícula #${idEnEdicion}.`
            : "Selecciona el alumno, el curso y la modalidad."}
        </p>
        {alumnos.length === 0 && (
          <div className="aviso-suave">
            Primero necesitas un alumno.{" "}
            <button className="enlace" onClick={irAAlumnos}>
              Registrar un alumno <Icono nombre="flecha" tamano={14} />
            </button>
          </div>
        )}
        <form onSubmit={enviarFormulario}>
          <fieldset disabled={ocupado || alumnos.length === 0}>
            <label htmlFor="alumno">Alumno registrado</label>
            <select
              ref={campoAlumno}
              id="alumno"
              value={alumnoId}
              onChange={(e) => establecerAlumnoId(e.target.value)}
              required
            >
              <option value="" disabled>
                Selecciona un alumno
              </option>
              {alumnos.map((alumno) => (
                <option key={alumno.id} value={alumno.id}>
                  #{alumno.id} · {alumno.nombre}
                </option>
              ))}
            </select>
            <label htmlFor="curso">Curso</label>
            <select
              id="curso"
              value={curso}
              onChange={(e) => establecerCurso(e.target.value)}
              required
            >
              <option value="" disabled>
                Selecciona un curso
              </option>
              {cursos.map((disponible) => (
                <option key={disponible} value={disponible}>
                  {disponible}
                </option>
              ))}
            </select>
            <label htmlFor="modalidad">Modalidad</label>
            <select
              id="modalidad"
              value={modalidad}
              onChange={(evento) => establecerModalidad(evento.target.value)}
              required
            >
              <option value="Presencial">Presencial</option>
              <option value="Virtual">Virtual</option>
            </select>
            <button className="boton boton-principal ancho" type="submit">
              <Icono nombre={idEnEdicion ? "verificar" : "mas"} />
              {ocupado
                ? "Guardando…"
                : idEnEdicion
                  ? "Guardar cambios"
                  : "Registrar matrícula"}
            </button>
            {idEnEdicion && (
              <button
                className="boton boton-secundario ancho"
                type="button"
                onClick={limpiarFormulario}
              >
                Cancelar edición
              </button>
            )}
          </fieldset>
        </form>
      </section>

      <section
        className="panel panel-lista"
        aria-labelledby="titulo-lista-matriculas"
      >
        <div className="cabecera-lista">
          <div>
            <h2 id="titulo-lista-matriculas">
              Lista de matrículas{" "}
              <span className="contador">{matriculas.length}</span>
            </h2>
            <p className="texto-suave">
              Consulta y administra las inscripciones.
            </p>
          </div>
        </div>
        <div className="buscador">
          <Icono nombre="buscar" tamano={18} />
          <input
            aria-label="Buscar matrícula por alumno, curso o ID"
            placeholder="Buscar por alumno, curso o ID…"
            value={busqueda}
            onChange={(e) => establecerBusqueda(e.target.value)}
          />
        </div>
        <div className="tabla-desplazable">
          <table>
            <thead>
              <tr>
                <th scope="col">ID</th>
                <th scope="col">Alumno</th>
                <th scope="col">Curso y modalidad</th>
                <th scope="col" className="alinear-derecha">
                  Acciones
                </th>
              </tr>
            </thead>
            <tbody>
              {visibles.map((matricula) => (
                <tr key={matricula.id}>
                  <td className="id-registro">
                    #{String(matricula.id).padStart(2, "0")}
                  </td>
                  <td>
                    <div className="persona">
                      <div>
                        <strong>{nombreDelAlumno(matricula.alumno_id)}</strong>
                        <small>Alumno #{matricula.alumno_id}</small>
                      </div>
                    </div>
                  </td>
                  <td>
                    <span className="curso">{matricula.curso}</span>
                    <small className="modalidad-matricula">
                      {matricula.modalidad}
                    </small>
                    {/* Esta indicación viene de Visitor en Go. React solo la muestra. */}
                    <small className="indicacion-visitante">
                      {matricula.indicacion}
                    </small>
                  </td>
                  <td>
                    <div className="acciones">
                      <button
                        disabled={ocupado}
                        className="accion"
                        onClick={() => editar(matricula)}
                        aria-label={`Editar matrícula ${matricula.id} de ${nombreDelAlumno(matricula.alumno_id)}`}
                        title="Editar matrícula"
                      >
                        <Icono nombre="editar" tamano={17} />
                      </button>
                      <button
                        disabled={ocupado}
                        className="accion accion-peligro"
                        onClick={() => solicitarEliminar(matricula)}
                        aria-label={`Eliminar matrícula ${matricula.id} de ${nombreDelAlumno(matricula.alumno_id)}`}
                        title="Eliminar matrícula"
                      >
                        <Icono nombre="eliminar" tamano={17} />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {visibles.length === 0 && (
          <div className="estado-vacio">
            <Icono nombre="matriculas" tamano={32} />
            <h3>
              {matriculas.length
                ? "No encontramos coincidencias"
                : "No hay matrículas registradas"}
            </h3>
            <p>
              {matriculas.length
                ? "Prueba con otro alumno, curso o ID."
                : "Selecciona un alumno y registra su primera matrícula."}
            </p>
          </div>
        )}
        <div className="pie-lista">
          <span>
            {visibles.length} de {matriculas.length} matrículas
          </span>
        </div>
      </section>
    </div>
  );
}

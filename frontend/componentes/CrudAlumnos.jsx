"use client";

// React es una librería para crear interfaces. useState y useRef vienen de React.
import { useRef, useState } from "react";
import Icono from "./Icono";

// Esto es un componente: una función que devuelve una parte de la pantalla (JSX).
// Las props son los datos y funciones que recibimos del componente principal.
// Este formulario envía los datos por HTTP; FachadaAlumnos en Go coordina el CRUD 1.
export default function CrudAlumnos({
  alumnos,
  matriculas,
  ocupado,
  guardar,
  solicitarEliminar,
  irAMatriculas,
}) {
  // Esto es un estado, no una señal de Angular. Al cambiarlo, React actualiza la pantalla.
  const [nombre, establecerNombre] = useState("");
  const [edad, establecerEdad] = useState("");
  const [idEnEdicion, establecerIdEnEdicion] = useState(null);
  const [busqueda, establecerBusqueda] = useState("");
  // Una referencia permite enfocar el campo sin buscarlo manualmente en el documento.
  const campoNombre = useRef(null);

  function limpiarFormulario() {
    establecerNombre("");
    establecerEdad("");
    establecerIdEnEdicion(null);
  }

  async function enviarFormulario(evento) {
    evento.preventDefault(); // Evita que el formulario recargue toda la página.
    const correcto = await guardar(idEnEdicion, { nombre, edad: Number(edad) });
    if (correcto) limpiarFormulario();
  }

  function editar(alumno) {
    establecerIdEnEdicion(alumno.id);
    establecerNombre(alumno.nombre);
    establecerEdad(String(alumno.edad));
    campoNombre.current?.focus();
  }

  // filter crea una lista nueva; no modifica los alumnos guardados.
  const visibles = alumnos.filter((alumno) =>
    `${alumno.nombre} ${alumno.id}`
      .toLocaleLowerCase("es")
      .includes(busqueda.trim().toLocaleLowerCase("es")),
  );

  return (
    <div className="area-crud">
      <section
        className="panel panel-formulario"
        aria-labelledby="titulo-formulario-alumno"
      >
        <h2 id="titulo-formulario-alumno">
          {idEnEdicion ? "Editar alumno" : "Registrar alumno"}
        </h2>
        <p className="texto-suave">
          {idEnEdicion
            ? `Estás modificando al alumno #${idEnEdicion}.`
            : "Completa los datos del alumno."}
        </p>
        <form onSubmit={enviarFormulario}>
          <fieldset disabled={ocupado}>
            <label htmlFor="nombre">Nombre completo</label>
            <input
              ref={campoNombre}
              id="nombre"
              name="nombre"
              autoComplete="name"
              placeholder="Ej. María García"
              value={nombre}
              onChange={(e) => establecerNombre(e.target.value)}
              minLength={3}
              maxLength={100}
              required
            />
            <label htmlFor="edad">Edad</label>
            <div className="entrada-unidad">
              <input
                id="edad"
                name="edad"
                type="number"
                inputMode="numeric"
                placeholder="Ej. 20"
                value={edad}
                onChange={(e) => establecerEdad(e.target.value)}
                min="1"
                max="120"
                step="1"
                required
              />
              <span>años</span>
            </div>
            <button className="boton boton-principal ancho" type="submit">
              <Icono nombre={idEnEdicion ? "verificar" : "mas"} />
              {ocupado
                ? "Guardando…"
                : idEnEdicion
                  ? "Guardar cambios"
                  : "Registrar alumno"}
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
        aria-labelledby="titulo-lista-alumnos"
      >
        <div className="cabecera-lista">
          <div>
            <h2 id="titulo-lista-alumnos">
              Lista de alumnos{" "}
              <span className="contador">{alumnos.length}</span>
            </h2>
            <p className="texto-suave">
              Consulta y administra los alumnos registrados.
            </p>
          </div>
        </div>
        <div className="buscador">
          <Icono nombre="buscar" tamano={18} />
          <input
            aria-label="Buscar alumno por nombre o ID"
            placeholder="Buscar por nombre o ID…"
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
                <th scope="col">Edad</th>
                <th scope="col" className="alinear-derecha">
                  Acciones
                </th>
              </tr>
            </thead>
            <tbody>
              {visibles.map((alumno) => {
                const cantidad = matriculas.filter(
                  (matricula) => matricula.alumno_id === alumno.id,
                ).length;
                return (
                  <tr key={alumno.id}>
                    <td className="id-registro">
                      #{String(alumno.id).padStart(2, "0")}
                    </td>
                    <td>
                      <div className="persona">
                        <span className={`avatar color-${alumno.id % 3}`}>
                          {alumno.nombre
                            .split(" ")
                            .filter(Boolean)
                            .slice(0, 2)
                            .map((palabra) => palabra[0])
                            .join("")}
                        </span>
                        <div>
                          <strong>{alumno.nombre}</strong>
                          <small>
                            {cantidad}{" "}
                            {cantidad === 1 ? "matrícula" : "matrículas"}
                          </small>
                        </div>
                      </div>
                    </td>
                    <td>
                      {alumno.edad} <span className="texto-suave">años</span>
                    </td>
                    <td>
                      <div className="acciones">
                        <button
                          disabled={ocupado}
                          className="accion"
                          onClick={() => editar(alumno)}
                          aria-label={`Editar a ${alumno.nombre}`}
                          title="Editar alumno"
                        >
                          <Icono nombre="editar" tamano={17} />
                        </button>
                        <button
                          disabled={ocupado}
                          className="accion accion-peligro"
                          onClick={() => solicitarEliminar(alumno)}
                          aria-label={`Eliminar a ${alumno.nombre}`}
                          title="Eliminar alumno"
                        >
                          <Icono nombre="eliminar" tamano={17} />
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
        {visibles.length === 0 && (
          <div className="estado-vacio">
            <Icono nombre="alumnos" tamano={32} />
            <h3>
              {alumnos.length
                ? "No encontramos coincidencias"
                : "No hay alumnos registrados"}
            </h3>
            <p>
              {alumnos.length
                ? "Prueba con otro nombre o ID."
                : "Completa el formulario para registrar al primero."}
            </p>
          </div>
        )}
        <div className="pie-lista">
          <span>
            {visibles.length} de {alumnos.length} alumnos
          </span>
          <button className="enlace" disabled={ocupado} onClick={irAMatriculas}>
            Ir a matrículas <Icono nombre="flecha" tamano={16} />
          </button>
        </div>
      </section>
    </div>
  );
}

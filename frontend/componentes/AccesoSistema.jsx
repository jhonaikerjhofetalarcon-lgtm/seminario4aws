"use client";

import { useEffect, useState } from "react";
import { autenticacionAcademica } from "../servicios/api";
import SistemaAcademico from "./SistemaAcademico";
import Icono from "./Icono";

export default function AccesoSistema() {
  const [usuario, establecerUsuario] = useState(null);
  const [verificando, establecerVerificando] = useState(true);
  const [correo, establecerCorreo] = useState("");
  const [clave, establecerClave] = useState("");
  const [mostrarClave, establecerMostrarClave] = useState(false);
  const [enviando, establecerEnviando] = useState(false);
  const [error, establecerError] = useState("");

  useEffect(() => {
    let activo = true;
    autenticacionAcademica.sesion()
      .then(({ usuario: actual }) => { if (activo) establecerUsuario(actual); })
      .catch((fallo) => { if (activo && fallo.estado !== 401) establecerError(fallo.message); })
      .finally(() => { if (activo) establecerVerificando(false); });
    const vencer = () => {
      establecerUsuario(null);
      establecerClave("");
      establecerError("Tu sesión terminó. Inicia sesión para continuar.");
    };
    window.addEventListener("aula:sesion-vencida", vencer);
    return () => {
      activo = false;
      window.removeEventListener("aula:sesion-vencida", vencer);
    };
  }, []);

  async function entrar(evento) {
    evento.preventDefault();
    if (enviando) return;
    establecerEnviando(true);
    establecerError("");
    try {
      const resultado = await autenticacionAcademica.entrar(correo.trim(), clave);
      establecerClave("");
      establecerMostrarClave(false);
      establecerUsuario(resultado.usuario);
    } catch (fallo) {
      establecerError(fallo.message);
    } finally {
      establecerEnviando(false);
    }
  }

  async function salir() {
    await autenticacionAcademica.salir();
    establecerUsuario(null);
    establecerClave("");
    establecerError("");
  }

  if (verificando) {
    return <main className="acceso-verificando" role="status"><span className="cargando" /><p>Verificando tu sesión…</p></main>;
  }
  if (usuario) return <SistemaAcademico usuario={usuario} onSalir={salir} />;

  return (
    <main className="acceso">
      <section className="acceso-presentacion" aria-label="Aula, gestión académica">
        <a className="marca" href="/" aria-label="Aula, inicio">
          <span className="simbolo-marca"><Icono nombre="libro" tamano={25} /></span>
          <span>aula<span className="punto-marca">.</span></span>
        </a>
        <div className="acceso-introduccion">
          <p className="acceso-etiqueta">GESTIÓN ACADÉMICA</p>
          <h1>Más orden.<br /> Más tiempo para enseñar.</h1>
          <p>Un espacio para organizar a tus alumnos y acompañar cada nueva matrícula.</p>
          <div className="acceso-modulos">
            <span><Icono nombre="alumnos" tamano={20} />Alumnos</span>
            <span><Icono nombre="matriculas" tamano={20} />Matrículas</span>
          </div>
        </div>
        <p className="acceso-pie">Todo tu sistema académico, en un solo lugar.</p>
      </section>

      <section className="acceso-formulario" aria-labelledby="titulo-acceso">
        <div className="acceso-tarjeta">
          <span className="acceso-insignia"><Icono nombre="libro" tamano={25} /></span>
          <p className="sobre-titulo">BIENVENIDO A AULA</p>
          <h2 id="titulo-acceso">Inicia sesión</h2>
          <p className="acceso-descripcion">Ingresa con tu cuenta para continuar.</p>

          <form onSubmit={entrar}>
            <fieldset disabled={enviando}>
              <div className="acceso-campo">
                <label htmlFor="correo-acceso">Correo electrónico</label>
                <input id="correo-acceso" name="correo" type="email" autoComplete="username" placeholder="tu.correo@colegio.com" value={correo} onChange={(evento) => establecerCorreo(evento.target.value)} maxLength={254} required />
              </div>
              <div className="acceso-campo">
                <label htmlFor="clave-acceso">Contraseña</label>
                <div className="acceso-clave">
                  <input id="clave-acceso" name="clave" type={mostrarClave ? "text" : "password"} autoComplete="current-password" placeholder="Escribe tu contraseña" value={clave} onChange={(evento) => establecerClave(evento.target.value)} maxLength={1024} required />
                  <button type="button" onClick={() => establecerMostrarClave(!mostrarClave)} aria-pressed={mostrarClave} aria-label={mostrarClave ? "Ocultar contraseña" : "Mostrar contraseña"}>{mostrarClave ? "Ocultar" : "Mostrar"}</button>
                </div>
              </div>
              {error && <p className="acceso-error" role="alert">{error}</p>}
              <button className="boton boton-principal acceso-entrar" type="submit">{enviando ? "Ingresando…" : "Iniciar sesión"}<span aria-hidden="true">→</span></button>
            </fieldset>
          </form>
          <p className="acceso-ayuda">¿Necesitas acceso? Comunícate con la administración.</p>
        </div>
        <p className="acceso-derechos">Aula · Alumnos y matrículas</p>
      </section>
    </main>
  );
}

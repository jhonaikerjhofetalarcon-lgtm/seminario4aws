// Genera un SQL de importación sin enviar datos ni credenciales a ningún servidor.
import { readFileSync, writeFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

const rutaDatos = fileURLToPath(new URL('../backend/datos/academia.json', import.meta.url));
const rutaSalida = fileURLToPath(new URL('./importar_datos_locales.sql', import.meta.url));
const datos = JSON.parse(readFileSync(rutaDatos, 'utf8'));

if (!Array.isArray(datos.alumnos) || !Array.isArray(datos.matriculas)) {
  throw new Error('El archivo local no contiene las dos listas esperadas.');
}

function entero(valor) {
  if (!Number.isSafeInteger(valor) || valor < 1 || valor > 2147483646) {
    throw new Error('Se encontró un ID o contador inválido. No se generó la importación.');
  }
  return valor;
}

function texto(valor) {
  if (typeof valor !== 'string') throw new Error('Se esperaba un texto en los datos.');
  // Las comillas simples se duplican para mantener el texto dentro del literal SQL.
  return "'" + valor.replaceAll("'", "''") + "'";
}

function secuencia(tabla, registros, siguiente) {
  const proximo = entero(Math.max(siguiente || 1, ...registros.map(registro => entero(registro.id) + 1)));
  return `select setval(pg_get_serial_sequence('public.${tabla}', 'id'), ${proximo === 1 ? 1 : proximo - 1}, ${proximo === 1 ? 'false' : 'true'});`;
}

const sentencias = [
  '-- Importación de tus datos locales. Ejecutar DESPUÉS de 01_crear_tablas.sql.',
  '-- Se detiene si las tablas de destino ya contienen registros. No borra datos.',
  'begin;',
  'lock table public.alumnos, public.matriculas in exclusive mode;',
  `do $$ begin
    if exists (select 1 from public.alumnos) or exists (select 1 from public.matriculas) then
      raise exception 'Importación cancelada: las tablas de destino deben estar vacías.';
    end if;
  end $$;`,
];

for (const alumno of datos.alumnos) {
  sentencias.push(`insert into public.alumnos (id, nombre, edad) values (${entero(alumno.id)}, ${texto(alumno.nombre)}, ${entero(alumno.edad)});`);
}
for (const matricula of datos.matriculas) {
  sentencias.push(`insert into public.matriculas (id, alumno_id, curso, modalidad) values (${entero(matricula.id)}, ${entero(matricula.alumno_id)}, ${texto(matricula.curso)}, ${texto(matricula.modalidad || 'Presencial')});`);
}
sentencias.push(secuencia('alumnos', datos.alumnos, datos.siguiente_alumno));
sentencias.push(secuencia('matriculas', datos.matriculas, datos.siguiente_matricula));
sentencias.push('commit;');
writeFileSync(rutaSalida, sentencias.join('\n') + '\n', 'utf8');
console.log('Importación generada: supabase/importar_datos_locales.sql');
console.log(`${datos.alumnos.length} alumnos y ${datos.matriculas.length} matrículas. Revisa el archivo antes de ejecutarlo en SQL Editor.`);

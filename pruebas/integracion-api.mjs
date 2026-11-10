// Ejecutar con el backend encendido: node pruebas/integracion-api.mjs
// No lee claves de Supabase. Inicia sesión como administrador y limpia sus propios datos.
import assert from 'node:assert/strict';
import { randomUUID, createHash } from 'node:crypto';
import { readFile, writeFile } from 'node:fs/promises';

const base = process.env.API_PRUEBAS_URL || 'http://localhost:8080/api';
const rutaJSON = new URL('../backend/datos/academia.json', import.meta.url);
const rutaInforme = new URL('./RESULTADOS.md', import.meta.url);
const marca = `Prueba ${randomUUID()}`;
const nombresPropios = new Set([`${marca} María García`, `${marca} María Torres`]);
const resultados = [];
const idsPropios = new Set();
let inicial, hashInicial, alumno, matricula;
let cookieSesion = '';
let limpieza = 'No fue necesaria.';

async function solicitar(metodo, ruta, datos, estado = 200, crudo = false) {
  const respuesta = await fetch(base + ruta, {
    method: metodo,
    headers: {
      'Content-Type': 'application/json',
      ...(process.env.API_PRUEBAS_AUTH ? { Authorization: process.env.API_PRUEBAS_AUTH } : {}),
      ...(cookieSesion ? { Cookie: cookieSesion } : {}),
    },
    body: datos === undefined ? undefined : crudo ? datos : JSON.stringify(datos),
    signal: AbortSignal.timeout(15000),
  });
  const texto = await respuesta.text();
  assert.equal(respuesta.status, estado, `${metodo} ${ruta}: HTTP esperado ${estado}, recibido ${respuesta.status}`);
  if (estado === 204) {
    assert.equal(texto, '', 'HTTP 204 debe devolver un cuerpo vacío.');
    return;
  }
  assert.match(respuesta.headers.get('content-type') || '', /application\/json/);
  const contenido = JSON.parse(texto);
  if (estado >= 400) assert.equal(typeof contenido.error, 'string');
  return contenido;
}

async function prueba(nombre, ejecutar) {
  const inicio = performance.now();
  try {
    await ejecutar();
    resultados.push({ nombre, estado: 'OK', detalle: '', ms: Math.round(performance.now() - inicio) });
  } catch (error) {
    resultados.push({ nombre, estado: 'FALLO', detalle: error.message, ms: Math.round(performance.now() - inicio) });
  }
  const resultado = resultados.at(-1);
  console.log(`${String(resultados.length).padStart(2, '0')}. ${resultado.estado}: ${nombre}`);
  if (resultado.detalle) console.error(resultado.detalle);
}

async function huellaJSON() {
  return createHash('sha256').update(await readFile(rutaJSON)).digest('hex');
}

function exigirAlumno() {
  assert.ok(alumno?.id && idsPropios.has(alumno.id), 'Se necesita el alumno temporal creado por esta ejecución.');
}

async function comprobarMatricula(esperada) {
  const lista = await solicitar('GET', '/matriculas');
  assert.deepEqual(lista.find(item => item.id === esperada.id), esperada);
}

async function limpiar() {
  const lista = await solicitar('GET', '/alumnos');
  const previos = new Set(inicial.alumnos.map(item => item.id));
  const propios = lista.filter(item => idsPropios.has(item.id) || nombresPropios.has(item.nombre));
  for (const item of propios) {
    assert.ok(!previos.has(item.id), 'La limpieza nunca debe eliminar un alumno preexistente.');
    assert.ok(nombresPropios.has(item.nombre), 'El alumno temporal fue modificado externamente; se detiene su limpieza.');
    await solicitar('DELETE', `/alumnos/${item.id}`, undefined, 204);
  }
  limpieza = 'Registros temporales eliminados. Solo se borraron alumnos propios de esta ejecución y sus matrículas.';
}

// La fotografía inicial permite comprobar que ninguna prueba altera los datos previos.
async function ejecutarPruebas() {
inicial = await solicitar('GET', '/resumen');
hashInicial = await huellaJSON();
try {
  await prueba('Consultar resumen y relaciones existentes', async () => {
    assert.ok(Array.isArray(inicial.alumnos));
    assert.ok(Array.isArray(inicial.matriculas));
    assert.equal(inicial.cursos.length, 4);
    const ids = new Set(inicial.alumnos.map(item => item.id));
    for (const item of inicial.matriculas) {
      assert.ok(ids.has(item.alumno_id), 'La matrícula debe tener un alumno existente.');
      assert.ok(item.indicacion?.length > 0, 'Visitor debe generar la indicación.');
    }
  });

  await prueba('Crear alumno y normalizar espacios y caracteres Unicode', async () => {
    alumno = await solicitar('POST', '/alumnos', { nombre: `  ${marca}   María   García  `, edad: 20 }, 201);
    idsPropios.add(alumno.id);
    assert.ok(Number.isInteger(alumno.id) && alumno.id > 0);
    assert.equal(alumno.nombre, `${marca} María García`);
    const lista = await solicitar('GET', '/alumnos');
    assert.deepEqual(lista.find(item => item.id === alumno.id), alumno);
  });

  await prueba('Editar alumno y consultar los cambios guardados', async () => {
    exigirAlumno();
    alumno = await solicitar('PUT', `/alumnos/${alumno.id}`, { nombre: `${marca} María Torres`, edad: 21 });
    assert.equal(alumno.edad, 21);
    assert.equal(alumno.nombre, `${marca} María Torres`);
    const lista = await solicitar('GET', '/alumnos');
    assert.deepEqual(lista.find(item => item.id === alumno.id), alumno);
  });

  await prueba('Rechazar alumno con nombre vacío (HTTP 400)', async () => {
    await solicitar('POST', '/alumnos', { nombre: '   ', edad: 20 }, 400);
  });

  await prueba('Rechazar alumno con edad fuera de rango (HTTP 400)', async () => {
    await solicitar('POST', '/alumnos', { nombre: `${marca} María García`, edad: 121 }, 400);
  });

  await prueba('Rechazar JSON incompleto (HTTP 400)', async () => {
    await solicitar('POST', '/alumnos', '{"nombre":', 400, true);
  });

  await prueba('Crear matrícula presencial con indicación de Visitor', async () => {
    exigirAlumno();
    matricula = await solicitar('POST', '/matriculas', { alumno_id: alumno.id, curso: 'Desarrollo Web', modalidad: 'Presencial' }, 201);
    assert.equal(matricula.alumno_id, alumno.id);
    assert.equal(matricula.modalidad, 'Presencial');
    assert.equal(matricula.indicacion, 'Asiste al instituto para el curso Desarrollo Web.');
    await comprobarMatricula(matricula);
  });

  await prueba('Editar matrícula a virtual y recalcular Visitor', async () => {
    assert.ok(matricula?.id, 'Se necesita la matrícula temporal.');
    matricula = await solicitar('PUT', `/matriculas/${matricula.id}`, { alumno_id: alumno.id, curso: 'Base de Datos', modalidad: 'Virtual' });
    assert.equal(matricula.curso, 'Base de Datos');
    assert.equal(matricula.modalidad, 'Virtual');
    assert.equal(matricula.indicacion, 'Ingresa al aula virtual para el curso Base de Datos.');
    await comprobarMatricula(matricula);
  });

  await prueba('Rechazar matrícula duplicada (HTTP 409)', async () => {
    assert.ok(matricula?.id, 'Se necesita la matrícula temporal.');
    await solicitar('POST', '/matriculas', { alumno_id: alumno.id, curso: matricula.curso, modalidad: 'Presencial' }, 409);
  });

  await prueba('Rechazar matrícula de alumno inexistente (HTTP 400)', async () => {
    const lista = await solicitar('GET', '/alumnos');
    const ids = new Set(lista.map(item => item.id));
    let inexistente = 2147483647;
    while (ids.has(inexistente)) inexistente--;
    await solicitar('POST', '/matriculas', { alumno_id: inexistente, curso: 'Desarrollo Web', modalidad: 'Virtual' }, 400);
  });

  await prueba('Rechazar curso no permitido (HTTP 400)', async () => {
    exigirAlumno();
    await solicitar('POST', '/matriculas', { alumno_id: alumno.id, curso: 'Curso inexistente', modalidad: 'Virtual' }, 400);
  });

  await prueba('Rechazar modalidad no permitida (HTTP 400)', async () => {
    exigirAlumno();
    await solicitar('POST', '/matriculas', { alumno_id: alumno.id, curso: 'Desarrollo Web', modalidad: 'Híbrida' }, 400);
  });

  await prueba('Eliminar matrícula conservando al alumno (HTTP 204)', async () => {
    assert.ok(matricula?.id, 'Se necesita la matrícula temporal.');
    await solicitar('DELETE', `/matriculas/${matricula.id}`, undefined, 204);
    const lista = await solicitar('GET', '/matriculas');
    assert.ok(!lista.some(item => item.id === matricula.id));
    const alumnos = await solicitar('GET', '/alumnos');
    assert.ok(alumnos.some(item => item.id === alumno.id));
  });

  await prueba('Eliminar alumno y sus dos matrículas en cascada', async () => {
    exigirAlumno();
    await solicitar('POST', '/matriculas', { alumno_id: alumno.id, curso: 'Desarrollo Web', modalidad: 'Virtual' }, 201);
    await solicitar('POST', '/matriculas', { alumno_id: alumno.id, curso: 'Programación I', modalidad: 'Presencial' }, 201);
    const antes = await solicitar('GET', '/matriculas');
    assert.equal(antes.filter(item => item.alumno_id === alumno.id).length, 2);
    await solicitar('DELETE', `/alumnos/${alumno.id}`, undefined, 204);
    const despues = await solicitar('GET', '/resumen');
    assert.ok(!despues.alumnos.some(item => item.id === alumno.id));
    assert.ok(!despues.matriculas.some(item => item.alumno_id === alumno.id));
  });
} finally {
  try { await limpiar(); }
  catch (error) { limpieza = `FALLO DE LIMPIEZA: ${error.message}`; }
}

await prueba('Conservar datos originales, JSON local y ausencia de temporales', async () => {
  assert.ok(!limpieza.startsWith('FALLO'), limpieza);
  const final = await solicitar('GET', '/resumen');
  for (const tabla of ['alumnos', 'matriculas']) {
    for (const previo of inicial[tabla]) {
      assert.deepEqual(final[tabla].find(item => item.id === previo.id), previo, `Cambió un registro previo de ${tabla}.`);
    }
  }
  assert.ok(!final.alumnos.some(item => idsPropios.has(item.id) || nombresPropios.has(item.nombre)));
  assert.ok(!final.matriculas.some(item => idsPropios.has(item.alumno_id)));
  assert.equal(await huellaJSON(), hashInicial, 'El JSON local debe permanecer intacto.');
});

const aprobadas = resultados.filter(item => item.estado === 'OK').length;
const fecha = new Intl.DateTimeFormat('es-PE', { dateStyle: 'long', timeStyle: 'medium', timeZone: 'America/Lima' }).format(new Date());
const escapar = texto => texto.replaceAll('|', '\\|').replaceAll('\n', ' ');
const informe = [
  '# Resultados de las 15 pruebas de integración', '',
  `Fecha: ${fecha} (America/Lima).`, '',
  `Resultado: **${aprobadas}/${resultados.length} aprobadas**. API: \`${base}\`.`, '',
  'Peticiones HTTP reales con sesión de administrador al backend conectado a Supabase; no se automatizaron clics en el navegador. No se leyeron claves privadas de Supabase.', '',
  '| N.º | Prueba | Resultado | Tiempo |', '| --- | --- | --- | --- |',
  ...resultados.map((item, indice) => `| ${indice + 1} | ${escapar(item.nombre)} | ${item.estado} | ${item.ms} ms |`), '',
  `Datos iniciales: ${inicial.alumnos.length} alumnos y ${inicial.matriculas.length} matrículas.`, '',
  limpieza, '',
  'Las pruebas consumen IDs de PostgreSQL. Los saltos después de eliminar los temporales son normales.', '',
  'Para repetir desde la carpeta `seminario 4 react`, con Go encendido:', '',
  '```powershell', 'node pruebas/integracion-api.mjs', '```', '',
  ...resultados.filter(item => item.detalle).map(item => `- ${escapar(item.nombre)}: ${escapar(item.detalle)}`),
].join('\n');
await writeFile(rutaInforme, informe + '\n', 'utf8');
console.log(`\n${aprobadas}/${resultados.length} aprobadas. Informe: pruebas/RESULTADOS.md`);
if (aprobadas !== 15 || resultados.length !== 15) process.exitCode = 1;
}

try {
  const texto = await readFile(new URL('../backend/ACCESO-ADMIN.txt', import.meta.url), 'utf8');
  const credenciales = Object.fromEntries(texto.split(/\r?\n/).filter(linea => /^(Correo|Clave)=/.test(linea)).map(linea => [linea.slice(0, linea.indexOf('=')), linea.slice(linea.indexOf('=') + 1)]));
  const login = await fetch(base + '/auth/login', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ correo: credenciales.Correo, clave: credenciales.Clave }), signal: AbortSignal.timeout(15000) });
  assert.equal(login.status, 200, 'El administrador de pruebas no pudo iniciar sesión.');
  cookieSesion = login.headers.get('set-cookie').split(';')[0];
  await ejecutarPruebas();
} finally {
  if (cookieSesion) await solicitar('POST', '/auth/logout', undefined, 204);
}

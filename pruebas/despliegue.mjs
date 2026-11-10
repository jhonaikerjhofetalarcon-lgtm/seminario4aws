// Comprueba el login real y la protección de la API. No modifica datos académicos.
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';

const base = process.env.URL_PRUEBAS_AWS || 'http://localhost:8080';
const texto = await readFile(new URL('../backend/ACCESO-ADMIN.txt', import.meta.url), 'utf8');
const cred = Object.fromEntries(texto.split(/\r?\n/).filter(linea => /^(Correo|Clave)=/.test(linea)).map(linea => [linea.slice(0, linea.indexOf('=')), linea.slice(linea.indexOf('=') + 1)]));
const origen = process.env.ORIGEN_PRUEBAS || 'http://localhost:3000';
const pedir = (ruta, opciones = {}) => fetch(base + ruta, { signal: AbortSignal.timeout(20000), ...opciones });
const resultados = [];
let cookie = '', entrada;
const json = datos => ({ method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(datos) });
async function caso(nombre, verificar) {
  try { await verificar(); resultados.push(true); console.log(`OK: ${nombre}`); }
  catch (error) { resultados.push(false); console.error(`FALLO: ${nombre}: ${error.message}`); }
}

try {
  await caso('La API rechaza consultas sin sesión', async () => {
    for (const ruta of ['/api/resumen', '/api/alumnos', '/api/matriculas']) assert.equal((await pedir(ruta)).status, 401);
  });
  await caso('La API rechaza escrituras sin sesión', async () => assert.equal((await pedir('/api/alumnos', json({}))).status, 401));
  await caso('Consultar una sesión inexistente devuelve 401', async () => assert.equal((await pedir('/api/auth/sesion')).status, 401));
  await caso('La contraseña incorrecta se rechaza', async () => assert.equal((await pedir('/api/auth/login', json({ correo: cred.Correo, clave: 'incorrecta-solo-prueba' }))).status, 401));
  await caso('El login rechaza JSON incompleto', async () => assert.equal((await pedir('/api/auth/login', { ...json({}), body: '{"correo":' })).status, 400));
  await caso('El login bloquea un origen externo', async () => {
    const opciones = json({ correo: cred.Correo, clave: cred.Clave }); opciones.headers.Origin = 'https://ajeno.test';
    assert.equal((await pedir('/api/auth/login', opciones)).status, 403);
  });
  await caso('CORS admite cookies solo para el frontend autorizado', async () => {
    const r = await pedir('/api/auth/login', { method: 'OPTIONS', headers: { Origin: origen } });
    assert.equal(r.status, 204); assert.equal(r.headers.get('access-control-allow-origin'), origen); assert.equal(r.headers.get('access-control-allow-credentials'), 'true');
  });
  await caso('El administrador inicia sesión con Supabase Auth', async () => {
    entrada = await pedir('/api/auth/login', json({ correo: cred.Correo, clave: cred.Clave }));
    assert.equal(entrada.status, 200);
    cookie = entrada.headers.get('set-cookie').split(';')[0];
  });
  await caso('La cookie es HttpOnly y SameSite=Lax', async () => {
    assert.ok(entrada); const cabecera = entrada.headers.get('set-cookie');
    assert.match(cabecera, /HttpOnly/i); assert.match(cabecera, /SameSite=Lax/i); assert.match(cabecera, /Path=\/api/i);
    if (origen.startsWith('https://')) assert.match(cabecera, /Secure/i);
  });
  await caso('La respuesta no expone tokens ni contraseña', async () => {
    const datos = await entrada.json();
    assert.deepEqual(Object.keys(datos), ['usuario']); assert.equal(datos.usuario.email, cred.Correo);
    assert.ok(!JSON.stringify(datos).includes(cred.Clave));
  });
  await caso('La sesión conserva la identidad del administrador', async () => {
    const r = await pedir('/api/auth/sesion', { headers: { Cookie: cookie } });
    assert.equal(r.status, 200); assert.equal((await r.json()).usuario.email, cred.Correo); assert.equal(r.headers.get('cache-control'), 'no-store');
  });
  await caso('La sesión permite consultar alumnos y matrículas', async () => {
    const r = await pedir('/api/resumen', { headers: { Cookie: cookie } }); assert.equal(r.status, 200);
    const datos = await r.json(); assert.ok(Array.isArray(datos.alumnos) && Array.isArray(datos.matriculas));
  });
  await caso('Una cookie falsificada no concede acceso', async () => assert.equal((await pedir('/api/resumen', { headers: { Cookie: 'aula_sesion=' + 'x'.repeat(43) } })).status, 401));
  await caso('Una sesión válida no permite escrituras desde otro sitio', async () => {
    const opciones = json({ nombre: 'No debe crearse', edad: 20 }); opciones.headers.Cookie = cookie; opciones.headers.Origin = 'https://ajeno.test';
    assert.equal((await pedir('/api/alumnos', opciones)).status, 403);
  });
  await caso('El cierre invalida la sesión, incluso si se copia la cookie', async () => {
    assert.equal((await pedir('/api/auth/logout', { method: 'POST', headers: { Cookie: cookie } })).status, 204);
    assert.equal((await pedir('/api/resumen', { headers: { Cookie: cookie } })).status, 401);
  });
} finally {
  if (cookie) await pedir('/api/auth/logout', { method: 'POST', headers: { Cookie: cookie } });
}
console.log(`\n${resultados.filter(Boolean).length}/${resultados.length} comprobaciones de login y despliegue aprobadas.`);
if (resultados.some(resultado => !resultado)) process.exitCode = 1;

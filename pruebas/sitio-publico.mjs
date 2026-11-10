// Verifica el enlace real, la entrega de React y la redireccion HTTPS.
import assert from 'node:assert/strict';

const origen = new URL(process.env.URL_PRUEBAS_AWS || 'https://d3grale84xd0q8.cloudfront.net');
assert.equal(origen.protocol, 'https:');
const pedir = (url, opciones = {}) => fetch(url, { signal: AbortSignal.timeout(20000), ...opciones });
const respuesta = await pedir(origen);
assert.equal(respuesta.status, 200);
assert.match(respuesta.headers.get('content-type'), /text\/html/i);
const html = await respuesta.text();
const archivos = [...new Set([...html.matchAll(/(?:src|href)="([^"<>]+\.(?:js|css)(?:\?[^"<>]*)?)"/g)].map(r => r[1]))];
assert.ok(archivos.some(archivo => archivo.includes('.js')), 'Faltan los archivos de React.');
assert.ok(archivos.some(archivo => archivo.includes('.css')), 'Faltan los estilos.');
await Promise.all(archivos.map(async archivo => {
  const url = new URL(archivo.replaceAll('&amp;', '&'), origen);
  assert.equal(url.origin, origen.origin, 'Un recurso se sirve desde un origen inesperado.');
  const r = await pedir(url);
  assert.equal(r.status, 200, `No se pudo cargar ${url.pathname}`);
  assert.match(r.headers.get('content-type'), url.pathname.endsWith('.css') ? /text\/css/i : /javascript/i);
  assert.ok((await r.arrayBuffer()).byteLength > 0);
}));
console.log(`OK: pagina HTTPS y ${archivos.length} archivos JavaScript/CSS disponibles.`);
const salud = await pedir(new URL('/healthz', origen));
assert.equal(salud.status, 200);
assert.deepEqual(await salud.json(), { estado: 'ok' });
console.log('OK: servicio de salud disponible.');
const http = new URL(origen);
http.protocol = 'http:';
const redireccion = await pedir(http, { redirect: 'manual' });
assert.ok([301, 302, 307, 308].includes(redireccion.status));
assert.equal(new URL(redireccion.headers.get('location')).origin, origen.origin);
console.log('OK: HTTP redirige al enlace HTTPS.');

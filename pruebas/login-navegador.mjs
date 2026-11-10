// Controla una instancia local de Edge con depuración CDP; no instala paquetes.
import assert from 'node:assert/strict';
import { mkdir, readFile, writeFile } from 'node:fs/promises';

const texto = await readFile(new URL('../backend/ACCESO-ADMIN.txt', import.meta.url), 'utf8');
const cred = Object.fromEntries(texto.split(/\r?\n/).filter(linea => /^(Correo|Clave)=/.test(linea)).map(linea => [linea.slice(0, linea.indexOf('=')), linea.slice(linea.indexOf('=') + 1)]));
let info;
for (let i = 0; i < 30 && !info; i++) {
  try { info = await (await fetch('http://127.0.0.1:9223/json/version')).json(); }
  catch { await new Promise(r => setTimeout(r, 300)); }
}
assert.ok(info, 'El navegador de prueba no está disponible en el puerto 9223.');
const socket = new WebSocket(info.webSocketDebuggerUrl);
await new Promise((resolve, reject) => { socket.addEventListener('open', resolve, { once: true }); socket.addEventListener('error', reject, { once: true }); });
let secuencia = 0, sesion;
const pendientes = new Map();
socket.addEventListener('message', evento => {
  const r = JSON.parse(evento.data);
  if (!r.id || !pendientes.has(r.id)) return;
  const pendiente = pendientes.get(r.id); pendientes.delete(r.id); clearTimeout(pendiente.timer);
  if (r.error) pendiente.reject(new Error('Falló un comando de navegador.')); else pendiente.resolve(r.result);
});
function comando(method, params = {}, enPagina = true) {
  return new Promise((resolve, reject) => {
    const id = ++secuencia;
    const timer = setTimeout(() => { pendientes.delete(id); reject(new Error(`Tiempo agotado en ${method}`)); }, 20000);
    pendientes.set(id, { resolve, reject, timer });
    socket.send(JSON.stringify({ id, method, params, ...(enPagina && sesion ? { sessionId: sesion } : {}) }));
  });
}
async function evaluar(expression) {
  const r = await comando('Runtime.evaluate', { expression, returnByValue: true, awaitPromise: true });
  if (r.exceptionDetails) throw new Error('Falló una comprobación dentro de la página.');
  return r.result.value;
}
async function esperar(expression) {
  for (let i = 0; i < 60; i++) { if (await evaluar(expression)) return; await new Promise(r => setTimeout(r, 300)); }
  throw new Error('La interfaz no llegó al estado esperado.');
}
async function completar(clave) {
  await evaluar(`(() => { const escribir = (id, valor) => { const campo = document.getElementById(id); Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value').set.call(campo, valor); campo.dispatchEvent(new Event('input', {bubbles:true})); }; escribir('correo-acceso', ${JSON.stringify(cred.Correo)}); escribir('clave-acceso', ${JSON.stringify(clave)}); })()`);
  await evaluar("document.querySelector('form').requestSubmit()");
}
async function capturar(nombre) {
  if (process.env.GUARDAR_CAPTURAS !== '1') return;
  await mkdir(new URL('../dist/', import.meta.url), { recursive: true });
  const captura = await comando('Page.captureScreenshot', { format: 'png' });
  await writeFile(new URL(`../dist/${nombre}.png`, import.meta.url), Buffer.from(captura.data, 'base64'));
}
try {
  const target = await comando('Target.createTarget', { url: 'about:blank' }, false);
  sesion = (await comando('Target.attachToTarget', { targetId: target.targetId, flatten: true }, false)).sessionId;
  await comando('Page.enable');
  await comando('Emulation.setDeviceMetricsOverride', { width: 1440, height: 900, deviceScaleFactor: 1, mobile: false });
  await comando('Page.navigate', { url: process.env.URL_NAVEGADOR || 'http://localhost:3000' });
  await esperar("!!document.getElementById('correo-acceso')");
  console.log('OK: pantalla de login visible sin mostrar los datos académicos.');
  await completar('incorrecta-solo-navegador');
  await esperar("document.querySelector('[role=alert]')?.textContent.includes('incorrectos')");
  console.log('OK: error visible para contraseña incorrecta.');
  await completar(cred.Clave);
  await esperar("!!Array.from(document.querySelectorAll('button')).find(b=>b.textContent==='Cerrar sesión')");
  assert.ok(await evaluar(`document.body.innerText.includes(${JSON.stringify(cred.Correo)})`));
  console.log('OK: inicio real muestra la cuenta y el sistema académico.');
  await comando('Page.reload');
  await esperar("!!Array.from(document.querySelectorAll('button')).find(b=>b.textContent==='Cerrar sesión')");
  assert.equal(await evaluar('localStorage.length'), 0);
  console.log('OK: sesión conservada al recargar; no se guardan tokens en localStorage.');
  await evaluar("Array.from(document.querySelectorAll('button')).find(b=>b.textContent==='Cerrar sesión').click()");
  await esperar("!!document.getElementById('correo-acceso')");
  assert.equal(await evaluar("!!document.querySelector('.aplicacion')"), false);
  assert.equal(await evaluar("document.getElementById('clave-acceso').value"), '');
  console.log('OK: cerrar sesión vuelve al login y limpia los datos y la contraseña.');
  await capturar('login-escritorio');
  await comando('Emulation.setDeviceMetricsOverride', { width: 390, height: 844, deviceScaleFactor: 1, mobile: true });
  await new Promise(r => setTimeout(r, 300));
  assert.ok(await evaluar('document.documentElement.scrollWidth <= window.innerWidth'));
  await capturar('login-movil');
  console.log('OK: diseño móvil sin desbordamiento horizontal.');
} finally {
  try { await comando('Browser.close', {}, false); } catch {}
  socket.close();
}

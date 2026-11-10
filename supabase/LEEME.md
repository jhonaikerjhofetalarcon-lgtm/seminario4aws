# Activar Supabase

**Conexión activa y verificada al 9 de noviembre de 2026.** El MCP autorizado creó el esquema en el proyecto `pgbnustgxnfqqwbucili` e importó los 2 alumnos y las 2 matrículas del JSON local. Go está configurado con `ALMACENAMIENTO=supabase` y la clave privada de `backend/.env`. RLS está activa y los roles `anon` y `authenticated` no tienen acceso a las tablas ni a la función.

Se probaron creación, consulta, edición y eliminación mediante la API de Go y se confirmó la persistencia directamente en Supabase mediante MCP. También funcionaron Visitor, el rechazo de duplicados, la eliminación en cascada y CORS. Se eliminaron los registros temporales; quedaron los 2 alumnos y las 2 matrículas originales. El JSON local no cambió.

El backend responde en `http://localhost:8080` y el frontend en `http://localhost:3000`. Los mensajes de Go se consultan en la terminal que ejecuta `go run .`. Si reinicias el equipo, inicia ambos servidores según el README.

Los pasos siguientes quedan como referencia para configurar otro equipo o proyecto. No repitas la importación en este proyecto: las tablas ya contienen registros. Tampoco sobrescribas el `.env` configurado.

## 1. Crear el proyecto

Entra en [Supabase](https://supabase.com/dashboard/new), inicia sesión y crea un proyecto. Espera a que termine de prepararse. Puedes usar un proyecto nuevo dedicado a este sistema académico.

## 2. Crear las tablas

En el proyecto, abre **SQL Editor**, crea una consulta y ejecuta el contenido completo de [01_crear_tablas.sql](01_crear_tablas.sql).

Este script crea:

- `alumnos`: ID, nombre y edad.
- `matriculas`: ID, referencia al alumno, curso y modalidad.
- La relación `ON DELETE CASCADE` y la restricción de matrícula única por alumno y curso.
- La función de consulta `consultar_sistema_academico`.
- RLS activa y acceso restringido al backend con la clave secreta.

Si vuelves a ejecutar el script en el mismo esquema compatible, conserva los registros. No adapta tablas de otro sistema que ya usen esos nombres con una estructura diferente.

## 3. Conservar los registros locales (opcional)

Desde la carpeta `seminario 4 react` ejecuta:

```powershell
node supabase/exportar_datos_locales.mjs
```

Se genera `supabase/importar_datos_locales.sql` con el contenido actual de tu archivo local. Revisa el archivo y ejecútalo en SQL Editor, después del script de tablas.

La importación comprueba que ambas tablas estén vacías y se detiene si contienen registros. Conserva los IDs, las relaciones y los contadores locales; convierte las modalidades antiguas vacías en Presencial. No modifica ni borra el JSON del equipo. Si cambian tus datos locales antes de importar, vuelve a generar el SQL.

Este archivo contiene datos de alumnos y está excluido de Git. No forma parte del esquema público del proyecto.

## 4. Configurar Go

Desde `seminario 4 react/backend`:

```powershell
# Solo si todavía no existe .env; evita sobrescribir una configuración existente.
if (-not (Test-Path .env)) { Copy-Item .env.example .env }
```

Abre `backend/.env` y reemplaza los valores de ejemplo:

```dotenv
ALMACENAMIENTO=supabase
SUPABASE_URL=https://TU-PROYECTO.supabase.co
SUPABASE_SECRET_KEY=sb_secret_REEMPLAZAR
SUPABASE_PUBLISHABLE_KEY=sb_publishable_REEMPLAZAR
AUTH_USUARIOS_PERMITIDOS=UUID_DEL_ADMINISTRADOR
ORIGEN_FRONTEND=http://localhost:3000
```

La URL está en la información de conexión de tu proyecto. La clave secreta se obtiene en **Settings → API Keys**; usa una clave `sb_secret_...`. Si tu proyecto todavía utiliza claves antiguas, puedes definir `SUPABASE_SERVICE_ROLE_KEY` con la clave `service_role` en lugar de `SUPABASE_SECRET_KEY`. No uses la clave pública `anon` ni `sb_publishable_...` para esta integración.

La clave se guarda únicamente en `backend/.env`, que está excluido de Git. No se necesita ninguna clave de Supabase en React. El frontend continúa usando `NEXT_PUBLIC_API_URL` para hablar con Go.

El login también requiere la clave pública del proyecto y el ID autorizado de Supabase Auth. En este equipo ya se creó `admin@colegio.com` y se completó esa configuración. Las credenciales iniciales están en `backend/ACCESO-ADMIN.txt`. El comando `supabase/crear-administrador.ps1` permite preparar otra cuenta con una clave pública indicada, sin reemplazar automáticamente contraseñas existentes.

## 5. Reiniciar el backend

Detén el proceso anterior para liberar el puerto 8080. Dentro de `backend`:

```powershell
go build -o academia.exe .
.\academia.exe
```

Go verifica el acceso y la función SQL antes de iniciar el servidor. Si todo funciona, la terminal mostrará:

```text
Almacenamiento activo: supabase
Backend académico disponible en http://localhost:8080/api/resumen
```

Si falta configuración, hay un error de credenciales o las tablas no están listas, el backend se detiene y comunica el problema. **No cambia silenciosamente a JSON.**

## 6. Verificar que se guarda en Supabase

Abre la aplicación en `http://localhost:3000`, registra un alumno y comprueba que aparezca en **Table Editor → alumnos** del mismo proyecto. Después registra una matrícula y compruébala en `matriculas`.

El texto de asistencia no es una columna de la base de datos: Visitor lo genera en Go al responder. Facade y Visitor siguen implementados en los mismos archivos.

## Volver al archivo local

Configura `ALMACENAMIENTO=archivo` en `backend/.env` y reinicia Go. Los datos de Supabase y del JSON son almacenamientos independientes; no se sincronizan automáticamente. Los datos del archivo original se conservan.

## Alcance

El backend local escucha en `127.0.0.1`. La aplicación ya tiene login con Supabase Auth y autoriza únicamente los IDs de `AUTH_USUARIOS_PERMITIDOS`. La cuenta inicial es `admin@colegio.com`; la contraseña está en el archivo privado `backend/ACCESO-ADMIN.txt`. Los tokens permanecen en Go y la sesión del navegador utiliza una cookie HttpOnly. La clave de servidor nunca se coloca en React.

Para AWS consulta `deploy/aws/LEEME.md`. Se transfiere la clave privada por separado junto con `secrets/auth.env`, que contiene la configuración del login. El paquete público no incluye la contraseña del administrador.

Referencias oficiales: [Data API](https://supabase.com/docs/guides/api), [claves del proyecto](https://supabase.com/docs/guides/getting-started/api-keys), [RLS](https://supabase.com/docs/guides/database/postgres/row-level-security).

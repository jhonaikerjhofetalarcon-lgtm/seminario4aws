# Sistema de matriculas publicado en AWS

Estado verificado el 9 de noviembre de 2026 (America/Lima): **publicado**.

- URL: https://d3grale84xd0q8.cloudfront.net
- Administrador: `admin@colegio.com`.
- La contrasena sigue en `backend/ACCESO-ADMIN.txt` de este equipo; no esta en el paquete ni en este documento.
- Cuenta AWS: `172155958605`; region: `us-east-2` (Ohio).
- Pila CloudFormation: `aula-matriculas`, estado `CREATE_COMPLETE`.
- Distribucion CloudFront: `E2QS4GO9OZBU50`, estado `Deployed`.
- Instancia EC2: `i-0af4e3effc0a6e34a`, tipo `t3.small`, Amazon Linux 2023.
- Bucket privado: `aula-matriculas-paquetes-djfsa1gdhguz`.
- Datos y autenticacion: proyecto Supabase existente.

## Comprobaciones realizadas

Pasaron las pruebas de Go y las **15 pruebas de autenticacion sobre la URL publica**: acceso autorizado, cookies HttpOnly/Secure/SameSite, consulta real de alumnos y matriculas, proteccion frente a sesiones falsas, origen externo y cierre de sesion.

`pruebas/sitio-publico.mjs` verifico la pagina, sus ocho archivos JavaScript/CSS, el servicio de salud y la redireccion de HTTP a HTTPS. Se corrigieron los permisos de los archivos publicos extraidos con umask 077. La clave privada y la configuracion del backend conservan permisos 600.

La revision automatica bloqueo el arranque del navegador de prueba con `blocked by policy`; no se ejecuto la prueba visual en navegador durante este despliegue. Las pruebas HTTP y de login remoto si se ejecutaron y pasaron.

## Administrar el servidor

Abre AWS Console en Ohio y entra en **Systems Manager > Session Manager**. Selecciona la instancia indicada. No se necesita una clave SSH y el puerto 22 esta cerrado.

```bash
sudo systemctl status aula-api aula-web --no-pager
sudo journalctl -u aula-api -u aula-web -n 80 --no-pager
sudo systemctl restart aula-api aula-web
```

La API escucha en `127.0.0.1:8080`. Caddy publica React y redirige `/api/*` hacia Go. El puerto 80 de EC2 acepta solo la lista de origenes de CloudFront; el navegador usa HTTPS hasta CloudFront. El tramo CloudFront-EC2 utiliza HTTP.

La configuracion privada final esta en `/etc/aula/`. La copia temporal subida al bucket y sus archivos de extraccion se eliminaron despues de instalar. No publiques ni copies estos secretos al frontend.

## Actualizaciones y costos

Genera `dist/aula-aws.zip` con `empaquetar.ps1`. Prepara la configuracion con `preparar-secretos.ps1`, solicita URLs firmadas nuevas para el bucket privado y transfiere mediante `subir-archivos.ps1`. Descarga los archivos desde la instancia con su rol, comprueba SHA256, extrae el paquete y la configuracion en `/opt/aula-instalacion` y ejecuta `sudo bash deploy/aws/instalar.sh`. Elimina despues la configuracion temporal tanto del bucket (incluida su version) como del directorio de instalacion.

La URL de CloudFront es estable. La instancia usa una IP publica automatica: **si detienes y vuelves a encender EC2**, revisa su nuevo nombre DNS publico y actualiza el origen de CloudFront. Un reinicio normal conserva la IP. Para evitar este paso se puede asociar una Elastic IP y actualizar la plantilla y el origen.

Los servicios se inician automaticamente al reiniciar el servidor. Reiniciar la API cierra las sesiones activas; los usuarios deben volver a entrar.

EC2, EBS, IPv4 publica, S3 y CloudFront pueden generar cargos mientras existan o se usen. Para retirar el despliegue, elimina la pila `aula-matriculas` desde CloudFormation. El bucket tiene politica Retain: conserva los paquetes y debe eliminarse por separado si ya no se necesita. La base de datos y las cuentas de Supabase no se eliminan con la pila.

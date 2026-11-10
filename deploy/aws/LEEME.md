# Aula en AWS EC2, con login y sin Docker

## Despliegue publico con AWS Core

La plantilla `publico.cloudformation.json` permite publicar sin dominio propio: CloudFront proporciona una URL `https://....cloudfront.net`, redirige HTTP a HTTPS y transmite las cookies del login sin almacenar respuestas en cache. El servidor EC2 solo recibe conexiones HTTP de los servidores de origen de CloudFront; los puertos SSH y de la API permanecen cerrados. La administracion se realiza mediante Systems Manager.

Esta alternativa crea su propia instancia, rol y bucket privado, sin modificar servidores de otros trabajos. El bucket cifra los archivos, bloquea el acceso publico y exige HTTPS. La configuracion privada se transfiere separada del paquete, se instala con permisos restringidos y se elimina del bucket tras la instalacion. La regla de ciclo de vida elimina tambien versiones anteriores de esa configuracion.

`subir-archivos.ps1 -ArchivoUrls RUTA-PRIVADA.json -UrlPublica https://DISTRIBUCION.cloudfront.net` recibe dos URLs S3 firmadas (`paquete` y `configuracion`). El archivo JSON contiene credenciales temporales de transferencia: guardalo fuera del proyecto y no lo compartas. El script construye la configuracion privada en memoria y no muestra claves ni contrasenas.

La cuenta conectada permite EC2 en Ohio (`us-east-2`). El sistema esta publicado en https://d3grale84xd0q8.cloudfront.net y sus identificadores se documentan en `DESPLIEGUE.md`. EC2, EBS, IPv4, S3 y CloudFront pueden generar cargos. El certificado HTTPS termina en CloudFront; CloudFront se comunica por HTTP con el origen restringido por el grupo de seguridad.

El paquete `dist/aula-aws.zip` contiene Go compilado para Linux x86_64, React estático, Caddy y dos servicios systemd. EC2 no necesita Node.js, Go ni Docker. Supabase conserva los alumnos y las matrículas y verifica el correo y la contraseña mediante Supabase Auth.

## Cuenta de acceso

La cuenta administradora es `admin@colegio.com`. La contraseña inicial está únicamente en `backend/ACCESO-ADMIN.txt` de tu equipo, excluida de Git y del ZIP. No subas ese archivo a AWS ni lo publiques. El acceso utiliza la pantalla de login de Aula.

Solo los IDs incluidos en `AUTH_USUARIOS_PERMITIDOS` pueden administrar los datos. Tener una cuenta cualquiera en Supabase no concede acceso. Los administradores autorizados pueden gestionar todos los alumnos y matrículas; no hay permisos individuales por alumno.

La sesión utiliza una cookie HttpOnly, SameSite=Lax y Secure cuando `PUBLIC_ORIGIN` es HTTPS. Los tokens de Supabase permanecen en la memoria de Go. La sesión dura como máximo 8 horas y se pierde al reiniciar el backend; hay que iniciar sesión de nuevo. Esta configuración está preparada para una sola instancia EC2.

## 1. Preparar archivos en tu equipo

Desde `seminario 4 react`, en PowerShell:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File deploy/aws/preparar-secretos.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File deploy/aws/empaquetar.ps1
```

El ZIP ya está generado. Solo vuelve a empaquetar si cambias el código. El primer comando prepara dos archivos que se transfieren por separado y están excluidos del ZIP:

- `deploy/aws/secrets/supabase_key.txt`: clave privada para acceder a los datos desde Go.
- `deploy/aws/secrets/auth.env`: clave pública de Auth y lista de IDs administradores autorizados.

El empaquetador comprueba que no incluya ni la clave privada ni la contraseña del administrador. Se genera también `dist/aula-aws.zip.sha256`. Los ejecutables y la exportación estática intermedios quedan en la carpeta temporal del sistema; en `dist` solo se conservan el ZIP y su checksum.

## 2. Crear EC2

En la región elegida de AWS, crea un par de claves EC2 y guarda el `.pem`. Abre **CloudFormation → Create stack → Upload a template file** y carga `deploy/aws/ec2.cloudformation.json`.

| Parámetro | Valor |
| --- | --- |
| VpcId | VPC con acceso a Internet; puede ser la predeterminada. |
| SubnetId | Subred pública de esa misma VPC, con ruta a un Internet Gateway. |
| KeyName | Par de claves EC2 de esa región. |
| AdminCidr | Tu IP pública seguida de `/32`. Actualízala si cambia tu conexión. |
| InstanceType | `t3.small` por defecto. |
| LatestAmiId | Deja la AMI pública de Amazon Linux 2023. |

Revisa el costo de EC2, EBS e IPv4 en tu región; no se asume cobertura gratuita. La plantilla crea un rol IAM: acepta el reconocimiento de IAM cuando AWS lo solicite. Espera `CREATE_COMPLETE` y copia **Outputs → PublicIp**. Estos pasos describen una instalación manual alternativa; el despliegue con AWS Core ya está creado y se detalla en `DESPLIEGUE.md`.

El puerto 8080 de Go no se expone a Internet. El 22 solo acepta tu dirección `/32`; los puertos 80 y 443 se reservan para el sitio público.

## 3. Transferir el ZIP y la configuración

En PowerShell, desde `seminario 4 react`, reemplaza `TU-CLAVE.pem` e `IP-EC2`:

```powershell
scp -i "TU-CLAVE.pem" "dist/aula-aws.zip" "dist/aula-aws.zip.sha256" ec2-user@IP-EC2:~/
ssh -i "TU-CLAVE.pem" ec2-user@IP-EC2
```

En EC2:

```bash
sudo cloud-init status --wait
test -f ~/aula-servidor-listo
cd ~
sha256sum -c aula-aws.zip.sha256
mkdir -p ~/aula
unzip -o ~/aula-aws.zip -d ~/aula
install -d -m 700 ~/aula/deploy/aws/secrets
```

En otra terminal de PowerShell de tu equipo:

```powershell
scp -i "TU-CLAVE.pem" "deploy/aws/secrets/supabase_key.txt" "deploy/aws/secrets/auth.env" ec2-user@IP-EC2:~/aula/deploy/aws/secrets/
```

## 4. Elegir dominio público o túnel SSH

**Con dominio:** apunta un registro DNS `A` a `PublicIp`. No crees un registro `AAAA` sin configurar IPv6. En EC2:

```bash
cd ~/aula
cp deploy/aws/.env.example deploy/aws/.env
nano deploy/aws/.env
```

Cambia `SITE_ADDRESS` por tu dominio, sin `https://`, y `PUBLIC_ORIGIN` por `https://` seguido del mismo dominio, sin barra final. La URL de Supabase ya corresponde al proyecto. Los puertos 80 y 443 deben estar disponibles para obtener el certificado. Al detener y encender EC2, su IP puede cambiar: actualiza el DNS o utiliza una Elastic IP.

**Sin dominio, para presentar el seminario:**

```bash
cd ~/aula
cp deploy/aws/.env.tunnel.example deploy/aws/.env
```

El instalador escuchará en `127.0.0.1:8088`. El acceso externo se realiza mediante el túnel SSH cifrado del paso 6.

## 5. Instalar e iniciar

En EC2, desde `~/aula`:

```bash
chmod 600 deploy/aws/secrets/*
sudo bash deploy/aws/instalar.sh
sudo systemctl status aula-api aula-web --no-pager
```

El instalador descarga Caddy de su distribución oficial y verifica su SHA512. Los servicios arrancan automáticamente al reiniciar EC2. La clave privada queda en `/etc/aula/supabase_key.txt`, legible solo para el usuario de la API; la configuración del login queda en `/etc/aula/backend.env`. No introduzcas la clave privada en React.

## 6. Abrir el sistema

Con dominio, abre `https://TU-DOMINIO` y entra en la pantalla de login con `admin@colegio.com` y la contraseña del archivo privado de tu equipo.

Sin dominio, mantén abierta esta conexión desde PowerShell:

```powershell
ssh -i "TU-CLAVE.pem" -N -L 8088:127.0.0.1:8088 ec2-user@IP-EC2
```

Abre `http://localhost:8088` y usa la misma cuenta. Para compartir una URL pública utiliza el modo con dominio; el túnel requiere acceso SSH autorizado.

Verifica que puedas consultar los datos, recargar sin perder la sesión y usar **Cerrar sesión**. Sin sesión, la API devuelve HTTP 401 y no permite consultar ni modificar datos. `/healthz` devuelve solo `{"estado":"ok"}` y comprueba el proceso, no la base de datos.

## 7. Registros y actualizaciones

```bash
sudo journalctl -u aula-api -u aula-web -f
sudo systemctl restart aula-api aula-web
sudo systemctl stop aula-web aula-api
```

Para actualizar, genera y transfiere otro ZIP, extráelo y ejecuta el instalador. Conserva `.env` y `secrets/`; no están incluidos en el paquete. Tras reiniciar el backend, los usuarios deben volver a iniciar sesión.

Para autorizar otra cuenta, créala en Supabase Auth y añade su ID a `AUTH_USUARIOS_PERMITIDOS` en `backend/.env`. Vuelve a preparar y transferir `secrets/auth.env` y ejecuta el instalador. El frontend no ofrece registro público. Los metadatos editables por el usuario no conceden permisos.

Eliminar la pila de AWS no borra la base de datos ni las cuentas de Supabase. Guarda antes cualquier configuración local que necesites conservar.

## Validación

Pasaron las pruebas de Go, las 15 pruebas funcionales y las 15 comprobaciones de login. El navegador verificó el ingreso, la sesión al recargar, el cierre y el diseño móvil. Las pruebas permanecen en `pruebas`; las capturas solo se generan si se define `GUARDAR_CAPTURAS=1`. La plantilla EC2 pasó `cfn-lint 1.60.0` y el ZIP se revisa en cada empaquetado.

El instalador systemd ya se ejecutó en EC2. Pasaron las 15 comprobaciones de autenticación sobre el enlace HTTPS de CloudFront y la verificación de la página y sus ocho archivos JavaScript/CSS. La prueba visual de navegador durante el despliegue quedó bloqueada por la revisión automática; los controles HTTP sí se completaron. La ruta de salud comprueba el proceso, no la disponibilidad de Supabase.

El asesor de Supabase detectó desactivada la [protección contra contraseñas filtradas](https://supabase.com/docs/guides/auth/password-security#password-strength-and-leaked-password-protection), disponible en planes Pro y superiores. La contraseña inicial se generó con un generador criptográfico; no se modificó el plan de la cuenta. Los avisos informativos de RLS sin políticas corresponden a las tablas privadas del backend, cuyo acceso público sigue revocado.

## Referencias

- [Amazon Linux 2023 y CloudFormation](https://docs.aws.amazon.com/linux/al2023/ug/ec2.html).
- [Supabase Auth: contraseñas](https://supabase.com/docs/guides/auth/passwords).
- [Next.js: exportación estática](https://nextjs.org/docs/app/guides/static-exports).
- [Caddy: HTTPS automático](https://caddyserver.com/docs/automatic-https).

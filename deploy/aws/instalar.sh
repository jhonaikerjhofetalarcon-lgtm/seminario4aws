#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
test "$(id -u)" = 0 || { echo 'Ejecuta con sudo bash deploy/aws/instalar.sh' >&2; exit 1; }
test -f bin/academia && test -f web/index.html || { echo 'Extrae el ZIP compilado aula-aws.zip.' >&2; exit 1; }

# Instala Caddy desde su distribucion oficial y verifica el SHA512 publicado.
version=2.11.9
temporal=$(mktemp -d)
trap 'rm -rf -- "$temporal"' EXIT
archivo="caddy_${version}_linux_amd64.tar.gz"
curl -fL --retry 3 "https://github.com/caddyserver/caddy/releases/download/v${version}/${archivo}" -o "$temporal/$archivo"
curl -fL --retry 3 "https://github.com/caddyserver/caddy/releases/download/v${version}/caddy_${version}_checksums.txt" -o "$temporal/checksums.txt"
(cd "$temporal" && grep " ${archivo}$" checksums.txt | sha512sum -c -)
tar -xzf "$temporal/$archivo" -C "$temporal" caddy
install -m 755 "$temporal/caddy" /usr/local/bin/caddy
if test "${1:-}" = '--solo-caddy'; then
  echo 'Caddy instalado.'
  exit 0
fi
test -s deploy/aws/.env && test -s deploy/aws/secrets/supabase_key.txt && test -s deploy/aws/secrets/auth.env || { echo 'Faltan .env, el secreto privado o auth.env. Consulta LEEME.md.' >&2; exit 1; }
if grep -Eq 'REEMPLAZAR|tu-dominio|TU-PROYECTO' deploy/aws/.env; then
  echo 'Completa los valores de ejemplo en deploy/aws/.env.' >&2
  exit 1
fi

id aula-api >/dev/null 2>&1 || useradd --system --user-group --home-dir /opt/aula --shell /sbin/nologin aula-api
id aula-web >/dev/null 2>&1 || useradd --system --user-group --home-dir /var/lib/aula-web --shell /sbin/nologin aula-web
install -d -m 755 /opt/aula /opt/aula/web /etc/caddy
install -d -m 750 -o root -g aula-api /etc/aula
install -d -m 700 -o aula-web -g aula-web /var/lib/aula-web
systemctl stop aula-web aula-api 2>/dev/null || true
install -m 755 bin/academia /opt/aula/academia
cp -a web/. /opt/aula/web/
# El ZIP puede extraerse con umask 077. Solo los recursos publicos de React
# deben poder ser leidos por el usuario separado de Caddy.
find /opt/aula/web -type d -exec chmod 755 {} +
find /opt/aula/web -type f -exec chmod 644 {} +
install -m 600 -o aula-api -g aula-api deploy/aws/secrets/supabase_key.txt /etc/aula/supabase_key.txt
install -m 600 deploy/aws/.env /etc/aula/web.env
install -m 644 deploy/aws/Caddyfile /etc/caddy/aula.Caddyfile

origen=$(sed -n 's/^PUBLIC_ORIGIN=//p' deploy/aws/.env | tr -d '\r')
supabase=$(sed -n 's/^SUPABASE_URL=//p' deploy/aws/.env | tr -d '\r')
test -n "$origen" && test -n "$supabase" || { echo 'Faltan PUBLIC_ORIGIN o SUPABASE_URL.' >&2; exit 1; }
printf 'ORIGEN_FRONTEND=%s\nSUPABASE_URL=%s\n' "$origen" "$supabase" > /etc/aula/backend.env
grep -E '^(SUPABASE_PUBLISHABLE_KEY|AUTH_USUARIOS_PERMITIDOS)=' deploy/aws/secrets/auth.env | tr -d '\r' >> /etc/aula/backend.env
chmod 600 /etc/aula/backend.env

# Sin dominio se escucha solo en loopback; el acceso externo pasa por SSH.
if grep -q '^SITE_ADDRESS=http://:80' deploy/aws/.env; then
  sed -i 's|^SITE_ADDRESS=.*|SITE_ADDRESS=http://localhost:8088|' /etc/aula/web.env
  sed -i '/^{$SITE_ADDRESS} {/a\	bind 127.0.0.1' /etc/caddy/aula.Caddyfile
fi
install -m 644 deploy/aws/aula-api.service deploy/aws/aula-web.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now aula-api aula-web
listo=false
for intento in $(seq 1 30); do
  if curl --fail --silent http://127.0.0.1:8080/healthz; then listo=true; break; fi
  sleep 1
done
test "$listo" = true || { echo 'La API no arranco. Revisa journalctl -u aula-api.' >&2; exit 1; }
systemctl is-active --quiet aula-api
systemctl is-active --quiet aula-web
echo
echo 'Aula instalada. Los servicios arrancan automaticamente al reiniciar EC2.'
echo 'Registros: sudo journalctl -u aula-api -u aula-web -f'

$ErrorActionPreference = 'Stop'
$rutaBackendEnv = Join-Path $PSScriptRoot '../../backend/.env'
$configuracion = @{}
foreach ($linea in Get-Content -LiteralPath $rutaBackendEnv) {
    if ($linea -match '^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=(.*)$') {
        $configuracion[$matches[1]] = $matches[2].Trim().Trim([char]34, [char]39)
    }
}
$clave = $configuracion['SUPABASE_SECRET_KEY']
if (-not $clave) { $clave = $configuracion['SUPABASE_SERVICE_ROLE_KEY'] }
if (-not $clave) { throw 'Falta la clave privada en backend/.env.' }
$carpeta = Join-Path $PSScriptRoot 'secrets'
$null = New-Item -ItemType Directory -Path $carpeta -Force
[IO.File]::WriteAllText((Join-Path $carpeta 'supabase_key.txt'), $clave, (New-Object Text.UTF8Encoding($false)))
if (-not $configuracion['SUPABASE_PUBLISHABLE_KEY'] -or -not $configuracion['AUTH_USUARIOS_PERMITIDOS']) {
    throw 'Falta la configuracion del administrador y del login en backend/.env.'
}
$auth = 'SUPABASE_PUBLISHABLE_KEY=' + $configuracion['SUPABASE_PUBLISHABLE_KEY'] + "`n" + 'AUTH_USUARIOS_PERMITIDOS=' + $configuracion['AUTH_USUARIOS_PERMITIDOS'] + "`n"
[IO.File]::WriteAllText((Join-Path $carpeta 'auth.env'), $auth, (New-Object Text.UTF8Encoding($false)))
Write-Host 'Secreto preparado en deploy/aws/secrets/supabase_key.txt. No se incluye en el ZIP ni en las imagenes.'
Write-Host 'Configuracion del login preparada en deploy/aws/secrets/auth.env. Transfierela tambien por SSH.'

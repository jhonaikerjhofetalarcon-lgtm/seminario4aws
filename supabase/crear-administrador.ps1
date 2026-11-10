param(
    [string]$Correo = 'admin@colegio.com',
    [Parameter(Mandatory = $true)][string]$ClavePublica
)
$ErrorActionPreference = 'Stop'
$backend = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../backend'))
$rutaEnv = Join-Path $backend '.env'
$config = @{}
foreach ($linea in Get-Content -LiteralPath $rutaEnv) {
    if ($linea -match '^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=(.*)$') {
        $config[$matches[1]] = $matches[2].Trim().Trim([char]34, [char]39)
    }
}
$privada = $config['SUPABASE_SECRET_KEY']
if (-not $privada) { $privada = $config['SUPABASE_SERVICE_ROLE_KEY'] }
if (-not $privada -or -not $config['SUPABASE_URL']) { throw 'Faltan las credenciales privadas de Supabase.' }
$cabeceras = @{ apikey = $privada }
if (-not $privada.StartsWith('sb_secret_')) { $cabeceras['Authorization'] = 'Bearer ' + $privada }
$base = $config['SUPABASE_URL'].TrimEnd('/') + '/auth/v1'
try {
    $lista = Invoke-RestMethod -UserAgent 'AulaConfiguracion' -Uri ($base + '/admin/users?page=1&per_page=1000') -Headers $cabeceras -TimeoutSec 20
} catch { throw 'No se pudo consultar Supabase Auth; revisa la clave privada.' }
$existente = @($lista.users | Where-Object { $_.email -eq $Correo })
$credenciales = Join-Path $backend 'ACCESO-ADMIN.txt'
if ($existente.Count -gt 0) {
    if (-not (Test-Path -LiteralPath $credenciales)) { throw 'La cuenta ya existe. No se cambia su clave automaticamente; usa Supabase para restablecerla.' }
    $usuario = $existente[0]
} else {
    $aleatorio = New-Object byte[] 24
    $generador = [Security.Cryptography.RandomNumberGenerator]::Create()
    try { $generador.GetBytes($aleatorio) } finally { $generador.Dispose() }
    $clave = 'Aula!' + [Convert]::ToBase64String($aleatorio)
    $cuerpo = @{ email = $Correo; password = $clave; email_confirm = $true; app_metadata = @{ aplicacion = 'aula' } } | ConvertTo-Json -Compress
    try {
        $usuario = Invoke-RestMethod -Method Post -UserAgent 'AulaConfiguracion' -Uri ($base + '/admin/users') -Headers $cabeceras -ContentType 'application/json; charset=utf-8' -Body ([Text.Encoding]::UTF8.GetBytes($cuerpo)) -TimeoutSec 20
    } catch { throw 'No se pudo crear el administrador en Supabase Auth.' }
    [IO.File]::WriteAllText($credenciales, ("Correo=$Correo`nClave=$clave`nAcceso local=http://localhost:3000`n`nArchivo privado. No lo subas a Git ni a AWS. Puedes cambiar la clave en Supabase Authentication > Users.`n"), (New-Object Text.UTF8Encoding($false)))
}
if (-not $usuario.id) { throw 'Supabase Auth no devolvio el ID de usuario.' }
$ids = @($config['AUTH_USUARIOS_PERMITIDOS'] -split ',' | Where-Object { $_ })
$ids += $usuario.id
$valores = @{ SUPABASE_PUBLISHABLE_KEY = $ClavePublica; AUTH_USUARIOS_PERMITIDOS = (($ids | Select-Object -Unique) -join ',') }
$contenido = [IO.File]::ReadAllText($rutaEnv)
foreach ($nombre in $valores.Keys) {
    if ($contenido -match "(?m)^$nombre=") {
        $contenido = [regex]::Replace($contenido, "(?m)^$nombre=[^\r\n]*", ($nombre + '=' + $valores[$nombre]))
    } else { $contenido += "`n$nombre=$($valores[$nombre])`n" }
}
[IO.File]::WriteAllText($rutaEnv, $contenido, (New-Object Text.UTF8Encoding($false)))
Write-Host "Cuenta autorizada: $Correo. Credenciales guardadas en backend/ACCESO-ADMIN.txt."

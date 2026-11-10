# Crea un paquete Linux listo para instalar; no requiere Docker.
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.IO.Compression
$raiz = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../..'))
$dist = Join-Path $raiz 'dist'
$baseTemporal = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
$trabajo = [IO.Path]::GetFullPath((Join-Path $baseTemporal ('aula-empaquetado-' + [guid]::NewGuid().ToString('N'))))
if (-not $trabajo.StartsWith($baseTemporal, [StringComparison]::OrdinalIgnoreCase)) { throw 'Directorio temporal no valido.' }
$bin = Join-Path $trabajo 'bin'
$null = New-Item -ItemType Directory -Path $dist -Force
$null = New-Item -ItemType Directory -Path $bin -Force
$variables = @('GOOS', 'GOARCH', 'CGO_ENABLED', 'EXPORTAR_ESTATICO', 'DIRECTORIO_NEXT', 'NEXT_PUBLIC_API_URL', 'NEXT_TELEMETRY_DISABLED')
$anteriores = @{}
foreach ($nombre in $variables) { $anteriores[$nombre] = [Environment]::GetEnvironmentVariable($nombre, 'Process') }
Push-Location $raiz
try {
    $env:GOOS = 'linux'; $env:GOARCH = 'amd64'; $env:CGO_ENABLED = '0'
    Push-Location (Join-Path $raiz 'backend')
    try {
        go build -trimpath -ldflags='-s -w' -o (Join-Path $bin 'academia') .
        if ($LASTEXITCODE -ne 0) { throw 'Fallo la compilacion de Go para Linux.' }
    } finally { Pop-Location }
    $env:EXPORTAR_ESTATICO = '1'
    $env:DIRECTORIO_NEXT = '.next-aws'
    $env:NEXT_PUBLIC_API_URL = '/api'
    $env:NEXT_TELEMETRY_DISABLED = '1'
    Push-Location (Join-Path $raiz 'frontend')
    try {
        if (-not (Test-Path -LiteralPath 'node_modules/next')) {
            npm.cmd ci
            if ($LASTEXITCODE -ne 0) { throw 'Fallo npm ci.' }
        }
        npm.cmd run build
        if ($LASTEXITCODE -ne 0) { throw 'Fallo la exportacion de React.' }
    } finally { Pop-Location }
} finally {
    Pop-Location
    foreach ($nombre in $variables) { [Environment]::SetEnvironmentVariable($nombre, $anteriores[$nombre], 'Process') }
}

$archivos = @()
$archivos += @{ archivo = (Join-Path $bin 'academia'); entrada = 'bin/academia' }
$web = Join-Path $raiz 'frontend/.next-aws'
foreach ($archivo in Get-ChildItem -LiteralPath $web -File -Recurse) {
    $archivos += @{ archivo = $archivo.FullName; entrada = 'web/' + $archivo.FullName.Substring($web.Length + 1).Replace('\', '/') }
}
foreach ($nombre in @('.env.example', '.env.tunnel.example', 'Caddyfile', 'aula-api.service', 'aula-web.service', 'instalar.sh', 'ec2.cloudformation.json', 'publico.cloudformation.json', 'LEEME.md')) {
    $ruta = Join-Path $PSScriptRoot $nombre
    if (-not (Test-Path -LiteralPath $ruta)) { throw "Falta un archivo del despliegue: $nombre" }
    $archivos += @{ archivo = $ruta; entrada = 'deploy/aws/' + $nombre }
}
$rutaZip = Join-Path $dist 'aula-aws.zip'
$flujo = [IO.File]::Open($rutaZip, [IO.FileMode]::Create)
$zip = New-Object IO.Compression.ZipArchive($flujo, [IO.Compression.ZipArchiveMode]::Create)
try {
    foreach ($archivo in $archivos) {
        $entrada = $zip.CreateEntry($archivo.entrada, [IO.Compression.CompressionLevel]::Optimal)
        $origen = [IO.File]::OpenRead($archivo.archivo)
        $destino = $entrada.Open()
        try { $origen.CopyTo($destino) } finally { $origen.Dispose(); $destino.Dispose() }
    }
} finally { $zip.Dispose(); $flujo.Dispose() }
Write-Host "Paquete Linux sin Docker: $rutaZip"
Write-Host 'No contiene claves, datos locales, node_modules ni archivos .env privados.'
$hash = Get-FileHash -LiteralPath $rutaZip -Algorithm SHA256
[IO.File]::WriteAllText(($rutaZip + '.sha256'), ($hash.Hash.ToLower() + '  aula-aws.zip' + "`n"), (New-Object Text.UTF8Encoding($false)))
& (Join-Path $PSScriptRoot 'validar-paquete.ps1')
if ($LASTEXITCODE -ne 0) { throw 'Fallo la validacion del paquete.' }
# Los intermedios viven en la carpeta temporal del sistema. El proyecto conserva
# solo el ZIP y el checksum; no hace falta borrar directorios recursivamente.
$webResuelta = (Resolve-Path -LiteralPath $web).ProviderPath
$destinoWeb = [IO.Path]::GetFullPath((Join-Path $trabajo 'web'))
if (-not $webResuelta.StartsWith($raiz + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase) -or
    -not $destinoWeb.StartsWith($trabajo + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) {
    throw 'Rutas de trabajo fuera de los directorios previstos.'
}
if ((Get-Item -LiteralPath $webResuelta -Force).Attributes -band [IO.FileAttributes]::ReparsePoint) { throw 'No se mueve un enlace de carpeta.' }
Move-Item -LiteralPath $webResuelta -Destination $destinoWeb
$hash

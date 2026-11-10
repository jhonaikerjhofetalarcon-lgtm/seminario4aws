param(
    [Parameter(Mandatory=$true)][string]$ArchivoUrls,
    [Parameter(Mandatory=$true)][string]$UrlPublica
)
# Transfiere a URLs S3 firmadas. Las claves privadas no salen por la consola.
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
Add-Type -AssemblyName System.IO.Compression
$raiz = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../..'))
$urls = Get-Content -LiteralPath $ArchivoUrls -Raw | ConvertFrom-Json
foreach ($url in @($urls.paquete, $urls.configuracion)) {
    $uri = [uri]$url
    if ($uri.Scheme -ne 'https' -or $uri.Host -notmatch '\.s3[.-]us-east-2\.amazonaws\.com$') { throw 'Destino S3 no valido.' }
}
if ($UrlPublica -notmatch '^https://[a-z0-9]+\.cloudfront\.net$') { throw 'Origen CloudFront no valido.' }
$zipRuta = Join-Path $raiz 'dist/aula-aws.zip'
$hash = (Get-FileHash -LiteralPath $zipRuta -Algorithm SHA256).Hash.ToLower()
$esperado = ((Get-Content -LiteralPath ($zipRuta + '.sha256') -Raw).Trim() -split '\s+')[0]
if ($hash -ne $esperado) { throw 'Checksum incorrecto.' }
$supabase = ''
foreach ($linea in Get-Content -LiteralPath (Join-Path $raiz 'backend/.env')) {
    if ($linea -match '^SUPABASE_URL=(.+)$') { $supabase = $matches[1].Trim().Trim('"').Trim("'") }
}
if ($supabase -notmatch '^https://[a-z0-9]+\.supabase\.co$') { throw 'URL Supabase no valida.' }
$memoria = New-Object IO.MemoryStream
$zip = New-Object IO.Compression.ZipArchive($memoria, [IO.Compression.ZipArchiveMode]::Create, $true)
try {
    foreach ($nombre in @('supabase_key.txt', 'auth.env')) {
        $ruta = Join-Path $PSScriptRoot ('secrets/' + $nombre)
        if (-not (Test-Path -LiteralPath $ruta) -or (Get-Item -LiteralPath $ruta).Length -eq 0) { throw 'Falta configuracion privada.' }
        $entrada = $zip.CreateEntry('deploy/aws/secrets/' + $nombre)
        $origen = [IO.File]::OpenRead($ruta)
        $destino = $entrada.Open()
        try { $origen.CopyTo($destino) } finally { $origen.Dispose(); $destino.Dispose() }
    }
    $entrada = $zip.CreateEntry('deploy/aws/.env')
    $destino = $entrada.Open()
    $bytes = [Text.Encoding]::UTF8.GetBytes("SITE_ADDRESS=:80`nPUBLIC_ORIGIN=$UrlPublica`nSUPABASE_URL=$supabase`n")
    try { $destino.Write($bytes, 0, $bytes.Length) } finally { $destino.Dispose() }
} finally { $zip.Dispose() }
try {
    $null = Invoke-WebRequest -UseBasicParsing -Method Put -Uri $urls.paquete -InFile $zipRuta -ContentType 'application/octet-stream'
    $null = Invoke-WebRequest -UseBasicParsing -Method Put -Uri $urls.configuracion -Body $memoria.ToArray() -ContentType 'application/octet-stream'
    Write-Host "Paquete y configuracion transferidos por HTTPS. SHA256: $hash"
} catch {
    throw 'Fallo la transferencia a S3; revisa la conexion y la vigencia de las URLs firmadas.'
} finally { $memoria.Dispose() }

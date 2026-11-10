$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.IO.Compression
$raiz = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../..'))
$rutaZip = Join-Path $raiz 'dist/aula-aws.zip'
$secreto = Join-Path $PSScriptRoot 'secrets/supabase_key.txt'
$clave = if (Test-Path -LiteralPath $secreto) { [IO.File]::ReadAllText($secreto).Trim() } else { '' }
$claveAdmin = ''
$accesoAdmin = Join-Path $raiz 'backend/ACCESO-ADMIN.txt'
if (Test-Path -LiteralPath $accesoAdmin) {
    foreach ($linea in Get-Content -LiteralPath $accesoAdmin) { if ($linea -match '^Clave=(.*)$') { $claveAdmin = $matches[1] } }
}
$flujo = [IO.File]::OpenRead($rutaZip)
$zip = New-Object IO.Compression.ZipArchive($flujo, [IO.Compression.ZipArchiveMode]::Read)
try {
    foreach ($necesario in @('bin/academia', 'web/index.html', 'deploy/aws/Caddyfile', 'deploy/aws/instalar.sh', 'deploy/aws/ec2.cloudformation.json', 'deploy/aws/aula-api.service', 'deploy/aws/aula-web.service', 'deploy/aws/.env.example', 'deploy/aws/.env.tunnel.example', 'deploy/aws/LEEME.md')) {
        if (-not $zip.GetEntry($necesario)) { throw "Falta un archivo necesario en el paquete: $necesario" }
    }
    foreach ($entrada in $zip.Entries) {
        if ($entrada.FullName -match '(^|/)(\.env|secrets|academia\.json|ACCESO-ADMIN\.txt|node_modules)(/|$)') {
            throw 'El paquete contiene un archivo privado o una dependencia local.'
        }
        $origen = $entrada.Open()
        $memoria = New-Object IO.MemoryStream
        try { $origen.CopyTo($memoria); $bytes = $memoria.ToArray() } finally { $origen.Dispose(); $memoria.Dispose() }
        $texto = [Text.Encoding]::UTF8.GetString($bytes)
        if ($clave -and $texto.Contains($clave)) { throw 'El paquete contiene la clave privada; no debe transferirse.' }
        if ($claveAdmin -and $texto.Contains($claveAdmin)) { throw 'El paquete contiene la contrasena del administrador.' }
        if ($entrada.FullName.StartsWith('web/') -and $texto.Contains('localhost:8080')) {
            throw 'El frontend compilado sigue apuntando al backend de desarrollo.'
        }
        if ($entrada.FullName -eq 'bin/academia') {
            if ($bytes.Length -lt 20 -or $bytes[0] -ne 127 -or [Text.Encoding]::ASCII.GetString($bytes, 1, 3) -ne 'ELF' -or $bytes[4] -ne 2 -or $bytes[18] -ne 62) {
                throw 'El ejecutable no tiene formato ELF de Linux x86_64.'
            }
        }
    }
    Write-Host "Paquete verificado: $($zip.Entries.Count) archivos; Linux x86_64; sin secretos ni datos locales; frontend con API del mismo origen."
} finally { $zip.Dispose(); $flujo.Dispose() }

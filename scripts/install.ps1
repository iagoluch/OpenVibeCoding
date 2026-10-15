param([string]$Binary, [string]$Version="latest", [switch]$AddToPath)
$ErrorActionPreference = 'Stop'
$staging = $null
if (-not $Binary) {
    $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLower()
    if ($arch -eq 'x64') { $arch = 'amd64' }
    if ($arch -notin @('amd64','arm64')) { throw 'Unsupported CPU architecture' }
    if ($Version -notmatch '^[a-zA-Z0-9._-]+$') { throw 'Invalid version' }
    $base = if ($Version -eq 'latest') { 'https://github.com/iagoluch/OpenVibeCoding/releases/latest/download' } else { "https://github.com/iagoluch/OpenVibeCoding/releases/download/$Version" }
    $staging = Join-Path ([IO.Path]::GetTempPath()) ([Guid]::NewGuid().ToString())
    New-Item -ItemType Directory -Path $staging | Out-Null
    $asset = "interpreter-windows-$arch.zip"
    try {
        Invoke-WebRequest "$base/$asset" -OutFile (Join-Path $staging $asset)
        Invoke-WebRequest "$base/SHA256SUMS" -OutFile (Join-Path $staging 'SHA256SUMS')
        $line = Get-Content (Join-Path $staging 'SHA256SUMS') | Where-Object { $_ -match "  $([Regex]::Escape($asset))$" }
        $expected = ($line -split '\s+')[0]
        if (-not $expected -or (Get-FileHash (Join-Path $staging $asset) -Algorithm SHA256).Hash -ne $expected) { throw 'Checksum mismatch' }
        Expand-Archive (Join-Path $staging $asset) -DestinationPath $staging
        $Binary = Join-Path $staging 'interpreter.exe'
    } catch { Remove-Item $staging -Recurse -Force; throw "Release download failed. Build from source or supply -Binary. $_" }
}
$source = (Resolve-Path $Binary).Path
$version = & $source --version
if ($LASTEXITCODE -ne 0 -or $version -notmatch '^Open Interpreter ') { throw 'Invalid interpreter binary' }
$dest = if ($env:INTERPRETER_BIN_DIR) { $env:INTERPRETER_BIN_DIR } else { Join-Path $env:LOCALAPPDATA 'Programs\OpenInterpreter' }
New-Item -ItemType Directory -Force -Path $dest | Out-Null
foreach ($name in @('interpreter.exe', 'i.exe')) {
    $target = Join-Path $dest $name
    if (Test-Path $target) {
        $existing = & $target --version
        if ($LASTEXITCODE -ne 0 -or $existing -notmatch '^Open Interpreter ') { throw "Refusing to replace unrelated command: $target" }
    }
}
foreach ($name in @('interpreter.exe', 'i.exe')) {
    Copy-Item $source (Join-Path $dest $name) -Force
    & (Join-Path $dest $name) --version
    if ($LASTEXITCODE -ne 0) { throw 'Installation verification failed' }
}
Write-Output "Installed to $dest. Add this directory to your user PATH and reopen your terminal."

if ($AddToPath) {
    $current = [Environment]::GetEnvironmentVariable('PATH','User')
    if ($dest -notin ($current -split ';')) { [Environment]::SetEnvironmentVariable('PATH',"$current;$dest",'User') }
}
if ($staging) { Remove-Item $staging -Recurse -Force }

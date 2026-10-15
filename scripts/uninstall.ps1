$ErrorActionPreference='Stop'
$dest=if($env:OPENVIBECODING_BIN_DIR){$env:OPENVIBECODING_BIN_DIR}else{Join-Path $env:LOCALAPPDATA 'Programs\OpenVibeCoding'}
foreach($name in @('ovc.exe')){
    $path=Join-Path $dest $name
    if(Test-Path $path){if((& $path --version) -match '^OpenVibeCoding '){Remove-Item $path}}
}
Write-Output 'Launcher removed. Configuration and sessions preserved. Stop the runtime before uninstalling.'

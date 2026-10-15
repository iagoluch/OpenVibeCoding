$ErrorActionPreference='Stop'
$dest=if($env:INTERPRETER_BIN_DIR){$env:INTERPRETER_BIN_DIR}else{Join-Path $env:LOCALAPPDATA 'Programs\OpenInterpreter'}
foreach($name in @('interpreter.exe','i.exe')){
    $path=Join-Path $dest $name
    if(Test-Path $path){if((& $path --version) -match '^Open Interpreter '){Remove-Item $path}}
}
Write-Output 'Launchers removed. Configuration and sessions preserved. Stop the runtime before uninstalling.'

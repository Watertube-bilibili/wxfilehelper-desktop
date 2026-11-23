param([string]$Iscc = "", [string]$Python = "")
$ErrorActionPreference = "Stop"
Set-Location (Split-Path -Parent $PSScriptRoot)

if (!$Python) {
    $Python = if (Test-Path .venv/Scripts/python.exe) { ".venv/Scripts/python.exe" } else { (Get-Command python -CommandType Application).Source }
}
$Python = (Resolve-Path -LiteralPath $Python).Path
if (!$Iscc) {
    foreach ($candidate in @(".tools/inno/ISCC.exe", "${env:ProgramFiles(x86)}/Inno Setup 6/ISCC.exe", "$env:ProgramFiles/Inno Setup 7/ISCC.exe")) {
        if (Test-Path -LiteralPath $candidate) { $Iscc = $candidate; break }
    }
}
if (!$Iscc) { throw "Install Inno Setup from https://jrsoftware.org/isinfo.php or pass -Iscc PATH" }
$Iscc = (Resolve-Path -LiteralPath $Iscc).Path
$version = & $Python -c 'from wxfilehelper import __version__; print(__version__)'
if ($LASTEXITCODE -ne 0 -or !$version) { throw "Cannot read project version" }

$savedPath = $env:Path
$savedOnefile = $env:WX_ONEFILE
try {
    # System ICU and Python's own DLLs suffice. Foreign DLLs on PATH can shadow system ICU.
    $env:Path = "$env:SystemRoot\System32;$env:SystemRoot"
    & $Python scripts/collect_licenses.py
    if ($LASTEXITCODE -ne 0) { throw "License collection failed" }
    & $Python scripts/make_icon.py
    if ($LASTEXITCODE -ne 0) { throw "Icon generation failed" }
    $env:WX_ONEFILE = "0"
    & $Python -m PyInstaller --clean --noconfirm packaging/WxFileHelper.spec
    if ($LASTEXITCODE -ne 0) { throw "Directory build failed" }
    $env:WX_ONEFILE = "1"
    & $Python -m PyInstaller --noconfirm packaging/WxFileHelper.spec
    if ($LASTEXITCODE -ne 0) { throw "Portable build failed" }

    $portable = "dist/WXFileHelper-$version-Windows-x64-Portable.exe"
    & "$PSScriptRoot/smoke_windows.ps1" "dist/WXFileHelper/WXFileHelper.exe"
    & "$PSScriptRoot/smoke_windows.ps1" $portable
    $compilerOut = (Join-Path (Resolve-Path build).Path "iscc-output.log")
    $compilerErr = (Join-Path (Resolve-Path build).Path "iscc-error.log")
    $compiler = Start-Process -FilePath $Iscc -ArgumentList @("/DAppVersion=$version", "packaging/windows.iss") `
        -WorkingDirectory (Get-Location).Path -WindowStyle Hidden -Wait -PassThru `
        -RedirectStandardOutput $compilerOut -RedirectStandardError $compilerErr
    Get-Content -LiteralPath $compilerOut, $compilerErr
    if ($compiler.ExitCode -ne 0) { throw "Installer compilation failed" }
    & $Python scripts/checksums.py $portable "dist/WXFileHelper-$version-Windows-x64-Setup.exe"
    if ($LASTEXITCODE -ne 0) { throw "Checksum generation failed" }
}
finally {
    $env:Path = $savedPath
    $env:WX_ONEFILE = $savedOnefile
}

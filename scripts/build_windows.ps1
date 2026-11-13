param([string]$Iscc = "")
$ErrorActionPreference = "Stop"
Set-Location (Split-Path -Parent $PSScriptRoot)
$python = if (Test-Path .venv/Scripts/python.exe) { (Resolve-Path .venv/Scripts/python.exe).Path } else { "python" }
& $python scripts/collect_licenses.py
if ($LASTEXITCODE -ne 0) { throw "License collection failed" }
& $python scripts/make_icon.py
if ($LASTEXITCODE -ne 0) { throw "Icon generation failed" }
$env:WX_ONEFILE = "0"
& $python -m PyInstaller --noconfirm packaging/WxFileHelper.spec
if ($LASTEXITCODE -ne 0) { throw "Directory build failed" }
$env:WX_ONEFILE = "1"
& $python -m PyInstaller --noconfirm packaging/WxFileHelper.spec
if ($LASTEXITCODE -ne 0) { throw "Portable build failed" }
Remove-Item Env:WX_ONEFILE
if (!$Iscc) {
    foreach ($candidate in @(".tools/inno/ISCC.exe", "${env:ProgramFiles(x86)}/Inno Setup 6/ISCC.exe", "$env:ProgramFiles/Inno Setup 7/ISCC.exe")) {
        if (Test-Path -LiteralPath $candidate) { $Iscc = (Resolve-Path -LiteralPath $candidate).Path; break }
    }
}
if (!$Iscc) { throw "Install Inno Setup from https://jrsoftware.org/isinfo.php or pass -Iscc PATH" }
& $Iscc packaging/windows.iss
if ($LASTEXITCODE -ne 0) { throw "Installer compilation failed" }
& $python scripts/checksums.py
if ($LASTEXITCODE -ne 0) { throw "Checksum generation failed" }

param([Parameter(Mandatory = $true)][string]$Executable)

$ErrorActionPreference = "Stop"
$package = (Resolve-Path -LiteralPath $Executable).Path
$isInstaller = $package.EndsWith("-Setup.exe", [StringComparison]::OrdinalIgnoreCase)
$root = (New-Item -ItemType Directory -Force -Path (Join-Path $PSScriptRoot "../../build/windows")).FullName
$sandbox = Join-Path $root ("smoke-" + [guid]::NewGuid().ToString("N"))
$target = Join-Path $sandbox ("app " + [char]0x6D4B + [char]0x8BD5)
$report = Join-Path $sandbox "result.txt"
$previousReport = $env:WXFILEHELPER_SMOKE_REPORT
$uninstallKey = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\{6D2C9269-414D-46E2-8E62-A2C3DC74AACC}_is1"

function Invoke-Hidden([string]$Path, [string[]]$Arguments) {
    $process = Start-Process -FilePath $Path -ArgumentList $Arguments -WorkingDirectory $sandbox `
        -WindowStyle Hidden -PassThru
    try {
        if (!$process.WaitForExit(60000)) {
            $process.Kill()
            throw "Timed out: $Path"
        }
        if ($process.ExitCode -ne 0) {
            throw "Exit $($process.ExitCode): $Path"
        }
    }
    finally { $process.Dispose() }
}

function Invoke-Setup([string]$Path, [string[]]$Arguments) {
    # Inno Setup hands off to a temporary process and must finish its entire
    # process tree before the install directory or registry is checked.
    $process = Start-Process -FilePath $Path -ArgumentList $Arguments -WorkingDirectory $sandbox `
        -WindowStyle Hidden -Wait -PassThru
    if ($process.ExitCode -ne 0) {
        throw "Exit $($process.ExitCode): $Path (logs: $sandbox)"
    }
}

if ($isInstaller -and (Test-Path -LiteralPath $uninstallKey)) {
    throw "A native 1.2 installation is already registered. Test the installer on a clean profile."
}
New-Item -ItemType Directory -Force -Path $sandbox | Out-Null
try {
    $app = $package
    if ($isInstaller) {
        Invoke-Setup $package @("/VERYSILENT", "/SUPPRESSMSGBOXES", "/SP-", "/NORESTART", "/NOICONS", "/DIR=`"$target`"", "/LOG=`"$sandbox\setup.log`"")
        $app = Join-Path $target "WXFileHelper.exe"
        if ((Get-ChildItem -LiteralPath $target -Filter "*.dll" -File -Recurse).Count -ne 0) {
            throw "The installer included an external DLL"
        }
    }
    $env:WXFILEHELPER_SMOKE_REPORT = $report
    Invoke-Hidden $app @("--smoke-test")
    if (!(Test-Path -LiteralPath $report) -or (Get-Content -LiteralPath $report -Raw).Trim() -ne "OK") {
        throw "Application did not verify native window and account tabs"
    }
    Write-Output "Native Windows startup succeeded: $package"
}
finally {
    $env:WXFILEHELPER_SMOKE_REPORT = $previousReport
    if ($isInstaller) {
        $uninstaller = Join-Path $target "unins000.exe"
        if (Test-Path -LiteralPath $uninstaller) {
            Invoke-Setup $uninstaller @("/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART", "/LOG=`"$sandbox\uninstall.log`"")
            if ((Test-Path -LiteralPath (Join-Path $target "WXFileHelper.exe")) -or (Test-Path -LiteralPath $uninstallKey)) {
                throw "Uninstall did not remove the test installation"
            }
            Write-Output "Native Windows uninstall succeeded"
        }
    }
}

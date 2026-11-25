param(
    [Parameter(Mandatory = $true)][string]$Executable,
    [switch]$VerifyShortcuts
)

$ErrorActionPreference = "Stop"
$package = (Resolve-Path -LiteralPath $Executable).Path
$isInstaller = $package.EndsWith("-Setup.exe", [StringComparison]::OrdinalIgnoreCase)
if ($VerifyShortcuts -and !$isInstaller) { throw "-VerifyShortcuts requires an installer" }
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

function Assert-Link([string]$Path, [bool]$Expected, [string]$Arguments) {
    $exists = Test-Path -LiteralPath $Path
    if ($exists -ne $Expected) { throw "Unexpected shortcut state: $Path (exists=$exists, expected=$Expected)" }
    if (!$exists) { return }
    $link = $shell.CreateShortcut($Path)
    if ($link.TargetPath -ne $app -or $link.Arguments -cne $Arguments -or $link.WorkingDirectory -ne $target) {
        throw "Unexpected shortcut at '$Path'. Target '$($link.TargetPath)' expected '$app'. Arguments '$($link.Arguments)' expected '$Arguments'. Working directory '$($link.WorkingDirectory)' expected '$target'."
    }
}

function Assert-TaskSelection([bool]$Desktop, [bool]$Autostart) {
    Assert-Link $desktopLink $Desktop ""
    $command = [Microsoft.Win32.Registry]::GetValue($startupKey, $startupName, $null)
    if ($Autostart) {
        if ($command -cne ('"' + $app + '" --background')) { throw "Incorrect sign-in command: $command" }
    } elseif ($null -ne $command) {
        throw "Unexpected sign-in entry: $startupName"
    }
}

function Install-ForTest([string]$Step, [string]$TaskArgument) {
    $arguments = @('/VERYSILENT', '/SUPPRESSMSGBOXES', '/SP-', '/NORESTART', '/NOICONS',
        ('/DIR="' + $target + '"'), ('/LOG="' + (Join-Path $sandbox "$Step.log") + '"'))
    if ($TaskArgument) { $arguments += $TaskArgument }
    Invoke-Setup $package $arguments
}

if ($isInstaller -and (Test-Path -LiteralPath $uninstallKey)) {
    throw "A native 1.2 installation is already registered. Test the installer on a clean profile."
}
if ($VerifyShortcuts) {
    $desktopLink = Join-Path ([Environment]::GetFolderPath('DesktopDirectory')) 'WX FileHelper.lnk'
    $startupKey = 'HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Run'
    $startupName = 'WXFileHelperDesktop.Native'
    $existingStartup = [Microsoft.Win32.Registry]::GetValue($startupKey, $startupName, $null)
    if ((Test-Path -LiteralPath $desktopLink) -or ($null -ne $existingStartup)) {
        throw "An application shortcut or sign-in entry already exists. Test in a clean Windows profile."
    }
    $shell = New-Object -ComObject WScript.Shell
}
New-Item -ItemType Directory -Force -Path $sandbox | Out-Null
try {
    $app = $package
    if ($isInstaller) {
        $app = Join-Path $target "WXFileHelper.exe"
        if ($VerifyShortcuts) {
            Install-ForTest 'setup' ''
            Assert-TaskSelection $true $false
            Install-ForTest 'startup-only' '/TASKS=autostart'
            Assert-TaskSelection $false $true
            Install-ForTest 'remember-tasks' ''
            Assert-TaskSelection $false $true
            Install-ForTest 'desktop-only' '/TASKS=desktopicon'
            Assert-TaskSelection $true $false
            Install-ForTest 'both' '/TASKS=desktopicon,autostart'
            Assert-TaskSelection $true $true
            Write-Output "Verified desktop shortcut, sign-in command, task memory, and removal on upgrade"
        } else {
            Install-ForTest 'setup' '/TASKS=""'
        }
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
            if ($VerifyShortcuts) { Assert-TaskSelection $false $false }
            Write-Output "Native Windows uninstall succeeded"
        }
    }
}

param(
    [Parameter(Mandatory = $true)][string]$Executable,
    [switch]$VerifyShortcuts,
    [string]$PreviousInstaller
)

$ErrorActionPreference = "Stop"
$package = (Resolve-Path -LiteralPath $Executable).Path
$isInstaller = $package.EndsWith("-Setup.exe", [StringComparison]::OrdinalIgnoreCase)
if ($VerifyShortcuts -and !$isInstaller) { throw "-VerifyShortcuts requires an installer" }
if ($PreviousInstaller) {
    if (!$VerifyShortcuts) { throw "-PreviousInstaller requires -VerifyShortcuts" }
    $PreviousInstaller = (Resolve-Path -LiteralPath $PreviousInstaller).Path
}
$root = (New-Item -ItemType Directory -Force -Path (Join-Path $PSScriptRoot "../../build/windows")).FullName
$sandbox = Join-Path $root ("smoke-" + [guid]::NewGuid().ToString("N"))
$target = Join-Path $sandbox ("app " + [char]0x6D4B + [char]0x8BD5 + [char]::ConvertFromUtf32(0x1F4DA))
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
    # WScript.Shell can return ANSI/8.3 path text when a profile's code page
    # cannot represent the target. Ask Explorer for the Unicode link instead.
    $item = $shell.NameSpace((Split-Path -Parent $Path)).ParseName((Split-Path -Leaf $Path))
    $link = $item.GetLink
    if (!$link) { throw "Shell could not read desktop shortcut: $Path" }
    $actualTarget = [string]$link.Path
    $actualDir = [string]$link.WorkingDirectory
    $actualArgs = [string]$link.Arguments
    if ($actualTarget -ne $app -or $actualArgs -cne $Arguments -or $actualDir -ne $target) {
        throw ("Incorrect desktop shortcut '$Path'." + [Environment]::NewLine +
            "Target   expected=[$app] actual=[$actualTarget]" + [Environment]::NewLine +
            "Work dir expected=[$target] actual=[$actualDir]" + [Environment]::NewLine +
            "Args     expected=[$Arguments] actual=[$actualArgs]")
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

function Install-ForTest([string]$Step, [string]$TaskArgument, [string]$Setup = $package) {
    $arguments = @('/VERYSILENT', '/SUPPRESSMSGBOXES', '/SP-', '/NORESTART', '/NOICONS',
        ('/DIR="' + $target + '"'), ('/LOG="' + (Join-Path $sandbox "$Step.log") + '"'))
    if ($TaskArgument) { $arguments += $TaskArgument }
    Invoke-Setup $Setup $arguments
}

function Uninstall-ForTest([string]$Step) {
    $uninstaller = Join-Path $target 'unins000.exe'
    if (Test-Path -LiteralPath $uninstaller) {
        Invoke-Setup $uninstaller @('/VERYSILENT', '/SUPPRESSMSGBOXES', '/NORESTART', ('/LOG="' + (Join-Path $sandbox "$Step.log") + '"'))
        if ((Test-Path -LiteralPath $app) -or (Test-Path -LiteralPath $uninstallKey)) {
            throw "Uninstall did not remove the test installation"
        }
    }
}

function Assert-ForeignShortcut([string]$Hash, [string]$Step) {
    if (!(Test-Path -LiteralPath $desktopLink) -or (Get-FileHash -LiteralPath $desktopLink).Hash -ne $Hash) {
        throw "The other application's shortcut was removed or changed during $Step"
    }
}

function Test-ForeignShortcutOwnership {
    # Create a known test link only after the clean-profile guard and regular
    # uninstall. Its bytes must survive installation, upgrade, and uninstall.
    $fixture = Join-Path $sandbox 'other-application.lnk'
    $writer = New-Object -ComObject WScript.Shell
    $sample = $writer.CreateShortcut($fixture)
    $sample.TargetPath = Join-Path $env:SystemRoot 'System32\notepad.exe'
    $sample.Save()
    $hash = (Get-FileHash -LiteralPath $fixture).Hash
    if (Test-Path -LiteralPath $desktopLink) { throw "Desktop path is already occupied" }
    try {
        Copy-Item -LiteralPath $fixture -Destination $desktopLink
        Install-ForTest 'pre-existing-selected' '/TASKS=desktopicon'
        Assert-ForeignShortcut $hash 'installation'
        Install-ForTest 'pre-existing-deselected' '/TASKS=!desktopicon'
        Assert-ForeignShortcut $hash 'task deselection'
        Uninstall-ForTest 'pre-existing-uninstall'
        Assert-ForeignShortcut $hash 'uninstall with a pre-existing link'
        Remove-Item -LiteralPath $desktopLink

        if ($PreviousInstaller) {
            Install-ForTest 'older-uninstall-log' '/TASKS=desktopicon' $PreviousInstaller
        }
        Install-ForTest 'owned-shortcut' '/TASKS=desktopicon'
        Assert-Link $desktopLink $true ''
        Copy-Item -LiteralPath $fixture -Destination $desktopLink -Force
        Uninstall-ForTest 'replaced-link-uninstall'
        Assert-ForeignShortcut $hash 'uninstall after a user replaced the link'
        Write-Output "Preserved pre-existing and later-replaced desktop shortcuts byte for byte"
    }
    finally {
        Uninstall-ForTest 'foreign-link-cleanup'
        if ((Test-Path -LiteralPath $desktopLink) -and (Get-FileHash -LiteralPath $desktopLink).Hash -eq $hash) {
            Remove-Item -LiteralPath $desktopLink
        }
    }
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
    $shell = New-Object -ComObject Shell.Application
}
New-Item -ItemType Directory -Force -Path $sandbox | Out-Null
try {
    $app = $package
    if ($isInstaller) {
        $app = Join-Path $target "WXFileHelper.exe"
        if ($VerifyShortcuts) {
            if ($PreviousInstaller) {
                Install-ForTest 'previous-setup' '' $PreviousInstaller
            } else {
                Install-ForTest 'setup' ''
            }
            Assert-TaskSelection $true $false
            $shortcutReport = Join-Path $sandbox 'desktop-launch.txt'
            $env:WXFILEHELPER_SMOKE_REPORT = $shortcutReport
            Invoke-Hidden $desktopLink @('--smoke-test')
            if (!(Test-Path -LiteralPath $shortcutReport) -or (Get-Content -LiteralPath $shortcutReport -Raw).Trim() -ne 'OK') {
                throw "Desktop shortcut did not start the native app: $desktopLink"
            }
            Write-Output "Started the native app through the desktop shortcut"
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
        Uninstall-ForTest 'uninstall'
        if ($VerifyShortcuts) { Assert-TaskSelection $false $false }
        Write-Output "Native Windows uninstall succeeded"
    }
}
if ($VerifyShortcuts) { Test-ForeignShortcutOwnership }

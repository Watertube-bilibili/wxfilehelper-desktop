param(
    [Parameter(Mandatory = $true)][string]$Executable,
    [string]$PreviousInstaller
)
$ErrorActionPreference = "Stop"
$program = (Resolve-Path -LiteralPath $Executable).Path
$installer = $program.EndsWith("-Setup.exe", [StringComparison]::OrdinalIgnoreCase)
if ($PreviousInstaller) {
    if (!$installer) { throw "-PreviousInstaller requires a -Setup.exe target" }
    $PreviousInstaller = (Resolve-Path -LiteralPath $PreviousInstaller).Path
}
$build = Join-Path (Split-Path -Parent $PSScriptRoot) "build"
$testRoot = Join-Path $build ("smoke-" + [guid]::NewGuid().ToString("N"))
$working = Join-Path $testRoot ("Qt " + [char]0x6D4B + [char]0x8BD5)
$temp = Join-Path $working "temp"
$installDir = Join-Path $working "installed app"
$report = Join-Path $testRoot "startup.txt"

function Test-NativeLaunch([string]$Path, [bool]$WithForeignQt) {
    if (Test-Path -LiteralPath $report) { Remove-Item -LiteralPath $report }
    $start = New-Object System.Diagnostics.ProcessStartInfo
    $start.FileName = $Path
    $start.Arguments = "--smoke-test"
    $start.UseShellExecute = $false
    $start.CreateNoWindow = $true
    $start.WindowStyle = [Diagnostics.ProcessWindowStyle]::Hidden
    $start.WorkingDirectory = $working
    $variables = $start.EnvironmentVariables
    foreach ($key in @($variables.Keys)) {
        if ($key -match '^(QT_|QML|PYTHON|PYSIDE|CONDA|VIRTUAL_ENV|_PYI_|PYINSTALLER_)') {
            $variables.Remove($key)
        }
    }
    $variables['PATH'] = "$env:SystemRoot\System32;$env:SystemRoot"
    $variables['TMP'] = $temp
    $variables['TEMP'] = $temp
    $variables['WXFILEHELPER_SMOKE_REPORT'] = $report
    if ($WithForeignQt) {
        $variables['QT_QPA_PLATFORM'] = 'unavailable'
        $variables['QT_PLUGIN_PATH'] = Join-Path $working "outside-qt"
        $variables['QT_QPA_PLATFORM_PLUGIN_PATH'] = Join-Path $working "outside-platforms"
    }
    $process = [Diagnostics.Process]::Start($start)
    try {
        if (!$process.WaitForExit(60000)) {
            # A PyInstaller single-file launcher owns a separate GUI child.
            & "$env:SystemRoot\System32\taskkill.exe" /PID $process.Id /T /F | Out-Null
            throw "Startup timed out: $Path (foreign Qt: $WithForeignQt)"
        }
        $result = if (Test-Path -LiteralPath $report) { Get-Content -LiteralPath $report -Raw } else { "No startup report" }
        if ($process.ExitCode -ne 0 -or $result.Trim() -ne "OK") {
            throw "Startup failed: $Path (exit $($process.ExitCode), foreign Qt: $WithForeignQt)`n$result"
        }
        Write-Output "Native Windows startup verified (foreign Qt: $WithForeignQt): $Path"
    }
    finally {
        $process.Dispose()
    }
}

function Invoke-Setup([string]$Path, [string[]]$Arguments) {
    $process = Start-Process -FilePath $Path -ArgumentList $Arguments -WorkingDirectory $working -WindowStyle Hidden -Wait -PassThru
    if ($process.ExitCode -ne 0) { throw "Setup operation failed: $Path (exit $($process.ExitCode)). Logs: $testRoot" }
}

function Assert-CleanInstallationProfile {
    # Inno's normal per-user installer registers an AppId, even with an explicit /DIR.
    # Use a fresh profile or CI rather than overwriting an existing user's uninstall entry.
    $key = 'Software\Microsoft\Windows\CurrentVersion\Uninstall\{1F1B99D3-7B37-4A07-9345-091DEDE09978}_is1'
    foreach ($hive in 'CurrentUser', 'LocalMachine') {
        foreach ($view in 'Registry32', 'Registry64') {
            $base = [Microsoft.Win32.RegistryKey]::OpenBaseKey($hive, $view)
            try {
                $entry = $base.OpenSubKey($key)
                if ($entry) {
                    $entry.Close()
                    throw "An installation is registered. Run the installer test on a clean Windows profile."
                }
            }
            finally { $base.Close() }
        }
    }
    $menu = Join-Path ([Environment]::GetFolderPath('Programs')) 'WX FileHelper Desktop'
    if (Test-Path -LiteralPath $menu) { throw "An existing Start menu folder is in use. Run on a clean Windows profile." }
}

if ($installer) { Assert-CleanInstallationProfile }
New-Item -ItemType Directory -Force -Path $temp | Out-Null
$passed = $false
try {
    if ($installer) {
        $argsCommon = @('/VERYSILENT', '/SUPPRESSMSGBOXES', '/SP-', '/NORESTART', '/NOICONS', ('/DIR="' + $installDir + '"'))
        if ($PreviousInstaller) {
            Invoke-Setup $PreviousInstaller ($argsCommon + ('/LOG="' + (Join-Path $testRoot 'previous.log') + '"'))
        }
        Invoke-Setup $program ($argsCommon + ('/LOG="' + (Join-Path $testRoot 'setup.log') + '"'))
        foreach ($retired in 'icuuc.dll', 'icudt78.dll') {
            if (Test-Path -LiteralPath (Join-Path $installDir "_internal\$retired")) {
                throw "Retired DLL survived installation: $retired"
            }
        }
        $program = Join-Path $installDir "WXFileHelper.exe"
    }
    Test-NativeLaunch $program $false
    Test-NativeLaunch $program $true
    $passed = $true
}
finally {
    if ($installer) {
        $uninstall = Join-Path $installDir "unins000.exe"
        if (Test-Path -LiteralPath $uninstall) {
            Invoke-Setup $uninstall @('/VERYSILENT', '/SUPPRESSMSGBOXES', '/NORESTART', ('/LOG="' + (Join-Path $testRoot 'uninstall.log') + '"'))
        } elseif ($passed) { throw "Uninstaller missing: $uninstall" }
        if ($passed -and (Test-Path -LiteralPath (Join-Path $installDir 'WXFileHelper.exe'))) { throw "Uninstall left the application executable in place" }
        if ($passed) { Assert-CleanInstallationProfile }
    }
    if ($passed) {
        $checked = (Resolve-Path -LiteralPath $testRoot).Path
        $parent = (Resolve-Path -LiteralPath $build).Path
        if ((Split-Path -Parent $checked) -ne $parent -or (Split-Path -Leaf $checked) -notmatch '^smoke-[a-f0-9]{32}$') {
            throw "Unexpected smoke directory: $checked"
        }
        Remove-Item -LiteralPath $checked -Recurse -Force
    } else {
        Write-Warning "Smoke test diagnostics retained in $testRoot"
    }
}

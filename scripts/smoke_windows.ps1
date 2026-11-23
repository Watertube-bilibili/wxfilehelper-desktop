param([Parameter(Mandatory=$true)][string]$Executable)
$ErrorActionPreference = "Stop"
$program = (Resolve-Path -LiteralPath $Executable).Path
$build = Join-Path (Split-Path -Parent $PSScriptRoot) "build"
New-Item -ItemType Directory -Force -Path $build | Out-Null
$report = Join-Path $build ("smoke-" + [guid]::NewGuid().ToString("N") + ".txt")
$savedReport = $env:WXFILEHELPER_SMOKE_REPORT
try {
    $env:WXFILEHELPER_SMOKE_REPORT = $report
    $process = Start-Process -FilePath $program -ArgumentList "--smoke-test" -WindowStyle Hidden -PassThru
    if (!$process.WaitForExit(45000)) {
        $process.Kill()
        throw "Startup timed out: $program"
    }
    $result = if (Test-Path -LiteralPath $report) { Get-Content -LiteralPath $report -Raw } else { "No startup report" }
    if ($process.ExitCode -ne 0 -or $result.Trim() -ne "OK") {
        throw "Startup failed: $program (exit $($process.ExitCode))`n$result"
    }
    Write-Output "Startup verified: $program"
}
finally {
    $env:WXFILEHELPER_SMOKE_REPORT = $savedReport
    if (Test-Path -LiteralPath $report) { Remove-Item -LiteralPath $report }
}

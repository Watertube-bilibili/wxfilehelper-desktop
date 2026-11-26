param(
    [string]$Go = "",
    [string]$Iscc = "",
    [string]$Upx = "",
    [string]$LocalSendArchive = "",
    [switch]$Verify
)

$ErrorActionPreference = "Stop"
$native = (Resolve-Path (Join-Path $PSScriptRoot "../..")).Path
$repo = Split-Path -Parent $native

function Find-Tool([string]$Configured, [string]$Name, [string[]]$Locations) {
    if ($Configured) { return (Resolve-Path -LiteralPath $Configured).Path }
    $command = Get-Command $Name -CommandType Application -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($command) { return $command.Source }
    foreach ($pattern in $Locations) {
        $file = Get-ChildItem -Path $pattern -File -ErrorAction SilentlyContinue |
            Sort-Object FullName -Descending | Select-Object -First 1
        if ($file) { return $file.FullName }
    }
    throw "$Name is required. Supply its path explicitly."
}

$Go = Find-Tool $Go "go.exe" @((Join-Path $repo ".tools/go/bin/go.exe"))
$Iscc = Find-Tool $Iscc "ISCC.exe" @(
    (Join-Path $repo ".tools/inno6/ISCC.exe"),
    "${env:ProgramFiles(x86)}/Inno Setup 6/ISCC.exe"
)
$versionLine = Select-String -LiteralPath (Join-Path $native "internal/desktop/controller.go") -Pattern '^const Version = "(\d+\.\d+\.\d+(?:-(?:alpha|beta|rc)\.\d+)?)"$'
if (!$versionLine) { throw "Could not read the native application version" }
$version = $versionLine.Matches[0].Groups[1].Value
$dist = (New-Item -ItemType Directory -Force -Path (Join-Path $native "dist/windows/$version")).FullName
$scratch = (New-Item -ItemType Directory -Force -Path (Join-Path $native "build/windows")).FullName
$portable = Join-Path $dist "UltraFile-$version-Windows-x64-Portable.exe"
$installer = Join-Path $dist "UltraFile-$version-Windows-x64-Setup.exe"
$obj = Join-Path $native "cmd/wxfilehelper/app_windows_amd64.syso"
$savedCgo = $env:CGO_ENABLED
$savedGOOS = $env:GOOS
$savedGOARCH = $env:GOARCH
$savedGOAMD64 = $env:GOAMD64
$savedGOCACHE = $env:GOCACHE
try {
    $env:CGO_ENABLED = "0"
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    $env:GOAMD64 = "v1"
    $env:GOCACHE = Join-Path $scratch "go-cache"
    Push-Location $PSScriptRoot
    try {
        & $Go run github.com/akavel/rsrc@v0.10.2 -arch amd64 -manifest app.manifest -ico app.ico -o $obj
        if ($LASTEXITCODE -ne 0) { throw "Windows resource compilation failed" }
    }
    finally { Pop-Location }

    Push-Location $native
    try {
        & $Go build -trimpath -buildvcs=false -ldflags "-H=windowsgui -s -w" -o $portable ./cmd/wxfilehelper
        if ($LASTEXITCODE -ne 0) { throw "Windows EXE build failed" }
    }
    finally { Pop-Location }
    if ($Verify) {
        & "$PSScriptRoot/smoke.ps1" -Executable $portable
    }

    $goRoot = (& $Go env GOROOT).Trim()
    if ($LASTEXITCODE -ne 0 -or !$goRoot) { throw "Could not find the Go license" }
    Copy-Item -LiteralPath (Join-Path $goRoot "LICENSE") -Destination (Join-Path $PSScriptRoot "go-license.txt") -Force
    $compilerArgs = @("/DAppVersion=$version", "/DAppHash=$((Get-FileHash $portable).Hash)")
    if ($LocalSendArchive) {
        $official = (Resolve-Path -LiteralPath $LocalSendArchive).Path
        if ((Get-FileHash -LiteralPath $official).Hash -ine '1b131a225e30bb46c7b4ffee23577716b2d0605108aded27899375033ff69abc') { throw 'Official LocalSend 1.19.0 archive checksum mismatch' }
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        $component = Join-Path $scratch ('localsend-package-' + [guid]::NewGuid().ToString('N'))
        [IO.Compression.ZipFile]::ExtractToDirectory($official,$component)
        $compilerArgs += ('"/DLocalSendDir=' + $component + '"')
    }
    $compilerArgs += 'setup.iss'
    $process = Start-Process -FilePath $Iscc -ArgumentList $compilerArgs `
        -WorkingDirectory $PSScriptRoot -WindowStyle Hidden -Wait -PassThru `
        -RedirectStandardOutput (Join-Path $scratch "iscc.log") -RedirectStandardError (Join-Path $scratch "iscc-error.log")
    if ($process.ExitCode -ne 0) {
        Get-Content (Join-Path $scratch "iscc.log"), (Join-Path $scratch "iscc-error.log") -Tail 20
        throw "Installer compilation failed"
    }
    if ($Verify) {
        & "$PSScriptRoot/smoke.ps1" -Executable $installer -VerifyShortcuts -VerifyLocalSend:([bool]$LocalSendArchive)
    }
    if ($Upx) {
        & $Upx --best --lzma $portable
        if ($LASTEXITCODE -ne 0) { throw "Portable compression failed" }
        & $Upx -t $portable
        if ($LASTEXITCODE -ne 0) { throw "Portable integrity check failed" }
        if ($Verify) { & "$PSScriptRoot/smoke.ps1" -Executable $portable }
    }
    @($portable, $installer) | ForEach-Object {
        $hash = (Get-FileHash -LiteralPath $_ -Algorithm SHA256).Hash.ToLowerInvariant()
        "$hash  $(Split-Path -Leaf $_)"
    } | Set-Content -LiteralPath (Join-Path $dist "SHA256SUMS.txt") -Encoding ascii
    $notices = @((Join-Path $goRoot "LICENSE"), (Join-Path $repo "licenses/go-qrcode-BSD.txt"), (Join-Path $repo "licenses/native-network-LICENSES.txt"), (Join-Path $repo "licenses/LocalSend-Apache-2.0.txt"), (Join-Path $repo "licenses/ItChat-MIT.txt"), (Join-Path $repo "licenses/WXFileHelper-MIT.txt"))
    Get-Content -LiteralPath $notices | Set-Content -LiteralPath (Join-Path $dist "DEPENDENCY-LICENSES.txt") -Encoding utf8
    Get-ChildItem -LiteralPath $portable,$installer,(Join-Path $dist "SHA256SUMS.txt") | Select-Object FullName,Length
}
finally {
    $env:CGO_ENABLED = $savedCgo
    $env:GOOS = $savedGOOS
    $env:GOARCH = $savedGOARCH
    $env:GOAMD64 = $savedGOAMD64
    $env:GOCACHE = $savedGOCACHE
}

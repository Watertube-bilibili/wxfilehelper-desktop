param(
    [Parameter(Mandatory=$true)][string]$Jdk,
    [Parameter(Mandatory=$true)][string]$BuildTools,
    [Parameter(Mandatory=$true)][string]$AndroidJar,
    [Parameter(Mandatory=$true)][string]$Zxing,
    [Parameter(Mandatory=$true)][string]$KeyStore,
    [string]$Alias = "androiddebugkey"
)
$ErrorActionPreference = "Stop"
function Invoke-Tool ($exe, [string[]]$arguments) {
    & $exe @arguments
    if ($LASTEXITCODE -ne 0) { throw "$exe exited with $LASTEXITCODE" }
}
$env:JAVA_HOME = (Resolve-Path $Jdk).Path
$bt = (Resolve-Path $BuildTools).Path
$platform = (Resolve-Path $AndroidJar).Path
$qr = (Resolve-Path $Zxing).Path
$key = (Resolve-Path $KeyStore).Path
$bin = Join-Path $env:JAVA_HOME "bin"
if (!$env:ULTRAFILE_STORE_PASSWORD) { throw "Set ULTRAFILE_STORE_PASSWORD for APK signing" }
$root = $PSScriptRoot
$work = (New-Item -ItemType Directory -Force (Join-Path $root ("build/"+[guid]::NewGuid().ToString("N")))).FullName
$dist = (New-Item -ItemType Directory -Force (Join-Path $root "dist")).FullName
$classes = (New-Item -ItemType Directory (Join-Path $work "classes")).FullName
$dex = (New-Item -ItemType Directory (Join-Path $work "dex")).FullName
$sources = (Get-ChildItem -Recurse -Path "$root/src" -Filter "*.java").FullName
Invoke-Tool "$bin/javac.exe" (@("-encoding","UTF-8","-source","8","-target","8","-classpath","$platform;$qr","-d",$classes)+$sources)
Invoke-Tool "$bin/jar.exe" @("cf","$work/classes.jar","-C",$classes,".")
Invoke-Tool "$bin/java.exe" @("-cp","$bt/lib/d8.jar","com.android.tools.r8.D8","--min-api","24","--lib",$platform,"--output",$dex,"$work/classes.jar",$qr)
Invoke-Tool "$bt/aapt2.exe" @("compile","--dir","$root/res","-o","$work/res.zip")
Invoke-Tool "$bt/aapt2.exe" @("link","-I",$platform,"--manifest","$root/AndroidManifest.xml","-o","$work/unsigned.apk","$work/res.zip")
Invoke-Tool "$bin/jar.exe" @("uf","$work/unsigned.apk","-C",$dex,"classes.dex","-C",$root,"NOTICE.txt")
Invoke-Tool "$bt/zipalign.exe" @("-f","-p","4","$work/unsigned.apk","$work/aligned.apk")
$apk = "$dist/UltraFile-2.0.0-Android.apk"
Invoke-Tool "$bin/java.exe" @("-jar","$bt/lib/apksigner.jar","sign","--ks",$key,"--ks-key-alias",$Alias,"--ks-pass","env:ULTRAFILE_STORE_PASSWORD","--out",$apk,"$work/aligned.apk")
Invoke-Tool "$bin/java.exe" @("-jar","$bt/lib/apksigner.jar","verify","--verbose",$apk)
Get-Item -LiteralPath $apk | Select-Object FullName,Length


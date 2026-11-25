#define Repo "https://github.com/Watertube-bilibili/wxfilehelper-desktop"
#define AppName "WX FileHelper Desktop"
#define AppPublisher "WX FileHelper Desktop contributors"

[Setup]
AppId={{6D2C9269-414D-46E2-8E62-A2C3DC74AACC}
AppName={#AppName}
AppVersion={#AppVersion}
AppPublisher={#AppPublisher}
AppPublisherURL={#Repo}
AppSupportURL={#Repo}/issues
AppUpdatesURL={#Repo}/releases
DefaultDirName={localappdata}\Programs\WXFileHelperNative
DefaultGroupName=WX FileHelper Desktop 1.2
DisableProgramGroupPage=yes
AllowNoIcons=yes
PrivilegesRequired=lowest
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
MinVersion=10.0.10240
LicenseFile=..\..\..\LICENSE
InfoBeforeFile=..\..\..\DISCLAIMER.md
UninstallDisplayIcon={app}\WXFileHelper.exe
SetupIconFile=app.ico
OutputDir=..\..\dist\windows
OutputBaseFilename=WXFileHelper-{#AppVersion}-Windows-x64-Setup
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
CloseApplications=yes
RestartApplications=no
SetupLogging=yes

[Messages]
WinVersionTooLowError=需要 64 位 Windows 10 1507（内部版本 10240）或更高版本。%nRequires x64 Windows 10 1507 (build 10240) or later.
OnlyOnTheseArchitectures=此包需要可运行 x64 应用的系统。%nThis installer requires x64 application support.%n%n%1
SetupFileCorrupt=安装包完整性检查失败。请重新下载并核对 SHA256SUMS.txt。%nInstaller integrity check failed. Download it again and verify SHA256SUMS.txt.

[Files]
Source: "..\..\dist\windows\WXFileHelper-{#AppVersion}-Windows-x64-Portable.exe"; DestDir: "{app}"; DestName: "WXFileHelper.exe"; Flags: ignoreversion
Source: "..\..\..\LICENSE"; DestDir: "{app}"
Source: "..\..\..\DISCLAIMER.md"; DestDir: "{app}"
Source: "..\..\..\THIRD_PARTY_NOTICES.md"; DestDir: "{app}"
Source: "..\..\..\licenses\ItChat-MIT.txt"; DestDir: "{app}\licenses"
Source: "..\..\..\licenses\WXFileHelper-MIT.txt"; DestDir: "{app}\licenses"
Source: "go-license.txt"; DestDir: "{app}\licenses"

[Icons]
Name: "{group}\{#AppName}"; Filename: "{app}\WXFileHelper.exe"
Name: "{group}\Uninstall {#AppName}"; Filename: "{uninstallexe}"

[Run]
Filename: "{app}\WXFileHelper.exe"; Description: "Launch / 启动 WX FileHelper Desktop"; Flags: nowait postinstall skipifsilent

#define AppVersion "0.1.0-beta.1"
#define Repo "https://github.com/Watertube-bilibili/wxfilehelper-desktop"
[Setup]
AppId={{1F1B99D3-7B37-4A07-9345-091DEDE09978}
AppName=WX FileHelper Desktop
AppVersion={#AppVersion}
AppVerName=WX FileHelper Desktop {#AppVersion}
AppPublisher=wxfilehelper-desktop contributors
AppPublisherURL={#Repo}
AppSupportURL={#Repo}/issues
AppUpdatesURL={#Repo}/releases
DefaultDirName={localappdata}\Programs\WXFileHelper
DefaultGroupName=WX FileHelper Desktop
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
MinVersion=10.0
LicenseFile=..\LICENSE
InfoBeforeFile=..\DISCLAIMER.md
UninstallDisplayIcon={app}\WXFileHelper.exe
SetupIconFile=..\build\app.ico
OutputDir=..\dist
OutputBaseFilename=WXFileHelper-{#AppVersion}-Windows-x64-Setup
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
CloseApplications=yes
RestartApplications=no

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "Create a desktop shortcut / 创建桌面快捷方式"; Flags: unchecked

[Files]
Source: "..\dist\WXFileHelper\*"; DestDir: "{app}"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "..\README.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\DISCLAIMER.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\LICENSE"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\THIRD_PARTY_NOTICES.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\licenses\*"; DestDir: "{app}\licenses"; Flags: ignoreversion recursesubdirs

[Icons]
Name: "{group}\WX FileHelper Desktop"; Filename: "{app}\WXFileHelper.exe"
Name: "{group}\Uninstall WX FileHelper Desktop"; Filename: "{uninstallexe}"
Name: "{autodesktop}\WX FileHelper Desktop"; Filename: "{app}\WXFileHelper.exe"; Tasks: desktopicon

[Run]
Filename: "{app}\WXFileHelper.exe"; Description: "Launch / 启动 WX FileHelper Desktop"; Flags: nowait postinstall skipifsilent

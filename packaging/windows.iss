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
AllowNoIcons=yes
PrivilegesRequired=lowest
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
MinVersion=10.0.17763
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
SetupLogging=yes

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Messages]
WinVersionTooLowError=需要 Windows 10 1809（内部版本 17763）或更高版本。%nRequires Windows 10 version 1809 (build 17763) or later.
OnlyOnTheseArchitectures=此安装包需要支持 x64 应用的 Windows。%nThis installer requires Windows with x64 application support.%n%n%1
SetupFileCorrupt=安装包完整性检查失败。请从项目 Release 重新下载安装包，并核对 SHA256SUMS.txt。%nThe installer failed its integrity check. Download it again from the project release and verify SHA256SUMS.txt.
LdrCannotCreateTemp=无法在临时目录创建文件。请检查可用空间和写入权限。%nSetup cannot create a temporary file. Check free space and write access in your temporary folder
LdrCannotExecTemp=无法在临时目录运行安装程序。若是学校或单位设备，请让管理员检查应用执行策略。%nSetup cannot run from your temporary folder. On a managed PC, ask the administrator to check the application execution policy

[Tasks]
Name: "desktopicon"; Description: "Create a desktop shortcut / 创建桌面快捷方式"; Flags: unchecked

[InstallDelete]
; The first preview installed a foreign ICU that shadows Windows ICU on upgrade.
Type: files; Name: "{app}\_internal\icuuc.dll"
Type: files; Name: "{app}\_internal\icudt78.dll"

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

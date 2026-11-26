#define Repo "https://github.com/Watertube-bilibili/wxfilehelper-desktop"
#define AppName "Ultra File"
#define AppPublisher "Ultra File contributors"
#define LinkName "WX FileHelper"
#define RunKey "Software\Microsoft\Windows\CurrentVersion\Run"
#define RunValue "WXFileHelperDesktop.Native"
#define LocalSendArchive "LocalSend-1.19.0-windows-x86-64.zip"
#define LocalSendSHA "1b131a225e30bb46c7b4ffee23577716b2d0605108aded27899375033ff69abc"

[Setup]
AppId={{6D2C9269-414D-46E2-8E62-A2C3DC74AACC}
AppName={#AppName}
AppVersion={#AppVersion}
AppPublisher={#AppPublisher}
AppPublisherURL={#Repo}
AppSupportURL={#Repo}/issues
AppUpdatesURL={#Repo}/releases
DefaultDirName={localappdata}\Programs\WXFileHelperNative
DefaultGroupName=Ultra File
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
OutputDir=..\..\dist\windows\{#AppVersion}
OutputBaseFilename=UltraFile-{#AppVersion}-Windows-x64-Setup
Compression=lzma2/ultra64
SolidCompression=yes
WizardStyle=modern
CloseApplications=yes
RestartApplications=no
SetupLogging=yes

[Messages]
WinVersionTooLowError=需要 64 位 Windows 10 1507（内部版本 10240）或更高版本。%nRequires x64 Windows 10 1507 (build 10240) or later.
OnlyOnTheseArchitectures=此包需要可运行 x64 应用的系统。%nThis installer requires x64 application support.%n%n%1
SetupFileCorrupt=安装包完整性检查失败。请重新下载并核对 SHA256SUMS.txt。%nInstaller integrity check failed. Download it again and verify SHA256SUMS.txt.

[Tasks]
Name: "localsend"; Description: "复制旁边的 LocalSend 官方离线包 / Copy the adjacent official LocalSend package"; GroupDescription: "传输服务 / Transfer services"; Check: ValidLocalSendArchive
Name: "desktopicon"; Description: "创建桌面快捷方式 / Create a desktop shortcut"; GroupDescription: "快捷方式 / Shortcuts"
Name: "autostart"; Description: "登录 Windows 后启动到托盘 / Start in the tray at sign-in"; GroupDescription: "启动 / Startup"; Flags: unchecked

[InstallDelete]
; An upgrade also applies a user's newly unchecked tasks to links from the last install.
Type: files; Name: "{userdesktop}\{#LinkName}.lnk"; Tasks: not desktopicon; Check: ManageLink('{userdesktop}')

[Files]
Source: "{src}\{#LocalSendArchive}"; DestDir: "{app}\components\LocalSend"; Flags: external ignoreversion skipifsourcedoesntexist; Tasks: localsend; Check: ValidLocalSendArchive
Source: "..\..\..\licenses\LocalSend-Apache-2.0.txt"; DestDir: "{app}\licenses"; Tasks: localsend
Source: "..\..\dist\windows\{#AppVersion}\UltraFile-{#AppVersion}-Windows-x64-Portable.exe"; DestDir: "{app}"; DestName: "WXFileHelper.exe"; Flags: ignoreversion; Check: PayloadNeedsInstall
Source: "..\..\..\LICENSE"; DestDir: "{app}"
Source: "..\..\..\DISCLAIMER.md"; DestDir: "{app}"
Source: "..\..\..\THIRD_PARTY_NOTICES.md"; DestDir: "{app}"
Source: "..\..\..\licenses\ItChat-MIT.txt"; DestDir: "{app}\licenses"
Source: "..\..\..\licenses\WXFileHelper-MIT.txt"; DestDir: "{app}\licenses"
Source: "..\..\..\licenses\go-qrcode-BSD.txt"; DestDir: "{app}\licenses"
Source: "..\..\..\licenses\native-network-LICENSES.txt"; DestDir: "{app}\licenses"
Source: "go-license.txt"; DestDir: "{app}\licenses"

[Icons]
Name: "{group}\{#AppName}"; Filename: "{app}\WXFileHelper.exe"
Name: "{group}\Uninstall {#AppName}"; Filename: "{uninstallexe}"
Name: "{userdesktop}\{#LinkName}"; Filename: "{app}\WXFileHelper.exe"; WorkingDir: "{app}"; Tasks: desktopicon; Check: ManageLink('{userdesktop}'); Flags: uninsneveruninstall

[Registry]
Root: HKCU; Subkey: "{#RunKey}"; ValueType: string; ValueName: "{#RunValue}"; ValueData: """{app}\WXFileHelper.exe"" --background"; Tasks: autostart; Flags: uninsdeletevalue
Root: HKCU; Subkey: "{#RunKey}"; ValueType: none; ValueName: "{#RunValue}"; Tasks: not autostart; Flags: dontcreatekey deletevalue; Check: OwnStartupEntry

[Run]
Filename: "{app}\WXFileHelper.exe"; Description: "Launch / 启动 Ultra File"; Flags: nowait postinstall skipifsilent

[Code]
var
  DesktopLink, SavedForeignLink: String;
  CheckedLocalSend, ValidLocalSend: Boolean;

function ValidLocalSendArchive: Boolean;
var
  Archive: String;
begin
  if not CheckedLocalSend then begin
    CheckedLocalSend := True;
    Archive := ExpandConstant('{src}\{#LocalSendArchive}');
    if FileExists(Archive) then
      ValidLocalSend := CompareText(GetSHA256OfFile(Archive), '{#LocalSendSHA}') = 0;
  end;
  Result := ValidLocalSend;
end;

function PayloadNeedsInstall: Boolean;
var
  Target: String;
begin
  Target := ExpandConstant('{app}\WXFileHelper.exe');
  Result := (not FileExists(Target)) or (CompareText(GetSHA256OfFile(Target), '{#AppHash}') <> 0);
end;

function OwnStartupEntry: Boolean;
var
  Current: String;
begin
  Result := RegQueryStringValue(HKCU, '{#RunKey}', '{#RunValue}', Current) and
    (CompareText(Current, ExpandConstant('"{app}\WXFileHelper.exe" --background')) = 0);
end;

function ManageLink(const Folder: String): Boolean;
var
  Filename: String;
  Shell, Directory, Item, Shortcut: Variant;
begin
  Filename := ExpandConstant(Folder + '\{#LinkName}.lnk');
  Result := not FileExists(Filename);
  if Result then Exit;
  try
    { IShellLinkDual.Path preserves Unicode in install paths on every locale. }
    Shell := CreateOleObject('Shell.Application');
    Directory := Shell.NameSpace(ExtractFileDir(Filename));
    Item := Directory.ParseName(ExtractFileName(Filename));
    Shortcut := Item.GetLink;
    Result := CompareText(Shortcut.Path, ExpandConstant('{app}\WXFileHelper.exe')) = 0;
  except
    Result := False;
  end;
  if not Result then Log('Preserving existing shortcut: ' + Filename);
end;

function InitializeUninstall: Boolean;
begin
  Result := True;
  DesktopLink := ExpandConstant('{userdesktop}\{#LinkName}.lnk');
  if FileExists(DesktopLink) and not ManageLink('{userdesktop}') then
  begin
    { Older releases appended unconditional shortcut deletions to the shared
      uninstall log. Keep the user's link in the private temporary directory
      in case one of those records is replayed. }
    SavedForeignLink := ExpandConstant('{tmp}\user-desktop-link.backup');
    if not CopyFile(DesktopLink, SavedForeignLink, True) then
    begin
      SuppressibleMsgBox('无法保留已有桌面快捷方式，卸载已停止。' + #13#10 +
        'Uninstall stopped because it could not preserve your existing desktop shortcut.',
        mbError, MB_OK, IDOK);
      Result := False;
    end;
  end;
end;

procedure CurUninstallStepChanged(Step: TUninstallStep);
begin
  if Step = usUninstall then
  begin
    { Check at uninstall time. A declarative [UninstallDelete] check would be
      evaluated at install time, before a user can replace their shortcut. }
    if FileExists(DesktopLink) and ManageLink('{userdesktop}') then
      DeleteFile(DesktopLink);
  end
  else if (Step = usPostUninstall) and (SavedForeignLink <> '') and not FileExists(DesktopLink) then
  begin
    if not CopyFile(SavedForeignLink, DesktopLink, True) then
      RaiseException('无法恢复桌面快捷方式 / Could not restore the desktop shortcut');
  end;
end;

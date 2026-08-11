#ifndef MyAppVersion
  #define MyAppVersion "0.0.0-dev"
#endif

#define MyAppName "AppAlias"
#define MyAppPublisher "AppAlias contributors"
#define MyAppURL "https://github.com/DylonH78/AppAlias"

[Setup]
AppId={{D44A8C1B-498D-4365-8FDD-FD0E7785D8E9}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}/issues
DefaultDirName={localappdata}\AppAlias
DefaultGroupName={#MyAppName}
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
OutputDir=..\dist
OutputBaseFilename=AppAlias-Setup-x64
Compression=lzma2
SolidCompression=yes
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
UninstallDisplayName=AppAlias

[Files]
Source: "..\dist\bin\appalias.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\dist\bin\launcher.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\dist\bin\appalias-gui.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\README.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\LICENSE"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{autoprograms}\AppAlias Manager"; Filename: "{app}\appalias-gui.exe"

[Run]
Filename: "{app}\appalias.exe"; Parameters: "init"; Flags: runhidden waituntilterminated; StatusMsg: "Configuring AppAlias for this user…"
Filename: "{app}\appalias.exe"; Parameters: "repair"; Flags: runhidden waituntilterminated; StatusMsg: "Refreshing AppAlias launchers…"

[UninstallRun]
Filename: "{app}\appalias.exe"; Parameters: "uninstall-path"; Flags: runhidden waituntilterminated

[UninstallDelete]
Type: filesandordirs; Name: "{app}\data"; Check: ShouldRemoveData
Type: filesandordirs; Name: "{app}\bin"; Check: ShouldRemoveData

[Code]
var
  RemoveData: Boolean;

procedure InitializeUninstallProgressForm();
begin
  RemoveData := MsgBox('Remove all saved aliases and AppAlias data as well?', mbConfirmation, MB_YESNO or MB_DEFBUTTON2) = IDYES;
end;

function ShouldRemoveData(): Boolean;
begin
  Result := RemoveData;
end;

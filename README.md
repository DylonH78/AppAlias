# AppAlias

AppAlias lets Windows users launch registered applications from PowerShell with
short names:

```powershell
appalias scan
appalias scan --apply
start 'wechat'
start 'code'
```

It is a Windows 10/11 x64 tool. It scans Start Menu shortcuts, App Paths, and
Microsoft Store entries; it never executes an app during scanning.

## Install

Download `AppAlias-Setup-x64.exe` from [GitHub Releases](https://github.com/DylonH78/AppAlias/releases). The per-user installer
installs into `%LOCALAPPDATA%\AppAlias` and adds its `bin` directory to the user
`PATH`. Open a new PowerShell window after installation.

The portable ZIP makes no changes until you run:

```powershell
.\appalias.exe init --portable
```

## Commands

```text
appalias init [--portable]
appalias scan [--json] [--apply]
appalias list
appalias add --name <alias> --target <program.exe> [--arg <argument>]
appalias rename <old> <new>
appalias remove <alias>
appalias launch <alias>
appalias doctor
appalias repair
appalias gui
```

`scan --apply` only creates safe, unique recommended aliases. Use `add` or the
GUI to choose another candidate name. Aliases may contain Unicode, but cannot
be Windows reserved file names or collide with an executable already on PATH.

## Security and support

Only `.exe` files and Microsoft Store AppUserModelIds are launch targets.
Scripts, URLs, and command-shell wrappers are intentionally excluded. Releases
include `SHA256SUMS.txt`; unsigned Windows downloads can initially show a
SmartScreen prompt.

## Development

Install Go, then run:

```powershell
go mod tidy
go test ./...
go vet ./...
go build ./cmd/appalias
go build ./cmd/appalias-launcher
```

The GUI is built with `go build ./cmd/appalias-gui`.

# AppAlias

AppAlias 让你在 Windows PowerShell 中通过短名称启动已安装应用：

```powershell
appalias scan
appalias scan --apply
start 'wechat'
start 'code'
```

它扫描开始菜单快捷方式、注册表 App Paths 和 Microsoft Store 应用。扫描只读取本机信息，不会启动任何应用。

## 安装与使用

从 GitHub Releases 下载 `AppAlias-Setup-x64.exe`。安装器为当前用户安装到 `%LOCALAPPDATA%\AppAlias` 并加入用户 PATH；完成后请重新打开 PowerShell。

便携版 ZIP 解压后不会修改系统。需要在解压目录执行：

```powershell
.\appalias.exe init --portable
```

常用命令：

```text
appalias scan [--json] [--apply]
appalias list
appalias add --name <别名> --target <程序.exe> [--arg <参数>]
appalias rename <旧名> <新名>
appalias remove <别名>
appalias doctor
appalias repair
appalias gui
```

`scan --apply` 只会创建安全且唯一的推荐别名。中文应用会生成原名称、完整拼音和拼音首字母候选。名称不能是 Windows 保留文件名，也不会覆盖 PATH 中已有同名程序。

## 安全与兼容性

首版支持 Windows 10/11 x64。为避免将命令包装器带入 PATH，只允许 `.exe` 和 Store AppUserModelId 作为启动目标；脚本、URL 和 `cmd.exe` / PowerShell 包装器会被拒绝。

发行包附带 SHA-256 校验和。首版未做代码签名，Windows SmartScreen 可能在首次下载时提示风险。

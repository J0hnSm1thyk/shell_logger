# shell_logger

シェルコマンドの実行履歴をJSON形式で記録するツールです。
ペネトレーションテスト中の操作ログを日付単位で自動保存します。

---

## ファイル構成

```
任意のフォルダ/
├── shell_logger.exe    # 実行ファイル
├── start_session.bat   # 起動用バッチファイル
├── README.md           # このファイル
└── 2026-05-03.json     # ログファイル（自動生成・日付単位）
```

---

## ビルド方法

### 必要なもの

- Go 1.21以上

### Linux MintからWindows向けにクロスコンパイル

```bash
# 標準ビルド
GOOS=windows GOARCH=amd64 go build -o shell_logger.exe main.go

# バイナリサイズを小さくする場合
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o shell_logger.exe main.go
```

---

## 使い方

### 方法1: バッチファイルから起動（推奨）

`start_session.bat` をダブルクリックするだけで起動します。

```bat
@echo off
chcp 65001 > nul
title Shell Logger Session

cd /d "%~dp0"

if not exist shell_logger.exe (
    echo [ERROR] shell_logger.exe が見つかりません
    pause
    exit /b 1
)

shell_logger.exe cmd
pause
```

### 方法2: コマンドラインから起動

```powershell
# cmd.exeモードでインタラクティブセッション開始
.\shell_logger.exe cmd

# PowerShellモードでインタラクティブセッション開始
.\shell_logger.exe ps

# 単発実行モード
.\shell_logger.exe cmd "ipconfig /all"
.\shell_logger.exe ps "Get-Process"
```

---

## セッション中の操作

| 入力 | 動作 |
|------|------|
| 任意のコマンド | 実行して結果を表示・ログに記録 |
| `!cmd` | cmd.exe に切り替え |
| `!ps` | PowerShell に切り替え |
| `exit` | セッション終了 |

### 実行例

```
[shell_logger] セッション開始 (shell=cmd)
[shell_logger] ログ: C:\tools\2026-05-03.json
[shell_logger] 終了するには exit と入力してください
[shell_logger] シェル切り替え: !cmd または !ps

C:\tools> whoami
desktop-xxx\tanaka

C:\tools> cd C:\Users\tanaka\Documents

C:\Users\tanaka\Documents> ipconfig
...

C:\Users\tanaka\Documents> !ps
[shell_logger] PowerShell に切り替えました

PS C:\Users\tanaka\Documents> Get-Process
...

PS C:\Users\tanaka\Documents> exit
[shell_logger] セッション終了
```

---

## ログファイル

### 保存先

`shell_logger.exe` と同じフォルダに日付単位で保存されます。

```
2026-05-03.json
2026-05-04.json   # 日付をまたいだ場合は自動で新ファイル
```

### ログのフォーマット（1コマンド = 1行JSON）

```json
{
  "timestamp": "2026-05-03T10:00:00+09:00",
  "hostname": "DESKTOP-XXX",
  "user": "tanaka",
  "shell": "cmd.exe",
  "command": "whoami",
  "start_time": "2026-05-03T10:00:00+09:00",
  "end_time": "2026-05-03T10:00:00+09:00",
  "duration_ms": 42,
  "exit_code": 0,
  "stdout": "desktop-xxx\\tanaka",
  "stderr": "",
  "error": ""
}
```

### シェル切り替えのログ

シェル切り替え（`!cmd` / `!ps`）も記録されます。

```json
{
  "shell": "shell_logger",
  "command": "[switch] cmd.exe -> powershell",
  ...
}
```

---

## ログの確認方法

PowerShellで読むと見やすいです。

```powershell
# 全コマンド一覧
Get-Content 2026-05-03.json | ConvertFrom-Json | Select-Object timestamp, command, exit_code

# 失敗したコマンドだけ抽出（exit_code != 0）
Get-Content 2026-05-03.json | ConvertFrom-Json | Where-Object { $_.exit_code -ne 0 }

# 特定コマンドの出力確認
Get-Content 2026-05-03.json | ConvertFrom-Json | Where-Object { $_.command -like "*ipconfig*" } | Select-Object command, stdout
```

---

## 注意事項

### 非対応のコマンド

以下のコマンドは正常に動作しません。

| コマンド例 | 理由 |
|------------|------|
| `more`、`pause`、`choice` | ユーザー入力待ちが発生しハングする |
| `set FOO=bar` | 環境変数が次のコマンドに引き継がれない |

### ログの注意

- `stdout` / `stderr` はコマンドの出力をそのまま記録します
- ログファイル自体には暗号化・難読化はありません
- 機密情報を含むコマンドの出力も平文で記録されます

---

## 動作環境

| 項目 | 要件 |
|------|------|
| OS | Windows 10 / 11 |
| アーキテクチャ | x86-64 |
| PowerShell | オプション（`!ps`使用時） |
| 管理者権限 | 不要（通常コマンドのみ実行の場合） |
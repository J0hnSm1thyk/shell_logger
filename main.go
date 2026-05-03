package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type LogEntry struct {
	Timestamp  string `json:"timestamp"`
	Hostname   string `json:"hostname"`
	User       string `json:"user"`
	Shell      string `json:"shell"`
	Command    string `json:"command"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
	DurationMs int64  `json:"duration_ms"`
	ExitCode   int    `json:"exit_code"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	Error      string `json:"error"`
}

func detectPowerShell() string {
	_, err := exec.LookPath("pwsh.exe")
	if err == nil {
		return "pwsh.exe"
	}
	return "powershell.exe"
}

func normalizeOutput(b []byte) string {
	s := string(b)
	s = strings.ReplaceAll(s, "\x00", "")
	return strings.TrimSpace(s)
}

func runCommand(shell string, command string) LogEntry {
	start := time.Now()
	var cmd *exec.Cmd
	var actualShell string

	command = strings.TrimSpace(command)
	if command == "" {
		return LogEntry{}
	}

	if shell == "ps" {
		ps := detectPowerShell()
		actualShell = ps
		cmd = exec.Command(ps, "-NoProfile", "-NonInteractive", "-Command", command)
	} else {
		actualShell = "cmd.exe"
		cmd = exec.Command("cmd.exe", "/c", command)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	end := time.Now()

	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}

	hostname, _ := os.Hostname()
	user := os.Getenv("USERNAME")

	entry := LogEntry{
		Timestamp:  time.Now().Format(time.RFC3339),
		Hostname:   hostname,
		User:       user,
		Shell:      actualShell,
		Command:    command,
		StartTime:  start.Format(time.RFC3339),
		EndTime:    end.Format(time.RFC3339),
		DurationMs: end.Sub(start).Milliseconds(),
		ExitCode:   exitCode,
		Stdout:     normalizeOutput(stdoutBuf.Bytes()),
		Stderr:     normalizeOutput(stderrBuf.Bytes()),
	}
	if err != nil {
		entry.Error = err.Error()
	}
	return entry
}

func writeLog(entry LogEntry, logfile string) error {
	// 空エントリはスキップ
	if entry.Command == "" {
		return nil
	}
	f, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	jsonData, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = f.WriteString(string(jsonData) + "\n")
	return err
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  shell_logger.exe <cmd|ps>         # インタラクティブセッション")
		fmt.Println("  shell_logger.exe <cmd|ps> \"cmd\"   # 単発実行")
		return
	}

	shell := os.Args[1]
	logfile := "shell_log.json"

	// 単発実行モード（引数あり）
	if len(os.Args) >= 3 {
		command := os.Args[2]
		entry := runCommand(shell, command)
		writeLog(entry, logfile)
		fmt.Println(entry.Stdout)
		if entry.Stderr != "" {
			fmt.Fprintln(os.Stderr, entry.Stderr)
		}
		return
	}

	// ─────────────────────────────────────────
	// インタラクティブセッションモード
	// stdinからコマンドを1行ずつ読み取り、実行・記録する
	// ─────────────────────────────────────────
	fmt.Printf("[shell_logger] セッション開始 (shell=%s, log=%s)\n", shell, logfile)
	fmt.Println("[shell_logger] 終了するには 'exit' と入力してください")

	scanner := bufio.NewScanner(os.Stdin)

	for {
		// プロンプト表示
		if shell == "ps" {
			fmt.Print("PS> ")
		} else {
			fmt.Print("CMD> ")
		}

		if !scanner.Scan() {
			// EOF（Ctrl+Z / Ctrl+D）
			break
		}

		line := scanner.Text()

		if strings.TrimSpace(line) == "exit" {
			fmt.Println("[shell_logger] セッション終了")
			break
		}

		entry := runCommand(shell, line)

		if err := writeLog(entry, logfile); err != nil {
			fmt.Fprintln(os.Stderr, "Log write error:", err)
		}

		// 結果を表示
		if entry.Stdout != "" {
			fmt.Println(entry.Stdout)
		}
		if entry.Stderr != "" {
			fmt.Fprintln(os.Stderr, entry.Stderr)
		}
		if entry.Error != "" && entry.ExitCode != 0 {
			fmt.Fprintf(os.Stderr, "[exit code %d]\n", entry.ExitCode)
		}
	}
}
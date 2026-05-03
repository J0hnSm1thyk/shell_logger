package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
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
	// CP932（Shift-JIS）→ UTF-8変換を試みる
	decoder := japanese.ShiftJIS.NewDecoder()
	decoded, _, err := transform.Bytes(decoder, b)
	if err != nil {
		decoded = b
	}
	s := string(decoded)
	s = strings.ReplaceAll(s, "\x00", "")
	return strings.TrimSpace(s)
}

func getLogPath() string {
	exePath, err := os.Executable()
	if err != nil {
		today := time.Now().Format("2006-01-02")
		return today + ".json"
	}
	exeDir := filepath.Dir(exePath)
	today := time.Now().Format("2006-01-02")
	return filepath.Join(exeDir, today+".json")
}

func getHostname() string {
	h, _ := os.Hostname()
	return h
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
		// OutputEncodingをUTF-8に強制
		psCommand := "[Console]::OutputEncoding = [System.Text.Encoding]::UTF8; " + command
		cmd = exec.Command(ps, "-NoProfile", "-NonInteractive", "-Command", psCommand)
	} else {
		actualShell = "cmd.exe"
		// chcp 65001でUTF-8に統一してからコマンド実行
		cmd = exec.Command("cmd.exe", "/c", "chcp 65001 > nul && "+command)
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

	entry := LogEntry{
		Timestamp:  time.Now().Format(time.RFC3339),
		Hostname:   getHostname(),
		User:       os.Getenv("USERNAME"),
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

func writeSwitchLog(logfile string, fromShell string, toShell string) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Hostname:  getHostname(),
		User:      os.Getenv("USERNAME"),
		Shell:     "shell_logger",
		Command:   fmt.Sprintf("[switch] %s -> %s", fromShell, toShell),
		StartTime: time.Now().Format(time.RFC3339),
		EndTime:   time.Now().Format(time.RFC3339),
	}
	writeLog(entry, logfile)
}

func writeLog(entry LogEntry, logfile string) error {
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
		fmt.Println("  shell_logger.exe <cmd|ps>        # インタラクティブセッション")
		fmt.Println("  shell_logger.exe <cmd|ps> \"cmd\"  # 単発実行")
		return
	}

	shell := os.Args[1]

	// 単発実行モード
	if len(os.Args) >= 3 {
		command := os.Args[2]
		logfile := getLogPath()
		entry := runCommand(shell, command)
		writeLog(entry, logfile)
		fmt.Println(entry.Stdout)
		if entry.Stderr != "" {
			fmt.Fprintln(os.Stderr, entry.Stderr)
		}
		return
	}

	// インタラクティブセッションモード
	logfile := getLogPath()
	fmt.Printf("[shell_logger] セッション開始 (shell=%s)\n", shell)
	fmt.Printf("[shell_logger] ログ: %s\n", logfile)
	fmt.Println("[shell_logger] 終了するには exit と入力してください")
	fmt.Println("[shell_logger] シェル切り替え: !cmd または !ps")

	scanner := bufio.NewScanner(os.Stdin)

	for {
		if shell == "ps" {
			fmt.Print("PS> ")
		} else {
			fmt.Print("CMD> ")
		}

		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		if line == "exit" {
			fmt.Println("[shell_logger] セッション終了")
			break
		}

		if line == "!ps" {
			if shell == "ps" {
				fmt.Println("[shell_logger] すでに PowerShell です")
				continue
			}
			logfile = getLogPath()
			writeSwitchLog(logfile, "cmd.exe", "powershell")
			shell = "ps"
			fmt.Println("[shell_logger] PowerShell に切り替えました")
			continue
		}

		if line == "!cmd" {
			if shell == "cmd" {
				fmt.Println("[shell_logger] すでに cmd.exe です")
				continue
			}
			logfile = getLogPath()
			writeSwitchLog(logfile, "powershell", "cmd.exe")
			shell = "cmd"
			fmt.Println("[shell_logger] cmd.exe に切り替えました")
			continue
		}

		logfile = getLogPath()
		entry := runCommand(shell, line)

		if err := writeLog(entry, logfile); err != nil {
			fmt.Fprintln(os.Stderr, "Log write error:", err)
		}

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
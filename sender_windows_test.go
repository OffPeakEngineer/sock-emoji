package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSendInputStreamsBeforeEOF(t *testing.T) {
	path := fmt.Sprintf(`\\.\pipe\sock-emoji-stream-%d-%d`, os.Getpid(), time.Now().UnixNano())
	listener, err := listenPipe(path)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	updates := make(chan string, 1)
	go servePipe(listener, updates)
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	result := make(chan error, 1)
	go func() { result <- sendInput(path, reader) }()
	if _, err := io.WriteString(writer, "🧑‍💻\n"); err != nil {
		t.Fatal(err)
	}
	waitForUpdate(t, updates, "🧑‍💻") // Producer is still connected.
	if _, err := io.WriteString(writer, "✅"); err != nil {
		t.Fatal(err)
	}
	writer.Close()
	waitForUpdate(t, updates, "✅") // Final unterminated message.
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("sender did not exit")
	}
}

func waitForUpdate(t *testing.T, updates <-chan string, want string) {
	t.Helper()
	select {
	case got := <-updates:
		if got != want {
			t.Fatalf("received %q, want %q", got, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no forwarded update")
	}
}

// Exercise the actual GUI-subsystem executable: stdin inheritance differs
// from the console-subsystem test runner, especially through PowerShell.
func TestGUIPipedInput(t *testing.T) {
	executable := filepath.Join(t.TempDir(), "sock-emoji.exe")
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	build := exec.CommandContext(ctx, "go", "build", "-ldflags=-H=windowsgui", "-o", executable, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
	path := fmt.Sprintf(`\\.\pipe\sock-emoji-gui-sender-%d-%d`, os.Getpid(), time.Now().UnixNano())
	listener, err := listenPipe(path)
	if err != nil {
		t.Fatal(err)
	}
	updates := make(chan string, 1)
	go servePipe(listener, updates)
	defer listener.Close()

	for _, shell := range []string{"powershell.exe", "pwsh.exe"} {
		t.Run(shell, func(t *testing.T) {
			shellPath, err := exec.LookPath(shell)
			if err != nil {
				t.Skip("shell unavailable")
			}
			command := `echo ([char]::ConvertFromUtf32(0x1F9E6)) | & $env:SOCK_EMOJI_TEST_EXE -pipe $env:SOCK_EMOJI_TEST_PIPE; exit $LASTEXITCODE`
			if shell == "powershell.exe" {
				// PowerShell 5.1 otherwise replaces Unicode with ASCII question marks
				// before the receiving executable ever sees the input.
				command = `$OutputEncoding = [System.Text.UTF8Encoding]::new($false); ` + command
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, shellPath, "-NoProfile", "-NonInteractive", "-Command", command)
			cmd.Env = append(os.Environ(), "SOCK_EMOJI_TEST_EXE="+executable, "SOCK_EMOJI_TEST_PIPE="+path)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("pipeline: %v\n%s", err, output)
			}
			waitForUpdate(t, updates, "🧦")
		})
	}

	t.Run("redirected file", func(t *testing.T) {
		file, err := os.CreateTemp(t.TempDir(), "input")
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		if _, err := file.WriteString("\ufeff✨"); err != nil {
			t.Fatal(err)
		}
		if _, err := file.Seek(0, 0); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, executable, "-pipe", path)
		cmd.Stdin = file
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("file input: %v\n%s", err, output)
		}
		waitForUpdate(t, updates, "✨")
	})

	listener.Close()
	t.Run("missing tray instance", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, executable, "-pipe", path)
		cmd.Stdin = strings.NewReader("✅\n")
		output, err := cmd.CombinedOutput()
		if ctx.Err() != nil {
			t.Fatal("sender hung without a tray instance")
		}
		if err == nil {
			t.Fatal("sender succeeded without a tray instance")
		}
		if !strings.Contains(string(output), "-listen first") {
			t.Fatalf("missing startup hint: %s", output)
		}
	})
}

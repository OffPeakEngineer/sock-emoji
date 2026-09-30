package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode"
)

type tickerConfig struct {
	width int
	delay time.Duration
}

func main() {
	path := flag.String("pipe", defaultPipe, "pipe to receive newline-delimited UTF-8 updates")
	listen := flag.Bool("listen", false, "start the tray app even when stdin is redirected")
	config := tickerConfig{width: 1, delay: 200 * time.Millisecond}
	if runtime.GOOS == "darwin" {
		flag.IntVar(&config.width, "width", 1, "number of visible Unicode clusters at a time")
		flag.DurationVar(&config.delay, "delay", 200*time.Millisecond, "delay between ticker updates")
	}
	flag.Parse()
	config.width = max(1, config.width)
	config.delay = maxDuration(10*time.Millisecond, config.delay)
	pipePath, err := expandPath(*path)
	if err == nil {
		if !*listen && hasRedirectedInput(os.Stdin) {
			err = sendInput(pipePath, os.Stdin)
		} else {
			err = runPlatform(pipePath, config)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func hasRedirectedInput(input *os.File) bool {
	info, err := input.Stat()
	// Explorer-launched Windows GUI apps may have no stdin handle. A console
	// or /dev/null also means normal startup; only a pipe or file means sender.
	return err == nil && (info.Mode()&os.ModeNamedPipe != 0 || info.Mode().IsRegular())
}

func sendInput(path string, input io.Reader) error {
	conn, err := dialUpdatePipe(path)
	if err != nil {
		return fmt.Errorf("connect to %s: %w (start sock-emoji -listen first)", path, err)
	}
	// Stream immediately so long-running producers need not exit to update the
	// icon. Closing the connection also delivers a final unterminated line.
	_, copyErr := io.Copy(conn, input)
	// Windows pipe writes can finish while bytes are still buffered. Drain them
	// to the reader before closing this short-lived sender's pipe handle.
	if flusher, ok := conn.(interface{ Flush() error }); ok && copyErr == nil {
		copyErr = flusher.Flush()
	}
	closeErr := conn.Close()
	if copyErr != nil {
		return fmt.Errorf("forward stdin: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close sender: %w", closeErr)
	}
	return nil
}

// Concurrent writers never block; newer status replaces pending status.
func sendLatest(updates chan string, value string) {
	for {
		select {
		case updates <- value:
			return
		default:
		}
		select {
		case <-updates:
		default:
		}
	}
}

func readUpdates(r io.Reader, updates chan string) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024), 1024*1024)
	first := true
	for scanner.Scan() {
		line := scanner.Text()
		if first {
			line = strings.TrimPrefix(line, "\ufeff")
			first = false
		}
		value := strings.TrimSpace(strings.ToValidUTF8(line, ""))
		if value != "" && !strings.ContainsRune(value, 0) {
			sendLatest(updates, value)
		}
	}
	return scanner.Err()
}

func iconText(value string) string {
	clusters := splitClusters(value)
	if len(clusters) == 0 {
		return ""
	}
	return clusters[0]
}
func expandPath(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
	}
	return path, nil
}

func tickerFrames(clusters []string, width int) []string {
	width = max(1, width)
	if len(clusters) <= width {
		return []string{strings.Join(clusters, "")}
	}

	frames := make([]string, 0, len(clusters)-width+1)
	for i := 0; i <= len(clusters)-width; i++ {
		frames = append(frames, strings.Join(clusters[i:i+width], ""))
	}
	return frames
}

func splitClusters(value string) []string {
	runes := []rune(value)
	clusters := make([]string, 0, len(runes))

	for i := 0; i < len(runes); {
		start := i
		i++

		if isRegionalIndicator(runes[start]) && i < len(runes) && isRegionalIndicator(runes[i]) {
			i++
		}

		for i < len(runes) {
			switch {
			case isClusterExtender(runes[i]):
				i++
			case runes[i] == '\u200d' && i+1 < len(runes):
				i += 2
				for i < len(runes) && isClusterExtender(runes[i]) {
					i++
				}
			default:
				clusters = append(clusters, string(runes[start:i]))
				goto nextCluster
			}
		}

		clusters = append(clusters, string(runes[start:i]))

	nextCluster:
	}

	return clusters
}

func isClusterExtender(r rune) bool {
	return unicode.Is(unicode.M, r) ||
		(r >= '\ufe00' && r <= '\ufe0f') ||
		(r >= '\U0001f3fb' && r <= '\U0001f3ff') ||
		r == '\u20e3'
}

func isRegionalIndicator(r rune) bool {
	return r >= '\U0001f1e6' && r <= '\U0001f1ff'
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

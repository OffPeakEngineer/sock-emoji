package main

/*
#cgo darwin LDFLAGS: -framework Cocoa
#include <stdlib.h>
#include "statusbar_darwin.h"
*/
import "C"

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unicode"
	"unsafe"
)

type tickerConfig struct {
	width int
	delay time.Duration
}

func main() {
	runtime.LockOSThread()

	pathFlag := flag.String("pipe", "~/sock", "FIFO path to receive text updates (echo \"🧦\" > ~/sock)")
	widthFlag := flag.Int("width", 1, "number of visible Unicode clusters at a time")
	delayFlag := flag.Duration("delay", 200*time.Millisecond, "delay between ticker updates")
	flag.Parse()

	config := tickerConfig{
		width: max(1, *widthFlag),
		delay: maxDuration(10*time.Millisecond, *delayFlag),
	}

	pipePath, err := expandPath(*pathFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to expand path: %v\n", err)
		os.Exit(1)
	}

	if err := preparePipe(pipePath); err != nil {
		fmt.Fprintf(os.Stderr, "unable to prepare pipe %s: %v\n", pipePath, err)
		os.Exit(1)
	}

	initialTitle := C.CString("🧦")
	C.initStatusBar(initialTitle)
	C.free(unsafe.Pointer(initialTitle))

	updates := make(chan string, 1)
	go watchPipe(pipePath, updates)
	go runTicker(updates, config)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		C.stopApp()
	}()

	C.runApp()
	C.cleanupStatusBar()
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

func preparePipe(path string) error {
	info, err := os.Lstat(path)
	if err == nil {
		if info.Mode()&os.ModeNamedPipe == 0 {
			return fmt.Errorf("path exists and is not a FIFO")
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	return syscall.Mkfifo(path, 0600)
}

func watchPipe(path string, updates chan string) {
	for {
		f, err := os.OpenFile(path, os.O_RDONLY, 0600)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open pipe failed: %v\n", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 0, 1024), 1024*1024)
		for scanner.Scan() {
			value := strings.TrimSpace(scanner.Text())
			if value != "" {
				sendLatest(updates, value)
			}
		}
		err = scanner.Err()
		f.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "read pipe failed: %v\n", err)
			time.Sleep(200 * time.Millisecond)
			continue
		}
	}
}

func sendLatest(updates chan string, value string) {
	select {
	case updates <- value:
	default:
		select {
		case <-updates:
		default:
		}
		updates <- value
	}
}

func runTicker(updates <-chan string, config tickerConfig) {
	for {
		value, ok := <-updates
		if !ok {
			return
		}

	animate:
		clusters := splitClusters(strings.ToValidUTF8(value, ""))
		if len(clusters) == 0 {
			continue
		}

		frames := tickerFrames(clusters, config.width)
		for i, frame := range frames {
			updateStatus(frame)

			if i == len(frames)-1 || config.delay <= 0 {
				continue
			}

			timer := time.NewTimer(config.delay)
			select {
			case next, ok := <-updates:
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				if !ok {
					return
				}
				value = next
				goto animate
			case <-timer.C:
			}
		}
	}
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

func updateStatus(value string) {
	value = strings.ToValidUTF8(value, "")
	cstr := C.CString(value)
	defer C.free(unsafe.Pointer(cstr))
	C.setStatusBarTitle(cstr)
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

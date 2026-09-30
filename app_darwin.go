package main

/*
#cgo darwin LDFLAGS: -framework Cocoa
#include <stdlib.h>
#include "statusbar_darwin.h"
*/
import "C"

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const defaultPipe = "~/sock"

func runPlatform(pipePath string, config tickerConfig) error {
	runtime.LockOSThread()

	if err := preparePipe(pipePath); err != nil {
		fmt.Fprintf(os.Stderr, "unable to prepare pipe %s: %v\n", pipePath, err)
		return err
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
	return nil
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

		err = readUpdates(f, updates)
		f.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "read pipe failed: %v\n", err)
			time.Sleep(200 * time.Millisecond)
			continue
		}
	}
}

func updateStatus(value string) {
	value = strings.ToValidUTF8(value, "")
	cstr := C.CString(value)
	defer C.free(unsafe.Pointer(cstr))
	C.setStatusBarTitle(cstr)
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

package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows"
)

const defaultPipe = `\\.\pipe\sock-emoji`

func validatePipePath(path string) error {
	const prefix = `\\.\pipe\`
	if !strings.HasPrefix(strings.ToLower(path), prefix) || len(path) == len(prefix) || strings.Contains(path[len(prefix):], `\`) {
		return fmt.Errorf("pipe must be a local Windows pipe name such as %s", defaultPipe)
	}
	return nil
}

func dialUpdatePipe(path string) (io.WriteCloser, error) {
	if err := validatePipePath(path); err != nil {
		return nil, err
	}
	timeout := 2 * time.Second
	return winio.DialPipe(path, &timeout)
}

func listenPipe(path string) (net.Listener, error) {
	if err := validatePipePath(path); err != nil {
		return nil, err
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, err
	}
	// Match the macOS FIFO's owner-only permissions. No network or other-user access.
	return winio.ListenPipe(path, &winio.PipeConfig{
		SecurityDescriptor: "D:P(A;;GA;;;" + user.User.Sid.String() + ")",
		InputBufferSize:    4096,
		OutputBufferSize:   4096,
	})
}

func servePipe(listener net.Listener, updates chan string) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		// An idle producer must not prevent other scripts from reporting status.
		go func() {
			defer conn.Close()
			if err := readUpdates(conn, updates); err != nil {
				fmt.Fprintln(os.Stderr, "read pipe:", err)
			}
		}()
	}
}

package main

import (
	"fmt"
	"io"
	"os"
	"syscall"
)

func dialUpdatePipe(path string) (io.WriteCloser, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeNamedPipe == 0 {
		return nil, fmt.Errorf("path is not a FIFO")
	}
	// A missing reader must fail rather than hang a script indefinitely.
	return os.OpenFile(path, os.O_WRONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
}

//go:build windows

package main

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Microsoft/go-winio"
	"github.com/lxn/win"
)

func TestNamedPipeUpdates(t *testing.T) {
	path := fmt.Sprintf(`\\.\pipe\sock-emoji-test-%d-%d`, os.Getpid(), time.Now().UnixNano())
	listener, err := listenPipe(path)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	updates := make(chan string, 1)
	go servePipe(listener, updates)
	// Reject a second server rather than silently stealing producers.
	if duplicate, err := listenPipe(path); err == nil {
		duplicate.Close()
		t.Fatal("duplicate listener succeeded")
	}
	timeout := 2 * time.Second
	idle, err := winio.DialPipe(path, &timeout)
	if err != nil {
		t.Fatal(err)
	}
	defer idle.Close()
	for _, value := range []string{"🧦", "🧑‍💻", "✅"} {
		conn, err := winio.DialPipe(path, &timeout)
		if err != nil {
			t.Fatal(err)
		}
		// Split UTF-8 across writes to exercise stream framing.
		bytes := []byte(value + "\r\n")
		if _, err := conn.Write(bytes[:1]); err != nil {
			t.Fatal(err)
		}
		if _, err := conn.Write(bytes[1:]); err != nil {
			t.Fatal(err)
		}
		if err := conn.(winio.PipeConn).Flush(); err != nil {
			t.Fatal(err)
		}
		conn.Close()
		select {
		case got := <-updates:
			if got != value {
				t.Fatalf("got %q, want %q", got, value)
			}
		case <-time.After(timeout):
			t.Fatal("timed out waiting for pipe update")
		}
	}
}

func TestRejectNonLocalPipe(t *testing.T) {
	for _, path := range []string{"", "sock", `\\server\pipe\sock`, `\\.\pipe\`, `\\.\pipe\a\b`} {
		if listener, err := listenPipe(path); err == nil {
			listener.Close()
			t.Errorf("accepted %q", path)
		}
	}
}

func TestRenderEmojiIcon(t *testing.T) {
	for _, value := range []string{"🧦", "✅", "🧑‍💻", "A", "&"} {
		icon, err := renderIcon(value)
		if err != nil {
			t.Fatalf("render %q: %v", value, err)
		}
		if !win.DestroyIcon(icon) {
			t.Fatal("destroy icon failed")
		}
	}
}

func TestEmojiPixelsAreColoredAndTransparent(t *testing.T) {
	for _, value := range []string{"🧦", "✅", "😀", "🧑‍💻", "👋🏽"} {
		pixels, err := renderEmojiPixels(value)
		if err != nil {
			t.Fatalf("render %q: %v", value, err)
		}
		colored, transparent := 0, 0
		for i := 0; i < len(pixels); i += 4 {
			b, g, r, a := pixels[i], pixels[i+1], pixels[i+2], pixels[i+3]
			if a == 0 {
				transparent++
			}
			if a > 0 && (r != g || g != b) {
				colored++
			}
			if r > a || g > a || b > a {
				t.Fatalf("%q has non-premultiplied pixels", value)
			}
		}
		if colored < 10 {
			t.Errorf("%q has only %d colored pixels", value, colored)
		}
		if transparent < 10 {
			t.Errorf("%q lost its transparent background", value)
		}
	}
}

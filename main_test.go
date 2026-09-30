package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestRedirectedInputDetection(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()
	if !hasRedirectedInput(reader) {
		t.Fatal("stdin pipe was not detected")
	}
	file, err := os.Create(filepath.Join(t.TempDir(), "input"))
	if err != nil {
		t.Fatal(err)
	}
	if !hasRedirectedInput(file) {
		t.Error("stdin file was not detected")
	}
	file.Close()
	if hasRedirectedInput(file) {
		t.Error("closed handle treated as input")
	}
	null, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer null.Close()
	if hasRedirectedInput(null) {
		t.Error("null device should allow tray startup")
	}
}

func TestTickerFrames(t *testing.T) {
	clusters := splitClusters("hello")
	got := tickerFrames(clusters, 2)
	want := []string{"he", "el", "ll", "lo"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tickerFrames() = %#v, want %#v", got, want)
	}
}

func TestReadUpdates(t *testing.T) {
	updates := make(chan string, 1)
	if err := readUpdates(strings.NewReader("\ufeff🧦\n"), updates); err != nil {
		t.Fatal(err)
	}
	if got := <-updates; got != "🧦" {
		t.Fatalf("BOM update = %q", got)
	}
	if err := readUpdates(strings.NewReader("\xef\xbb\xbf🧦\r\n\n ✨ \r\nignored\x00value\n"), updates); err != nil {
		t.Fatal(err)
	}
	if got := <-updates; got != "✨" {
		t.Fatalf("got %q", got)
	}
	if err := readUpdates(strings.NewReader("🧑‍💻"), updates); err != nil {
		t.Fatal(err)
	}
	if got := <-updates; got != "🧑‍💻" {
		t.Fatalf("EOF update = %q", got)
	}
	if err := readUpdates(strings.NewReader(strings.Repeat("x", 1024*1024)), updates); err == nil {
		t.Fatal("expected oversized update to fail")
	}
}

func TestIconText(t *testing.T) {
	for _, value := range []string{"🧦", "👋🏽", "🧑‍💻", "🇺🇸", "1️⃣"} {
		if got := iconText(value + " done"); got != value {
			t.Errorf("iconText = %q, want %q", got, value)
		}
	}
}

func TestConcurrentUpdatesDoNotBlock(t *testing.T) {
	updates := make(chan string, 1)
	var writers sync.WaitGroup
	for i := 0; i < 10; i++ {
		writers.Add(1)
		go func() {
			defer writers.Done()
			for j := 0; j < 100; j++ {
				sendLatest(updates, "status")
			}
		}()
	}
	writers.Wait()
	if got := <-updates; got != "status" {
		t.Fatal(got)
	}
}

func TestSplitClustersKeepsCommonEmojiTogether(t *testing.T) {
	got := splitClusters("A🇺🇸👋🏽🧑‍💻B")
	want := []string{"A", "🇺🇸", "👋🏽", "🧑‍💻", "B"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("splitClusters() = %#v, want %#v", got, want)
	}
}

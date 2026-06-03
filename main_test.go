package main

import (
	"reflect"
	"testing"
)

func TestTickerFrames(t *testing.T) {
	clusters := splitClusters("hello")
	got := tickerFrames(clusters, 2)
	want := []string{"he", "el", "ll", "lo"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tickerFrames() = %#v, want %#v", got, want)
	}
}

func TestSplitClustersKeepsCommonEmojiTogether(t *testing.T) {
	got := splitClusters("A🇺🇸👋🏽🧑‍💻B")
	want := []string{"A", "🇺🇸", "👋🏽", "🧑‍💻", "B"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("splitClusters() = %#v, want %#v", got, want)
	}
}

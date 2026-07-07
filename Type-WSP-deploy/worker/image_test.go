package main

import (
	"bytes"
	"io"
	"testing"
)

func TestReadRawImageAllowsLimitedData(t *testing.T) {
	data := []byte("small image payload")
	got, err := readRawImage(io.NopCloser(bytes.NewReader(data)))
	if err != nil {
		t.Fatalf("readRawImage returned error: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("readRawImage = %q", got)
	}
}

func TestReadRawImageRejectsOversizedData(t *testing.T) {
	data := bytes.Repeat([]byte("a"), maxRawImageBytes+1)
	if _, err := readRawImage(io.NopCloser(bytes.NewReader(data))); err == nil {
		t.Fatal("expected oversized raw image to be rejected")
	}
}

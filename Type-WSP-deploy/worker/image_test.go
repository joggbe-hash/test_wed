package main

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"strings"
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

func TestResizeAndCompressAlwaysReencodesImage(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	src.Set(0, 0, color.RGBA{R: 255, A: 255})
	src.Set(1, 0, color.RGBA{G: 255, A: 255})
	src.Set(0, 1, color.RGBA{B: 255, A: 255})
	src.Set(1, 1, color.RGBA{R: 255, G: 255, A: 255})

	var original bytes.Buffer
	if err := jpeg.Encode(&original, src, &jpeg.Options{Quality: jpegQuality}); err != nil {
		t.Fatalf("encode test JPEG: %v", err)
	}

	metadataMarker := []byte("GPS-EXIF-SECRET")
	exifPayload := append([]byte("Exif\x00\x00"), metadataMarker...)
	exifLength := len(exifPayload) + 2
	input := []byte{0xff, 0xd8, 0xff, 0xe1, byte(exifLength >> 8), byte(exifLength)}
	input = append(input, exifPayload...)
	input = append(input, original.Bytes()[2:]...)
	processed, err := resizeAndCompress(input)
	if err != nil {
		t.Fatalf("resizeAndCompress returned error: %v", err)
	}
	if processed.contentType != "image/jpeg" {
		t.Fatalf("content type = %q, want image/jpeg", processed.contentType)
	}
	if bytes.Contains(processed.data, metadataMarker) {
		t.Fatal("processed image retained metadata appended to the source image")
	}
	if _, _, err := image.Decode(bytes.NewReader(processed.data)); err != nil {
		t.Fatalf("processed image is not decodable: %v", err)
	}
}

func TestResizeAndCompressRejectsImageAboveWorkingSetBudgetBeforeDecode(t *testing.T) {
	input := pngWithDimensions(5000, 4000)
	_, err := resizeAndCompress(input)
	if err == nil || !strings.Contains(err.Error(), "working set") {
		t.Fatalf("expected working-set rejection, got %v", err)
	}
}

func TestResizeAndCompressRejectsDimensionsThatOverflowPixelMultiplication(t *testing.T) {
	input := pngWithDimensions(^uint32(0), ^uint32(0))
	_, err := resizeAndCompress(input)
	if err == nil {
		t.Fatalf("expected oversized-dimension rejection, got %v", err)
	}
}

func TestLimitedImageBufferPreservesBoundedOutput(t *testing.T) {
	buffer := &limitedImageBuffer{limit: 4}
	if _, err := buffer.Write([]byte("safe")); err != nil {
		t.Fatalf("write within limit: %v", err)
	}
	if _, err := buffer.Write([]byte("!")); err == nil {
		t.Fatal("expected output beyond the limit to be rejected")
	}
	if got := buffer.String(); got != "safe" {
		t.Fatalf("buffer changed after rejected write: %q", got)
	}
}

func TestProcessedImageKeyIsUniquePerAttempt(t *testing.T) {
	first := processedImageKey("raw/example.jpg", "attempt-one", ".jpg")
	second := processedImageKey("raw/example.jpg", "attempt-two", ".jpg")

	if first == second {
		t.Fatalf("processed keys must differ across attempts: %q", first)
	}
	if first != "processed/example-attempt-one.jpg" {
		t.Fatalf("unexpected processed key: %q", first)
	}
}

func TestProcessedImageBytesSumsPersistentObjects(t *testing.T) {
	images := []*processedImage{{data: make([]byte, 11)}, {data: make([]byte, 17)}}
	if got := processedImageBytes(images); got != 28 {
		t.Fatalf("processedImageBytes = %d, want 28", got)
	}
}

func pngWithDimensions(width, height uint32) []byte {
	result := append([]byte(nil), []byte("\x89PNG\r\n\x1a\n")...)
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:4], width)
	binary.BigEndian.PutUint32(ihdr[4:8], height)
	ihdr[8] = 8
	ihdr[9] = 6
	result = appendPNGChunk(result, "IHDR", ihdr)
	return appendPNGChunk(result, "IEND", nil)
}

func appendPNGChunk(destination []byte, chunkType string, data []byte) []byte {
	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, uint32(len(data)))
	destination = append(destination, length...)
	chunkStart := len(destination)
	destination = append(destination, chunkType...)
	destination = append(destination, data...)
	checksum := make([]byte, 4)
	binary.BigEndian.PutUint32(checksum, crc32.ChecksumIEEE(destination[chunkStart:]))
	return append(destination, checksum...)
}

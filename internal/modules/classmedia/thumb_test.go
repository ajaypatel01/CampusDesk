package classmedia

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

func jpegOf(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w/2; x++ { // left half red, so rotation is visible
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}
	var b bytes.Buffer
	_ = jpeg.Encode(&b, img, nil)
	return b.Bytes()
}

// withOrientation inserts an Exif APP1 segment carrying orientation o.
func withOrientation(j []byte, o uint16) []byte {
	tiff := []byte{'M', 'M', 0, 42, 0, 0, 0, 8, 0, 1, 0x01, 0x12, 0, 3, 0, 0, 0, 1, byte(o >> 8), byte(o), 0, 0, 0, 0, 0, 0}
	payload := append([]byte("Exif\x00\x00"), tiff...)
	size := len(payload) + 2
	seg := append([]byte{0xFF, 0xE1, byte(size >> 8), byte(size)}, payload...)
	return append(append([]byte{0xFF, 0xD8}, seg...), j[2:]...)
}

func size(t *testing.T, b []byte) (int, int) {
	img, err := jpeg.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	return img.Bounds().Dx(), img.Bounds().Dy()
}

func TestThumbnail(t *testing.T) {
	th, err := makeThumbnail(jpegOf(1600, 800), 480)
	if err != nil {
		t.Fatal(err)
	}
	if w, h := size(t, th); w != 480 || h != 240 {
		t.Errorf("landscape thumb %dx%d, want 480x240", w, h)
	}
	rotated := withOrientation(jpegOf(1600, 800), 6)
	if o := exifOrientation(rotated); o != 6 {
		t.Fatalf("orientation %d, want 6", o)
	}
	th, err = makeThumbnail(rotated, 480)
	if err != nil {
		t.Fatal(err)
	}
	if w, h := size(t, th); w != 240 || h != 480 {
		t.Errorf("rotated thumb %dx%d, want 240x480", w, h)
	}
	if _, err := makeThumbnail([]byte("not an image"), 480); err == nil {
		t.Error("garbage should fail")
	}
}

func TestCheckFile(t *testing.T) {
	j := jpegOf(10, 10)
	if ct, ext, err := checkFile(KindPhoto, "x.png", j); err != nil || ct != "image/jpeg" || ext != ".jpg" {
		t.Errorf("jpeg named .png: %v %v %v (should be judged by content)", ct, ext, err)
	}
	if _, _, err := checkFile(KindPhoto, "evil.jpg", []byte("<html><script>")); err == nil {
		t.Error("html posing as a photo accepted")
	}
	heic := append([]byte{0, 0, 0, 24}, []byte("ftypheic0000")...)
	if ct, _, err := checkFile(KindPhoto, "IMG_1.HEIC", heic); err != nil || ct != "image/heic" {
		t.Errorf("heic: %v %v", ct, err)
	}
	if ct, _, err := checkFile(KindDocument, "Timetable.PDF", []byte("%PDF-1.4")); err != nil || ct != "application/pdf" {
		t.Errorf("pdf: %v %v", ct, err)
	}
	if _, _, err := checkFile(KindDocument, "run.exe", []byte("MZ")); err == nil {
		t.Error("exe accepted as a document")
	}
	if n := cleanName(`..\\..\\a"b.pdf`); n != "ab.pdf" {
		t.Errorf("cleanName = %q", n)
	}
}

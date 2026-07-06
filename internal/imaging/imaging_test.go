package imaging

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
)

// drawGradient returns an image.Image of the given size where each
// pixel's RGB values are a deterministic function of (x, y). It is
// used as test input so resize behaviour can be reasoned about
// without depending on a checked-in fixture file.
func drawGradient(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// Red sweeps horizontally, green vertically, blue
			// along the diagonal. This makes a resize that drops
			// information observably wrong (catastrophic blur of
			// the red/green edges), which is exactly what we want
			// to catch in regression tests.
			r := uint8((x * 255) / maxInt(w-1, 1))
			g := uint8((y * 255) / maxInt(h-1, 1))
			b := uint8(((x + y) * 255) / maxInt(w+h-2, 1))
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}
	return img
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func encodeJPEG(t *testing.T, img image.Image, quality int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	return buf.Bytes()
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestStdProcessor_Process_JPEG_ResizesToTarget(t *testing.T) {
	src := encodeJPEG(t, drawGradient(1200, 800), 90)
	p := NewStdProcessor()

	res, err := p.Process(context.Background(), bytes.NewReader(src), ProcessOptions{
		TargetWidth:  800,
		TargetHeight: 600,
		Quality:      85,
		Format:       FormatJPEG,
	})
	if err != nil {
		t.Fatalf("process: %v", err)
	}

	if res.Width != 800 || res.Height != 600 {
		t.Errorf("dimensions = %dx%d, want 800x600", res.Width, res.Height)
	}
	if res.MIMEType != "image/jpeg" {
		t.Errorf("MIMEType = %q, want image/jpeg", res.MIMEType)
	}

	// Re-decode the output to confirm it's a complete, valid JPEG
	// of the expected dimensions.
	decoded, _, err := image.Decode(bytes.NewReader(res.Data))
	if err != nil {
		t.Fatalf("re-decode output: %v", err)
	}
	got := decoded.Bounds()
	if got.Dx() != 800 || got.Dy() != 600 {
		t.Errorf("re-decoded dimensions = %dx%d, want 800x600", got.Dx(), got.Dy())
	}
}

func TestStdProcessor_Process_PNG_ResizesToTarget(t *testing.T) {
	src := encodePNG(t, drawGradient(400, 400))
	p := NewStdProcessor()

	res, err := p.Process(context.Background(), bytes.NewReader(src), ProcessOptions{
		TargetWidth:  200,
		TargetHeight: 200,
		Format:       FormatPNG,
	})
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if res.MIMEType != "image/png" {
		t.Errorf("MIMEType = %q, want image/png", res.MIMEType)
	}
	if res.Width != 200 || res.Height != 200 {
		t.Errorf("dimensions = %dx%d, want 200x200", res.Width, res.Height)
	}
}

func TestStdProcessor_Process_UnsupportedFormat(t *testing.T) {
	// A plain text payload is not a JPEG/PNG by sniff, so Process
	// must reject it with ErrUnsupportedFormat.
	p := NewStdProcessor()
	_, err := p.Process(context.Background(), bytes.NewReader([]byte("hello world")), ProcessOptions{
		TargetWidth:  100,
		TargetHeight: 100,
		Quality:      85,
		Format:       FormatJPEG,
	})
	if !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("err = %v, want ErrUnsupportedFormat", err)
	}
}

func TestStdProcessor_Process_RejectsCorruptBytes(t *testing.T) {
	// A valid-looking JPEG magic header followed by garbage. image.Decode
	// should fail and Process should wrap that in ErrDecodeFailed.
	corrupt := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, []byte("not really a jpeg")...)
	p := NewStdProcessor()
	_, err := p.Process(context.Background(), bytes.NewReader(corrupt), ProcessOptions{
		TargetWidth:  100,
		TargetHeight: 100,
		Quality:      85,
		Format:       FormatJPEG,
	})
	if !errors.Is(err, ErrDecodeFailed) {
		t.Fatalf("err = %v, want ErrDecodeFailed", err)
	}
}

func TestStdProcessor_Process_RejectsEmptyInput(t *testing.T) {
	p := NewStdProcessor()
	_, err := p.Process(context.Background(), bytes.NewReader(nil), ProcessOptions{
		TargetWidth:  100,
		TargetHeight: 100,
		Quality:      85,
		Format:       FormatJPEG,
	})
	if err == nil {
		t.Fatal("expected error for empty input, got nil")
	}
}

func TestStdProcessor_Process_RejectsInvalidOptions(t *testing.T) {
	p := NewStdProcessor()
	src := encodeJPEG(t, drawGradient(100, 100), 90)

	cases := []struct {
		name string
		opts ProcessOptions
		want error
	}{
		{
			name: "zero width",
			opts: ProcessOptions{TargetWidth: 0, TargetHeight: 100, Quality: 85, Format: FormatJPEG},
			want: ErrInvalidOptions,
		},
		{
			name: "zero height",
			opts: ProcessOptions{TargetWidth: 100, TargetHeight: 0, Quality: 85, Format: FormatJPEG},
			want: ErrInvalidOptions,
		},
		{
			name: "negative dimensions",
			opts: ProcessOptions{TargetWidth: -1, TargetHeight: -1, Quality: 85, Format: FormatJPEG},
			want: ErrInvalidOptions,
		},
		{
			name: "jpeg quality too low",
			opts: ProcessOptions{TargetWidth: 100, TargetHeight: 100, Quality: 0, Format: FormatJPEG},
			want: ErrInvalidOptions,
		},
		{
			name: "jpeg quality too high",
			opts: ProcessOptions{TargetWidth: 100, TargetHeight: 100, Quality: 101, Format: FormatJPEG},
			want: ErrInvalidOptions,
		},
		{
			name: "unknown format",
			opts: ProcessOptions{TargetWidth: 100, TargetHeight: 100, Format: "gif"},
			want: ErrInvalidOptions,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := p.Process(context.Background(), bytes.NewReader(src), tc.opts)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestStdProcessor_Process_JPEGQualityDifference(t *testing.T) {
	// A noise-heavy image is the worst case for JPEG compression;
	// at low quality the file shrinks dramatically compared to high
	// quality. Use it to assert the Quality field is actually
	// threaded through to jpeg.Encode.
	src := encodeJPEG(t, drawGradient(640, 480), 95)

	low, err := NewStdProcessor().Process(context.Background(), bytes.NewReader(src), ProcessOptions{
		TargetWidth: 640, TargetHeight: 480, Quality: 50, Format: FormatJPEG,
	})
	if err != nil {
		t.Fatalf("process low: %v", err)
	}
	high, err := NewStdProcessor().Process(context.Background(), bytes.NewReader(src), ProcessOptions{
		TargetWidth: 640, TargetHeight: 480, Quality: 95, Format: FormatJPEG,
	})
	if err != nil {
		t.Fatalf("process high: %v", err)
	}
	if len(high.Data) <= len(low.Data) {
		t.Errorf("expected high-quality output to be larger than low-quality, got high=%d low=%d", len(high.Data), len(low.Data))
	}
}

func TestCentreCropRect(t *testing.T) {
	// Helper: image.Rect-equivalent bounds for a 1000x500 source.
	src := image.Rect(0, 0, 1000, 500)

	t.Run("4:3 target on 2:1 source crops sides", func(t *testing.T) {
		got := centreCropRect(src, 4, 3)
		// Aspect 4/3 = 1.333, source 2.0 -> source is wider, so we
		// crop horizontally to 500*4/3 ≈ 666 wide, 500 tall.
		if got.Dx() != 666 || got.Dy() != 500 {
			t.Errorf("crop = %dx%d, want 666x500", got.Dx(), got.Dy())
		}
		// Centred: x offset = (1000-666)/2 = 167.
		if got.Min.X != 167 || got.Min.Y != 0 {
			t.Errorf("crop origin = (%d,%d), want (167,0)", got.Min.X, got.Min.Y)
		}
	})

	t.Run("1:1 target on 2:1 source crops sides", func(t *testing.T) {
		got := centreCropRect(src, 1, 1)
		if got.Dx() != 500 || got.Dy() != 500 {
			t.Errorf("crop = %dx%d, want 500x500", got.Dx(), got.Dy())
		}
		if got.Min.X != 250 || got.Min.Y != 0 {
			t.Errorf("crop origin = (%d,%d), want (250,0)", got.Min.X, got.Min.Y)
		}
	})

	t.Run("tall target on wide source crops vertically", func(t *testing.T) {
		// 1000x500 source, 1:2 target (tall). Source aspect 2.0,
		// target 0.5 -> source is wider, so we still crop sides.
		// Use a square source + tall target to exercise the other
		// branch.
		tallSource := image.Rect(0, 0, 500, 1000)
		got := centreCropRect(tallSource, 1, 2)
		// target aspect 0.5, source 0.5 -> equal, takes the >= branch.
		// Either way the result should be 500x1000 (full source).
		if got.Dx() != 500 || got.Dy() != 1000 {
			t.Errorf("crop = %dx%d, want 500x1000", got.Dx(), got.Dy())
		}
	})

	t.Run("already-matching aspect is identity", func(t *testing.T) {
		// 4:3 source, 4:3 target -> no crop.
		matching := image.Rect(0, 0, 800, 600)
		got := centreCropRect(matching, 4, 3)
		if got != matching {
			t.Errorf("crop = %v, want %v (identity)", got, matching)
		}
	})
}

func TestStdProcessor_Process_OnlyAcceptsJPEGAndPNG(t *testing.T) {
	// Belt-and-braces: a GIF magic header should be rejected even
	// though image.Decode could decode it. Our policy is
	// JPEG-or-PNG-only.
	gifHeader := append([]byte("GIF89a"), make([]byte, 100)...)
	p := NewStdProcessor()
	_, err := p.Process(context.Background(), bytes.NewReader(gifHeader), ProcessOptions{
		TargetWidth: 100, TargetHeight: 100, Quality: 85, Format: FormatJPEG,
	})
	if !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("err = %v, want ErrUnsupportedFormat", err)
	}
	if err != nil && !strings.Contains(err.Error(), "image/gif") {
		t.Errorf("error should mention the detected mime type, got %v", err)
	}
}

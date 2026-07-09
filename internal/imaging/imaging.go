package imaging

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"

	"golang.org/x/image/draw"
)

// Format is the output encoding the Processor produces.
type Format string

const (
	// FormatJPEG re-encodes the image as JPEG. Quality on
	// ProcessOptions is honoured.
	FormatJPEG Format = "jpeg"
	// FormatPNG re-encodes the image as PNG. Quality on
	// ProcessOptions is ignored.
	FormatPNG Format = "png"
)

// Sentinel errors returned by Processor.Process. Callers should
// errors.Is against these to distinguish expected failures (bad
// input format, corrupt bytes) from unexpected ones (encoding bug).
var (
	// ErrUnsupportedFormat is returned when the source bytes
	// declare (or smell like) a MIME type other than image/jpeg or
	// image/png.
	ErrUnsupportedFormat = errors.New("imaging: unsupported image format")
	// ErrDecodeFailed is returned when the source declares a
	// supported MIME type but the bytes cannot be decoded (truncated,
	// corrupt, or encrypted payload).
	ErrDecodeFailed = errors.New("imaging: failed to decode image")
	// ErrEncodeFailed is returned when the processed image cannot
	// be re-encoded (typically an internal bug — the only public
	// failure mode is an io.Writer error, which we wrap).
	ErrEncodeFailed = errors.New("imaging: failed to encode image")
	// ErrInvalidOptions is returned when ProcessOptions is
	// malformed (zero/negative dimensions, unsupported Format,
	// out-of-range Quality).
	ErrInvalidOptions = errors.New("imaging: invalid process options")
)

// ProcessOptions configures a single Process call. TargetWidth and
// TargetHeight are mandatory and must be positive. Quality is only
// honoured for FormatJPEG and must be in 1..100; ignored for FormatPNG.
type ProcessOptions struct {
	TargetWidth  int
	TargetHeight int
	Quality      int
	Format       Format
}

// Result is what Process returns. Data is the encoded image bytes
// (always a complete, self-contained image in the requested Format).
// MIMEType is the HTTP Content-Type to use when serving or uploading
// the bytes (e.g. "image/jpeg"). Width and Height are the pixel
// dimensions of the encoded result, guaranteed to match the target
// dimensions.
type Result struct {
	Data     []byte
	MIMEType string
	Width    int
	Height   int
}

// Processor decodes, resizes, and re-encodes a single image. The
// zero value is not usable; obtain an instance via NewStdProcessor
// (or a test fake). Implementations must be safe for concurrent use.
type Processor interface {
	Process(ctx context.Context, src io.Reader, opts ProcessOptions) (*Result, error)
}

// StdProcessor is the default Processor. It uses image.Decode (which
// sniffs the first few bytes) for input, draw.CatmullRom for
// rescaling, and image/jpeg or image/png for re-encoding.
type StdProcessor struct{}

// NewStdProcessor returns a usable StdProcessor. The returned value
// is stateless and safe for concurrent use.
func NewStdProcessor() *StdProcessor { return &StdProcessor{} }

// Process reads src as a JPEG or PNG, centre-crops the image to the
// target aspect ratio, scales it to the target dimensions, and
// re-encodes it in the requested format. ctx is currently unused
// (processing is in-memory and synchronous) but is part of the
// signature so future implementations can do streaming work or honour
// cancellation without changing callers.
func (p *StdProcessor) Process(ctx context.Context, src io.Reader, opts ProcessOptions) (*Result, error) {
	if err := validateOptions(opts); err != nil {
		return nil, err
	}

	// Sniff the first 512 bytes for a cheap MIME-type gate before
	// we hand the rest to image.Decode. http.DetectContentType is
	// the stdlib's recommended approach and matches what net/http
	// does on the server side for multipart uploads.
	//
	// We can't Seek on the caller's reader, so we buffer the sniff
	// bytes and prepend them to the bytes image.Decode sees.
	var sniff [512]byte
	n, _ := io.ReadFull(src, sniff[:])
	if n == 0 {
		return nil, fmt.Errorf("%w: empty input", ErrDecodeFailed)
	}
	mime := http.DetectContentType(sniff[:n])
	if mime != "image/jpeg" && mime != "image/png" {
		return nil, fmt.Errorf("%w: %q is not a supported image type", ErrUnsupportedFormat, mime)
	}

	img, _, err := image.Decode(io.MultiReader(bytes.NewReader(sniff[:n]), src))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecodeFailed, err)
	}

	// Centre-crop to the target aspect ratio, then scale to the
	// target dimensions in one draw.CatmullRom pass.
	srcBounds := img.Bounds()
	crop := centreCropRect(srcBounds, opts.TargetWidth, opts.TargetHeight)
	cropped := image.NewRGBA(image.Rect(0, 0, crop.Dx(), crop.Dy()))
	draw.Draw(cropped, cropped.Bounds(), img, crop.Min, draw.Src)

	scaled := image.NewRGBA(image.Rect(0, 0, opts.TargetWidth, opts.TargetHeight))
	draw.CatmullRom.Scale(scaled, scaled.Bounds(), cropped, cropped.Bounds(), draw.Over, nil)

	var buf bytes.Buffer
	var contentType string
	switch opts.Format {
	case FormatJPEG:
		if err := jpeg.Encode(&buf, scaled, &jpeg.Options{Quality: opts.Quality}); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrEncodeFailed, err)
		}
		contentType = "image/jpeg"
	case FormatPNG:
		if err := png.Encode(&buf, scaled); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrEncodeFailed, err)
		}
		contentType = "image/png"
	default:
		// validateOptions should have caught this; defensive guard.
		return nil, fmt.Errorf("%w: format %q", ErrInvalidOptions, opts.Format)
	}

	return &Result{
		Data:     buf.Bytes(),
		MIMEType: contentType,
		Width:    opts.TargetWidth,
		Height:   opts.TargetHeight,
	}, nil
}

// validateOptions returns ErrInvalidOptions (wrapped with detail) if
// any field is out of range.
func validateOptions(opts ProcessOptions) error {
	if opts.TargetWidth <= 0 || opts.TargetHeight <= 0 {
		return fmt.Errorf("%w: TargetWidth and TargetHeight must be positive (got %dx%d)", ErrInvalidOptions, opts.TargetWidth, opts.TargetHeight)
	}
	switch opts.Format {
	case FormatJPEG:
		if opts.Quality < 1 || opts.Quality > 100 {
			return fmt.Errorf("%w: Quality must be 1..100 for JPEG (got %d)", ErrInvalidOptions, opts.Quality)
		}
	case FormatPNG:
		// Quality is ignored; no constraint.
	default:
		return fmt.Errorf("%w: Format must be %q or %q (got %q)", ErrInvalidOptions, FormatJPEG, FormatPNG, opts.Format)
	}
	return nil
}

// centreCropRect returns the largest rectangle inside bounds that has
// the same aspect ratio as targetW:targetH, centred on the source.
// The output rectangle is in the source's coordinate space.
func centreCropRect(bounds image.Rectangle, targetW, targetH int) image.Rectangle {
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	targetAspect := float64(targetW) / float64(targetH)
	srcAspect := float64(srcW) / float64(srcH)

	var cropW, cropH int
	if srcAspect > targetAspect {
		// Source is wider than target — crop horizontally.
		cropH = srcH
		cropW = int(float64(srcH) * targetAspect)
	} else {
		// Source is taller (or equal) — crop vertically.
		cropW = srcW
		cropH = int(float64(srcW) / targetAspect)
	}

	offX := (srcW - cropW) / 2
	offY := (srcH - cropH) / 2

	return image.Rect(bounds.Min.X+offX, bounds.Min.Y+offY, bounds.Min.X+offX+cropW, bounds.Min.Y+offY+cropH)
}

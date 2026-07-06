package routes

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"stren/internal/imaging"
	"stren/internal/utils"
)

// ExerciseImageConfig controls the two image variants the admin
// upload route produces. Both variants are derived from the same
// upload (a single source image goes in) and share the same 4:3
// aspect ratio so the display card and any future "view original"
// surface stay visually consistent.
type ExerciseImageConfig struct {
	// DisplayWidth / DisplayHeight is the size written to the
	// "img_url" storage key. Used in cards, dialogs, and any other
	// UI surface that needs the exercise image. Quality 85 keeps
	// file size modest for repeated network fetches.
	DisplayWidth  int
	DisplayHeight int
	// OriginalWidth / OriginalHeight is the size of the
	// "img_url_original" storage key. A 1920x1440 source surrogate
	// is large enough to look good in a future full-screen viewer
	// without bloating R2 storage. Quality 95 preserves detail.
	OriginalWidth  int
	OriginalHeight int
}

// DefaultExerciseImageConfig is the production configuration for the
// admin exercise image upload. Centralised so tests can refer to the
// same numbers.
var DefaultExerciseImageConfig = ExerciseImageConfig{
	DisplayWidth:   800,
	DisplayHeight:  600,
	OriginalWidth:  1920,
	OriginalHeight: 1440,
}

// imageUploadRequest is the JSON body the JS upload widget POSTs.
// Kept as a separate type so the response type can evolve without
// rippling through callers.
type imageUploadResponse struct {
	DisplayKey  string `json:"display_key"`
	OriginalKey string `json:"original_key"`
}

// exerciseImageProcessor is the subset of imaging.Processor the
// route uses. Declared as a local interface so the route is unit
// testable with a fake without depending on the full package.
type exerciseImageProcessor interface {
	Process(ctx context.Context, src io.Reader, opts imaging.ProcessOptions) (*imaging.Result, error)
}

// exerciseImageUploader is the subset of utils.PutObject the route
// uses. Declared as a local interface so tests can substitute a
// fake without touching the S3 client.
type exerciseImageUploader interface {
	PutObject(ctx context.Context, key, contentType string, body io.Reader) error
}

// AdminExerciseImageUpload handles the server-side upload flow for
// exercise images. The browser POSTs a multipart form with a single
// "file" field; this route validates, decodes, resizes via the
// imaging package, and writes two normalised variants to R2. The
// returned keys are set into the form's hidden inputs by the JS
// widget and persisted when the admin submits the exercise form.
//
// The route enforces a 10 MB body cap (matching the widget's
// client-side pre-check) via http.MaxBytesReader so a hostile client
// can't exhaust memory before the multipart parser runs. The MIME
// gate (image/jpeg or image/png) and a non-empty file are also
// enforced up front; the imaging processor's own format check is the
// second line of defence.
//
// On any failure mid-flight (e.g. the second upload to R2 errors
// after the first succeeded) the route does best-effort cleanup of
// the partially-written objects so R2 doesn't accumulate orphan
// files.
func (h *Handler) AdminExerciseImageUpload(c echo.Context) error {
	ctx := c.Request().Context()

	// 10 MB cap, matching the client-side pre-check. Echo's
	// default BodyLimit is unset for this route so the cap is
	// applied here. Using MaxBytesReader (rather than a global
	// middleware) keeps the limit scoped to the upload route only.
	const maxBody = 10 << 20 // 10 MiB
	c.Request().Body = http.MaxBytesReader(c.Response().Writer, c.Request().Body, maxBody)

	// Max multipart memory before spooling to disk. Echo's default
	// is 32 MB which is well above our body cap, so 8 MB is
	// comfortable and keeps the in-memory footprint small.
	if err := c.Request().ParseMultipartForm(8 << 20); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Could not parse upload (file may exceed 10 MB)")
	}

	file, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing 'file' field")
	}
	if file.Size <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Uploaded file is empty")
	}
	if file.Size > maxBody {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "File exceeds 10 MB limit")
	}

	src, err := file.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Could not read uploaded file")
	}
	defer src.Close()

	// The display variant is the small, frequently-loaded one, so
	// it gets processed first. If the original variant later fails
	// to upload we still have something useful to return, and we
	// can clean up the orphan display object.
	displayRes, err := h.imageProcessor.Process(ctx, src, imaging.ProcessOptions{
		TargetWidth:  h.imageConfig.DisplayWidth,
		TargetHeight: h.imageConfig.DisplayHeight,
		Quality:      85,
		Format:       imaging.FormatJPEG,
	})
	if err != nil {
		return mapImagingError(err)
	}

	// Re-open the file for the second Process call — the first
	// Process read src to EOF. Multipart File implements io.Seeker
	// (it's backed by a temp file in Echo's default config), so
	// Seek(0) is reliable.
	if _, seekErr := src.Seek(0, io.SeekStart); seekErr != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Could not re-read uploaded file")
	}
	originalRes, err := h.imageProcessor.Process(ctx, src, imaging.ProcessOptions{
		TargetWidth:  h.imageConfig.OriginalWidth,
		TargetHeight: h.imageConfig.OriginalHeight,
		Quality:      95,
		Format:       imaging.FormatJPEG,
	})
	if err != nil {
		return mapImagingError(err)
	}

	// Both variants are JPEG (forced by the imaging package's
	// Format constant in this route), so the file extensions and
	// content types are fixed.
	id := uuid.New().String()
	displayKey := "exercises/" + id + ".jpg"
	originalKey := "exercises/" + id + "_original.jpg"

	displayBody := bytes.NewReader(displayRes.Data)
	if err := h.imageUploader.PutObject(ctx, displayKey, displayRes.MIMEType, displayBody); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Could not upload display image")
	}

	if err := h.imageUploader.PutObject(ctx, originalKey, originalRes.MIMEType, bytes.NewReader(originalRes.Data)); err != nil {
		// Best-effort cleanup: drop the display object we just
		// uploaded so we don't leave an orphan in R2.
		if delErr := utils.DeleteObject(displayKey); delErr != nil {
			c.Logger().Warnf("failed to clean up display image %q after original upload failed: %v", displayKey, delErr)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Could not upload original image")
	}

	return c.JSON(http.StatusOK, imageUploadResponse{
		DisplayKey:  displayKey,
		OriginalKey: originalKey,
	})
}

// mapImagingError translates imaging sentinel errors into the
// HTTP-equivalent that callers should see. Keeps the route body
// focused on flow rather than error-mapping noise.
func mapImagingError(err error) error {
	switch {
	case err == imaging.ErrUnsupportedFormat, strings.Contains(err.Error(), imaging.ErrUnsupportedFormat.Error()):
		return echo.NewHTTPError(http.StatusBadRequest, "File must be a JPEG or PNG image")
	case err == imaging.ErrDecodeFailed, strings.Contains(err.Error(), imaging.ErrDecodeFailed.Error()):
		return echo.NewHTTPError(http.StatusBadRequest, "Image could not be decoded (file may be corrupt)")
	case err == imaging.ErrInvalidOptions, strings.Contains(err.Error(), imaging.ErrInvalidOptions.Error()):
		return echo.NewHTTPError(http.StatusInternalServerError, "Image processor misconfigured")
	default:
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Image processing failed: %v", err))
	}
}

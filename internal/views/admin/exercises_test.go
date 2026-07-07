package admin

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"stren/internal/models"
)

// renderToString renders a templ component to a string for assertions.
func renderToString(t *testing.T, component templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	if err := component.Render(context.Background(), &buf); err != nil {
		t.Fatalf("failed to render component: %v", err)
	}
	return buf.String()
}

func TestAdminExerciseForm_New_ContainsUploadWidget(t *testing.T) {
	html := renderToString(t, AdminExerciseForm(AdminExerciseFormData{IsEdit: false}, "Admin", true, true))

	// The ImageUpload widget must render with the expected data
	// attributes so the JS can find the upload endpoint.
	if !strings.Contains(html, `data-image-upload="exercise"`) {
		t.Error("expected data-image-upload attribute on the upload widget root")
	}
	if !strings.Contains(html, `data-upload-endpoint="/admin/exercises/image-upload"`) {
		t.Error("expected data-upload-endpoint pointing at the admin image upload route")
	}
	if !strings.Contains(html, `data-max-size-mb="10"`) {
		t.Error("expected 10 MB max-size attribute on the upload widget")
	}

	// The hidden inputs the form submits back to the server must
	// be present with the right names so the route can pick them
	// up via c.FormValue.
	if !strings.Contains(html, `name="img_key"`) {
		t.Error("expected hidden img_key input on the new form")
	}
	if !strings.Contains(html, `name="img_key_original"`) {
		t.Error("expected hidden img_key_original input on the new form")
	}

	// The legacy free-text "img_url" input must be gone — it was
	// the manual key-paste field that the new widget replaces.
	if strings.Contains(html, `id="img_url"`) {
		t.Error("legacy img_url text input must not appear on the new form")
	}

	// The "Remove current image" checkbox should only appear on
	// the edit form when an image exists, never on the new form.
	if strings.Contains(html, `id="clear_image"`) {
		t.Error("clear_image checkbox must not appear on the new form")
	}
}

func TestAdminExerciseForm_Edit_NoImage_ContainsUploadWidget(t *testing.T) {
	ex := &models.Exercise{ID: "ex-1", Name: "Squat", Type: models.ExerciseTypeStrength}
	html := renderToString(t, AdminExerciseForm(AdminExerciseFormData{Exercise: ex, IsEdit: true}, "Admin", true, true))

	// Upload widget must still be present on the edit form so
	// the admin can add an image to an exercise that has none.
	if !strings.Contains(html, `data-image-upload="exercise"`) {
		t.Error("expected ImageUpload widget on the edit form (no image)")
	}
	// The hidden inputs should render with empty values.
	if !strings.Contains(html, `name="img_key" value=""`) {
		t.Error("expected empty img_key on edit form with no current image")
	}
	// No "Remove current image" control when there's no image.
	if strings.Contains(html, `id="clear_image"`) {
		t.Error("clear_image checkbox should not appear when there is no current image")
	}
}

func TestAdminExerciseForm_Edit_WithImage_ContainsClearCheckbox(t *testing.T) {
	ex := &models.Exercise{
		ID:             "ex-1",
		Name:           "Squat",
		Type:           models.ExerciseTypeStrength,
		ImgURL:         "exercises/abc.jpg",
		ImgURLOriginal: "exercises/abc_original.jpg",
	}
	html := renderToString(t, AdminExerciseForm(AdminExerciseFormData{Exercise: ex, IsEdit: true}, "Admin", true, true))

	// Both keys should be pre-populated so the form submits them
	// back unchanged if the user doesn't touch the upload widget.
	if !strings.Contains(html, `name="img_key" value="exercises/abc.jpg"`) {
		t.Error("expected img_key pre-populated with the current key")
	}
	if !strings.Contains(html, `name="img_key_original" value="exercises/abc_original.jpg"`) {
		t.Error("expected img_key_original pre-populated with the current key")
	}

	// The "Remove current image" checkbox must appear on the edit
	// form when the exercise already has an image.
	if !strings.Contains(html, `id="clear_image"`) {
		t.Error("expected clear_image checkbox on the edit form when an image exists")
	}
	if !strings.Contains(html, `name="clear_image"`) {
		t.Error("expected clear_image name attribute")
	}
	if !strings.Contains(html, `value="true"`) {
		t.Error("expected clear_image checkbox to submit value=true")
	}

	// The current image preview should be wired up with the
	// public URL (the widget receives the resolved URL via
	// CurrentURL, so we just check the key ends up in the hidden
	// input — the actual public URL is computed by utils in the
	// view, and a unit test against the rendered HTML can only
	// see the absolute URL the template wrote).
	if !strings.Contains(html, `name="img_key"`) {
		t.Error("expected img_key input on the edit form")
	}
}

func TestAdminExerciseForm_Edit_UploadEndpointHardCoded(t *testing.T) {
	// Regression guard: the ImageUpload widget's upload endpoint
	// is a constant in the templ file. If a future refactor
	// changes it to derive from a prop, this test ensures the
	// admin form still POSTs to the admin route (not the generic
	// /api path that the default UploadEndpoint value would
	// produce).
	ex := &models.Exercise{ID: "ex-1", Name: "Squat", Type: models.ExerciseTypeStrength}
	html := renderToString(t, AdminExerciseForm(AdminExerciseFormData{Exercise: ex, IsEdit: true}, "Admin", true, true))

	if !strings.Contains(html, `data-upload-endpoint="/admin/exercises/image-upload"`) {
		t.Errorf("expected /admin/exercises/image-upload endpoint, got html: %s", html)
	}
}

func TestAdminExerciseForm_PreviewAspectMatchesCard(t *testing.T) {
	// The exercise image is used on the history card with
	// `w-full h-48 object-cover` inside a max-w-3xl container
	// (768px), giving a visible aspect of 768:192 = 4:1. The
	// admin form's preview must match that aspect so the admin
	// sees roughly what users will see — not the raw 4:3 file
	// aspect, which renders as a far taller rectangle.
	for _, tc := range []struct {
		name   string
		form   AdminExerciseFormData
	}{
		{
			name: "new form",
			form: AdminExerciseFormData{IsEdit: false},
		},
		{
			name: "edit form with image",
			form: AdminExerciseFormData{
				IsEdit:           true,
				Exercise:         &models.Exercise{ID: "ex-1", Name: "Squat", ImgURL: "exercises/abc.jpg"},
			},
		},
		{
			name: "edit form without image",
			form: AdminExerciseFormData{
				IsEdit:   true,
				Exercise: &models.Exercise{ID: "ex-1", Name: "Squat"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			html := renderToString(t, AdminExerciseForm(tc.form, "Admin", true, true))
			if !strings.Contains(html, "padding-bottom: 25%") {
				t.Errorf("expected preview to use 4:1 aspect (padding-bottom: 25%%), got html:\n%s", html)
			}
			if strings.Contains(html, "padding-bottom: 75%") {
				t.Errorf("preview still using 4:3 aspect (padding-bottom: 75%%), got html:\n%s", html)
			}
		})
	}
}

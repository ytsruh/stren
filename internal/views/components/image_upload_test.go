package components

import (
	"strings"
	"testing"
)

func TestImageUpload_RendersValidStyleAttribute(t *testing.T) {
	props := ImageUploadProps{
		Name:               "exercise",
		Label:              "Image",
		CurrentURL:         "https://pub-test.r2.dev/exercises/abc.jpg",
		CurrentKey:         "exercises/abc.jpg",
		CurrentOriginalKey: "exercises/abc_original.jpg",
		MaxSizeMB:          10,
		UploadEndpoint:     "/admin/exercises/image-upload",
		AspectWidth:        16,
		AspectHeight:       9,
	}
	html := renderToString(t, ImageUpload(props))
	if !strings.Contains(html, `padding-bottom: 56.25%`) {
		t.Errorf("expected 'padding-bottom: 56.25%%' in rendered HTML, got:\n%s", html)
	}
	if strings.Contains(html, "TemplUnsupportedStyleAttributeValue") {
		t.Error("rendered HTML still contains the unsupported-style-attribute sentinel — SafeURL leaked through")
	}
}

func TestImageUpload_RemoveButton_VisibleWhenCurrentURLSet(t *testing.T) {
	html := renderToString(t, ImageUpload(ImageUploadProps{
		Name:       "exercise",
		Label:      "Image",
		CurrentURL: "https://pub-test.r2.dev/exercises/abc.jpg",
	}))

	// The image and the remove button must share a single
	// flex row using justify-between — the button sits to the
	// right of the image rather than below it.
	if !strings.Contains(html, `class="flex items-start justify-between gap-3"`) {
		t.Error("expected image and remove button in a flex justify-between row")
	}

	// The small-text Remove button must render with the right id
	// so the JS can find it.
	if !strings.Contains(html, `id="image-remove-exercise"`) {
		t.Error("expected image-remove button id when CurrentURL is set")
	}
	// The visible text should be 'Remove img' (capitalised, as
	// the user requested).
	if !strings.Contains(html, `>Remove img<`) {
		t.Error("expected the remove button to render the text 'Remove img'")
	}
	// The hidden field carrying the form value must be present
	// with the default name "clear_image".
	if !strings.Contains(html, `name="clear_image"`) {
		t.Error("expected hidden clear_image field when CurrentURL is set")
	}
	if !strings.Contains(html, `id="image-remove-field-exercise"`) {
		t.Error("expected image-remove-field id on the hidden field")
	}
	// The data attributes that drive the JS confirm dialog must
	// be on the widget root.
	if !strings.Contains(html, `data-remove-field="clear_image"`) {
		t.Error("expected data-remove-field attribute on widget root")
	}
	if !strings.Contains(html, `data-confirm-message="Remove this image? It will be deleted when you save."`) {
		t.Error("expected data-confirm-message attribute on widget root")
	}
	if !strings.Contains(html, `data-state="idle"`) {
		t.Error("expected the remove button to start in the 'idle' state")
	}
	// The idle-state button must use a Basecoat button class
	// (btn btn-sm-outline) so it renders as a properly styled
	// small outline button rather than a plain text link.
	if !strings.Contains(html, `class="btn btn-sm-outline"`) {
		t.Error("expected the idle-state button to use the Basecoat 'btn btn-sm-outline' classes")
	}
	// The previous text-link styling must be gone from the
	// button itself. We check the button opening tag in
	// particular (not the entire HTML) because the CSS file
	// served alongside the page still defines the
	// text-muted-foreground class — that's fine, it just
	// shouldn't be on the button.
	btnStart := strings.Index(html, `id="image-remove-exercise"`)
	if btnStart < 0 {
		t.Fatal("could not locate the remove button in the rendered HTML")
	}
	btnEnd := strings.Index(html[btnStart:], ">")
	if btnEnd < 0 {
		t.Fatal("could not locate the end of the remove button opening tag")
	}
	btnOpen := html[btnStart : btnStart+btnEnd]
	if strings.Contains(btnOpen, "text-muted-foreground") {
		t.Errorf("stale text-muted-foreground class still on remove button: %q", btnOpen)
	}
	if strings.Contains(btnOpen, "hover:underline") {
		t.Errorf("stale hover:underline still on remove button: %q", btnOpen)
	}
}

func TestImageUpload_RemoveButton_HiddenWhenNoCurrentURL(t *testing.T) {
	html := renderToString(t, ImageUpload(ImageUploadProps{
		Name:  "exercise",
		Label: "Image",
		// CurrentURL is empty: nothing has been uploaded yet.
	}))

	if strings.Contains(html, `id="image-remove-exercise"`) {
		t.Error("image-remove button must not render when CurrentURL is empty")
	}
	if strings.Contains(html, `name="clear_image"`) {
		t.Error("clear_image hidden field must not render when CurrentURL is empty")
	}
}

func TestImageUpload_RemoveFieldName_OverridesDefault(t *testing.T) {
	// RemoveFieldName prop should be threaded through to the
	// hidden field's name attribute and the data attribute.
	html := renderToString(t, ImageUpload(ImageUploadProps{
		Name:            "photo",
		Label:           "Photo",
		CurrentURL:      "https://pub-test.r2.dev/weight/abc.jpg",
		RemoveFieldName: "remove_photo",
	}))

	if !strings.Contains(html, `name="remove_photo"`) {
		t.Error("expected hidden field name to be remove_photo when RemoveFieldName is set")
	}
	if !strings.Contains(html, `data-remove-field="remove_photo"`) {
		t.Error("expected data-remove-field to be remove_photo when RemoveFieldName is set")
	}
}

func TestImageUpload_RemoveLabel_OverridesDefault(t *testing.T) {
	html := renderToString(t, ImageUpload(ImageUploadProps{
		Name:        "photo",
		Label:       "Photo",
		CurrentURL:  "https://pub-test.r2.dev/weight/abc.jpg",
		RemoveLabel: "Remove current photo",
	}))

	if !strings.Contains(html, `title="Remove current photo"`) {
		t.Error("expected button title to reflect the RemoveLabel prop")
	}
	if !strings.Contains(html, `aria-label="Remove current photo"`) {
		t.Error("expected button aria-label to reflect the RemoveLabel prop")
	}
	if !strings.Contains(html, `data-remove-label="Remove current photo"`) {
		t.Error("expected data-remove-label to be RemoveLabel value")
	}
}

func TestImageUpload_UndoState_PresentInWidgetScript(t *testing.T) {
	// The widget ships the 'remove img' and 'undo' text labels
	// inline so the button can swap between them without a server
	// round-trip. Both must be present in the rendered JS so the
	// setRemoveState helper can use them.
	html := renderToString(t, ImageUpload(ImageUploadProps{
		Name:       "exercise",
		Label:      "Image",
		CurrentURL: "https://pub-test.r2.dev/exercises/abc.jpg",
	}))

	if !strings.Contains(html, "var REMOVE_TEXT = 'Remove img'") {
		t.Error("expected inline REMOVE_TEXT constant in the widget's script")
	}
	if !strings.Contains(html, "var UNDO_TEXT = 'Undo'") {
		t.Error("expected inline UNDO_TEXT constant in the widget's script")
	}
	// The confirm dialog wiring must be present so a click on
	// the button opens the global #confirm-dialog rather than
	// firing a destructive action immediately.
	if !strings.Contains(html, "openConfirm") {
		t.Error("expected openConfirm helper in the widget's script")
	}
	if !strings.Contains(html, "confirm-dialog") {
		t.Error("expected the widget to reference the global #confirm-dialog")
	}
	// Regression guard: the OK path of openConfirm must call
	// dialog.close() before resolving true. Without that call
	// the dialog stays open after the user clicks Confirm.
	if !strings.Contains(html, "dialog.close()") {
		t.Error("openConfirm must call dialog.close() on the OK path so the modal dismisses after Confirm")
	}
}

func TestImageUpload_FileInputHasHtmxUploadAttributes(t *testing.T) {
	// The file input is rendered with htmx attributes that make
	// it POST to the upload endpoint on change, with the response
	// (a views.Toast) swapped into #toaster. The server signals
	// the new storage keys back via the HX-Trigger header.
	html := renderToString(t, ImageUpload(ImageUploadProps{
		Name:           "exercise",
		Label:          "Image",
		UploadEndpoint: "/admin/exercises/image-upload",
	}))

	for _, want := range []string{
		`hx-post="/admin/exercises/image-upload"`,
		`hx-encoding="multipart/form-data"`,
		`hx-target="#toaster"`,
		`hx-swap="beforeend"`,
		`hx-trigger="change"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("expected file input to have %q", want)
		}
	}
}

func TestImageUpload_FileInputHasNameAttribute(t *testing.T) {
	// Regression guard: htmx serialises the triggering form via
	// `new FormData(form)` on the upload POST, and FormData only
	// includes form controls with a `name` attribute. A file input
	// without `name` is dropped from the multipart payload before
	// it reaches the server, so the route's `c.FormFile("file")`
	// returns http.ErrMissingFile and the upload silently fails.
	// The default FileFieldName must match the field name the
	// route reads.
	t.Run("default name is file", func(t *testing.T) {
		html := renderToString(t, ImageUpload(ImageUploadProps{
			Name:           "exercise",
			Label:          "Image",
			UploadEndpoint: "/admin/exercises/image-upload",
		}))
		if !strings.Contains(html, `type="file"`) {
			t.Fatal("expected a file input in the rendered HTML")
		}
		// Locate the file input and assert the name attribute is
		// present on the same tag. We deliberately don't just
		// substring-match `name="file"` because the hidden
		// `img_key` / `img_key_original` inputs use the same
		// pattern, so we anchor on `type="file"` to scope the
		// assertion to the right element.
		fileStart := strings.Index(html, `type="file"`)
		if fileStart < 0 {
			t.Fatal("could not locate file input in rendered HTML")
		}
		// Search forward for the next `>` that closes the input's
		// opening tag.
		tagEnd := strings.Index(html[fileStart:], ">")
		if tagEnd < 0 {
			t.Fatal("could not locate end of file input opening tag")
		}
		tagOpen := html[fileStart : fileStart+tagEnd]
		if !strings.Contains(tagOpen, `name="file"`) {
			t.Errorf("file input is missing name=\"file\"; opening tag: %q", tagOpen)
		}
	})

	t.Run("custom FileFieldName is threaded through", func(t *testing.T) {
		html := renderToString(t, ImageUpload(ImageUploadProps{
			Name:           "exercise",
			Label:          "Image",
			UploadEndpoint: "/admin/exercises/image-upload",
			FileFieldName:  "photo",
		}))
		fileStart := strings.Index(html, `type="file"`)
		if fileStart < 0 {
			t.Fatal("could not locate file input in rendered HTML")
		}
		tagEnd := strings.Index(html[fileStart:], ">")
		if tagEnd < 0 {
			t.Fatal("could not locate end of file input opening tag")
		}
		tagOpen := html[fileStart : fileStart+tagEnd]
		if !strings.Contains(tagOpen, `name="photo"`) {
			t.Errorf("file input did not pick up FileFieldName=photo; opening tag: %q", tagOpen)
		}
		if strings.Contains(tagOpen, `name="file"`) {
			t.Errorf("file input still has default name=\"file\" despite FileFieldName override; opening tag: %q", tagOpen)
		}
	})
}

func TestImageUpload_ImageUploadedEventListener(t *testing.T) {
	// The widget's JS listens for the `image-uploaded` custom
	// event (fired by the server via the HX-Trigger response
	// header on a successful upload) and copies the new keys
	// from evt.detail into the form's hidden inputs.
	html := renderToString(t, ImageUpload(ImageUploadProps{
		Name:       "exercise",
		Label:      "Image",
		CurrentURL: "https://pub-test.r2.dev/exercises/abc.jpg",
	}))

	if !strings.Contains(html, "image-uploaded") {
		t.Error("expected the widget's JS to listen for the 'image-uploaded' event")
	}
	if !strings.Contains(html, "display_key") {
		t.Error("expected the listener to read display_key from evt.detail")
	}
	if !strings.Contains(html, "original_key") {
		t.Error("expected the listener to read original_key from evt.detail")
	}
	if !strings.Contains(html, "displayHidden.value = evt.detail.display_key") {
		t.Error("expected the listener to copy display_key into displayHidden.value")
	}
	if !strings.Contains(html, "originalHidden.value = evt.detail.original_key") {
		t.Error("expected the listener to copy original_key into originalHidden.value")
	}
}

func TestImageUpload_NoInlineToastIcons(t *testing.T) {
	// Regression guard: the widget used to inline four icon SVGs
	// (success/error/warning/default) in a TOAST_ICONS constant
	// inside the script. Those are now defined once in the
	// layout's toastTemplate elements and cloned by the
	// weight widget's JS; the image widget doesn't need them at
	// all because the upload endpoint returns the toast markup
	// directly. Make sure nobody re-adds the duplication.
	html := renderToString(t, ImageUpload(ImageUploadProps{
		Name:       "exercise",
		Label:      "Image",
		CurrentURL: "https://pub-test.r2.dev/exercises/abc.jpg",
	}))

	// Find the widget's <script> tag and inspect only that —
	// not the whole HTML, which legitimately contains the icon
	// SVGs from the layout's toastTemplate elements.
	scriptStart := strings.Index(html, "<script>")
	scriptEnd := strings.Index(html, "</script>")
	if scriptStart < 0 || scriptEnd < 0 {
		t.Fatal("widget script not found")
	}
	widgetScript := html[scriptStart:scriptEnd]

	if strings.Contains(widgetScript, "TOAST_ICONS") {
		t.Error("widget script must not define a TOAST_ICONS constant; the toast markup is shared via the layout's toastTemplate elements")
	}
	if strings.Contains(widgetScript, "var showToast") {
		t.Error("widget script must not define a showToast function; the image upload endpoint returns the toast markup directly via htmx")
	}
	// The old inline SVGs in the widget's script are gone.
	if strings.Contains(widgetScript, "TRASH_SVG") {
		t.Error("widget script still has the stale TRASH_SVG constant from a previous design")
	}
}

func TestImageUpload_NoFetchCall(t *testing.T) {
	// Regression guard: the widget used to do its own
	// `fetch(endpoint, { method: 'POST', body: fd })` plus a
	// local FileReader preview. Both have been replaced by
	// htmx — the file input has hx-post/hx-trigger so htmx
	// does the upload, and the server's response (a toast)
	// is the only feedback the user sees. If anyone re-adds
	// a `fetch(` in this widget they'll be duplicating work
	// the server is already doing.
	html := renderToString(t, ImageUpload(ImageUploadProps{
		Name:       "exercise",
		Label:      "Image",
		CurrentURL: "https://pub-test.r2.dev/exercises/abc.jpg",
	}))

	scriptStart := strings.Index(html, "<script>")
	scriptEnd := strings.Index(html, "</script>")
	if scriptStart < 0 || scriptEnd < 0 {
		t.Fatal("widget script not found")
	}
	widgetScript := html[scriptStart:scriptEnd]

	if strings.Contains(widgetScript, "fetch(") {
		t.Error("widget script must not call fetch(); htmx handles the upload via hx-post/hx-trigger on the file input")
	}
	if strings.Contains(widgetScript, "FileReader") {
		t.Error("widget script must not use FileReader; there's no client-side preview anymore — the server's response is the only feedback")
	}
	if strings.Contains(widgetScript, "new FormData") {
		t.Error("widget script must not build FormData; htmx constructs the multipart body from the file input")
	}
}

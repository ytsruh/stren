package components

import (
	"strings"
	"testing"
)

// ratioImageContract asserts the markup every ratio-enforced
// image component must emit: a relative, overflow-hidden frame
// carrying the ratio's aspect utility, plus an absolutely
// positioned img that fills the frame with object-fit: cover.
// See image.templ for the expected upload ratios.
func ratioImageContract(t *testing.T, html, wantAspect string) {
	t.Helper()
	if !strings.Contains(html, "relative w-full overflow-hidden rounded-lg bg-muted") {
		t.Errorf("expected the standard ratio-image frame classes, got:\n%s", html)
	}
	if !strings.Contains(html, wantAspect) {
		t.Errorf("expected aspect class %q in rendered HTML, got:\n%s", wantAspect, html)
	}
	if !strings.Contains(html, `class="absolute inset-0 size-full object-cover"`) {
		t.Errorf("expected the fill img with object-cover, got:\n%s", html)
	}
}

func TestLandscapeImage_RendersSixteenNineFrame(t *testing.T) {
	html := renderToString(t, LandscapeImage(ImageProps{
		Src: "https://pub-test.r2.dev/exercises/abc.jpg",
		Alt: "Bench Press",
	}))
	ratioImageContract(t, html, "aspect-video")
	if !strings.Contains(html, `src="https://pub-test.r2.dev/exercises/abc.jpg"`) {
		t.Errorf("expected Src rendered as the img src, got:\n%s", html)
	}
	if !strings.Contains(html, `alt="Bench Press"`) {
		t.Errorf("expected Alt rendered as the img alt, got:\n%s", html)
	}
	if strings.Contains(html, "TemplUnsupportedStyleAttributeValue") {
		t.Error("rendered HTML contains the unsupported-URL sentinel — Src must be wrapped in templ.SafeURL")
	}
}

func TestPortraitImage_RendersThreeFourFrame(t *testing.T) {
	html := renderToString(t, PortraitImage(ImageProps{
		Src: "https://pub-test.r2.dev/weight/abc.jpg",
		Alt: "Before",
	}))
	ratioImageContract(t, html, "aspect-3/4")
}

func TestBannerImage_RendersThreeOneFrame(t *testing.T) {
	html := renderToString(t, BannerImage(ImageProps{
		Src: "https://pub-test.r2.dev/exercises/abc.jpg",
		Alt: "Bench Press",
	}))
	ratioImageContract(t, html, "aspect-[3/1]")
	if !strings.Contains(html, `alt="Bench Press"`) {
		t.Errorf("expected Alt rendered as the img alt, got:\n%s", html)
	}
}

func TestSquareImage_RendersSquareFrame(t *testing.T) {
	html := renderToString(t, SquareImage(ImageProps{
		Src: "https://pub-test.r2.dev/exercises/abc.jpg",
		Alt: "Avatar",
	}))
	ratioImageContract(t, html, "aspect-square")
}

func TestImage_ClassPropAppended(t *testing.T) {
	// Class lets callers add context-specific utilities (spacing,
	// max-width) without the component needing a variant each.
	html := renderToString(t, LandscapeImage(ImageProps{
		Src:   "https://pub-test.r2.dev/exercises/abc.jpg",
		Alt:   "Bench Press",
		Class: "mb-4 max-w-2xl",
	}))
	if !strings.Contains(html, "mb-4 max-w-2xl") {
		t.Errorf("expected Class utilities appended to the frame div, got:\n%s", html)
	}
}

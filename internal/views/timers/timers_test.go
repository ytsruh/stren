package timers

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"stren/internal/views"
)

func renderToString(t *testing.T, c templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	if err := c.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	return buf.String()
}

func TestTimerPage_RendersTabsWithTimerActive(t *testing.T) {
	data := views.PageData{Title: "Timer", IsAuthenticated: true, CurrentPath: "/timer"}
	html := renderToString(t, TimerPage(data))

	if !strings.Contains(html, `role="tablist"`) {
		t.Fatal("expected tablist")
	}
	if !strings.Contains(html, `id="timers-tab-timer"`) || !strings.Contains(html, `id="timers-tab-emom"`) {
		t.Fatal("expected both Timer and EMOM tab buttons")
	}
	timerActive := regexp.MustCompile(`id="timers-tab-timer"[^>]*aria-selected="true"`)
	emomActive := regexp.MustCompile(`id="timers-tab-emom"[^>]*aria-selected="true"`)
	if !timerActive.MatchString(html) {
		t.Fatal("expected Timer tab to be active")
	}
	if emomActive.MatchString(html) {
		t.Fatal("expected EMOM tab to be inactive")
	}
	emomPanelHidden := regexp.MustCompile(`<div[^>]*id="timers-panel-emom"[^>]*hidden`)
	if !emomPanelHidden.MatchString(html) {
		t.Fatal("expected EMOM panel to be hidden")
	}
}

func TestEMOMPage_RendersTabsWithEMOMActive(t *testing.T) {
	data := views.PageData{Title: "EMOM Timer", IsAuthenticated: true, CurrentPath: "/timer/emom"}
	html := renderToString(t, EMOMPage(data))

	timerActive := regexp.MustCompile(`id="timers-tab-timer"[^>]*aria-selected="true"`)
	emomActive := regexp.MustCompile(`id="timers-tab-emom"[^>]*aria-selected="true"`)
	if !emomActive.MatchString(html) {
		t.Fatal("expected EMOM tab to be active")
	}
	if timerActive.MatchString(html) {
		t.Fatal("expected Timer tab to be inactive")
	}
	timerPanelHidden := regexp.MustCompile(`<div[^>]*id="timers-panel-timer"[^>]*hidden`)
	if !timerPanelHidden.MatchString(html) {
		t.Fatal("expected Timer panel to be hidden")
	}
}


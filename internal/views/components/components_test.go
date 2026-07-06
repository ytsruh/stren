package components

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
)

func renderToString(t *testing.T, component interface {
	Render(context.Context, io.Writer) error
}) string {
	t.Helper()
	var buf bytes.Buffer
	if err := component.Render(context.Background(), &buf); err != nil {
		t.Fatalf("failed to render component: %v", err)
	}
	return buf.String()
}

func TestIcon(t *testing.T) {
	html := renderToString(t, Icon(IconProps{Name: "check", Size: 18}))
	if !strings.Contains(html, `width="18"`) {
		t.Error("expected width 18")
	}
	if !strings.Contains(html, `height="18"`) {
		t.Error("expected height 18")
	}
	if !strings.Contains(html, `lucide-check`) {
		t.Error("expected check icon")
	}
	if !strings.Contains(html, `<path d="M20 6 9 17l-5-5"></path>`) {
		t.Error("expected check icon path")
	}
}

func TestIconAlias(t *testing.T) {
	html := renderToString(t, Icon(IconProps{Name: "delete", Size: 24}))
	if !strings.Contains(html, `lucide-trash`) {
		t.Error("expected delete aliased to trash")
	}
	if !strings.Contains(html, `<path d="M3 6h18"></path>`) {
		t.Error("expected trash icon path")
	}
}

func TestIconDefaultSize(t *testing.T) {
	html := renderToString(t, Icon(IconProps{Name: "check"}))
	if !strings.Contains(html, `width="24"`) {
		t.Error("expected default width 24")
	}
}

func TestIconUnknownName(t *testing.T) {
	html := renderToString(t, Icon(IconProps{Name: "unknown-icon", Size: 24}))
	if !strings.Contains(html, `lucide-file-question-mark`) {
		t.Error("expected fallback icon for unknown name")
	}
}

func TestCard(t *testing.T) {
	html := renderToString(t, Card(CardProps{ID: "test-card", Class: "custom-class"}, nil))
	if !strings.Contains(html, `id="test-card"`) {
		t.Error("expected card id")
	}
	if !strings.Contains(html, `class="card custom-class"`) {
		t.Error("expected card class")
	}
}

func TestCardHeader(t *testing.T) {
	html := renderToString(t, CardHeader(CardHeaderProps{Class: "custom-header"}, nil))
	if !strings.Contains(html, `<header class="custom-header"`) {
		t.Error("expected header with class")
	}
}

func TestCardSection(t *testing.T) {
	html := renderToString(t, CardSection(CardSectionProps{Class: "custom-section"}, nil))
	if !strings.Contains(html, `<section class="custom-section"`) {
		t.Error("expected section with class")
	}
}

func TestStatCard(t *testing.T) {
	html := renderToString(t, StatCard(StatCardProps{Label: "Total Sets", Value: "42"}))
	if !strings.Contains(html, "42") {
		t.Error("expected value")
	}
	if !strings.Contains(html, "Total Sets") {
		t.Error("expected label")
	}
	if !strings.Contains(html, `class="card flex flex-col justify-around gap-0 md:gap-2"`) {
		t.Error("expected card class")
	}
}

func TestResolveIconName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"delete", "trash"},
		{"remove", "trash"},
		{"edit", "pencil"},
		{"close", "x"},
		{"check", "check"},
	}

	for _, tt := range tests {
		got := resolveIconName(tt.input)
		if got != tt.expected {
			t.Errorf("resolveIconName(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

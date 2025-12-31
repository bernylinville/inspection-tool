package report

import (
	"strings"
	"testing"
	"time"
)

func TestNewRegistry(t *testing.T) {
	t.Run("with nil timezone uses default", func(t *testing.T) {
		r := NewRegistry(nil, "", "")

		if r == nil {
			t.Fatal("expected non-nil registry")
		}

		// Should have both excel and html writers
		if len(r.writers) != 2 {
			t.Errorf("expected 2 writers, got %d", len(r.writers))
		}

		// Verify writers are registered
		if _, ok := r.writers["excel"]; !ok {
			t.Error("expected excel writer to be registered")
		}
		if _, ok := r.writers["html"]; !ok {
			t.Error("expected html writer to be registered")
		}
	})

	t.Run("with custom timezone", func(t *testing.T) {
		tz, _ := time.LoadLocation("America/New_York")
		r := NewRegistry(tz, "", "")

		if r == nil {
			t.Fatal("expected non-nil registry")
		}

		// Should still have both writers
		if len(r.writers) != 2 {
			t.Errorf("expected 2 writers, got %d", len(r.writers))
		}
	})

	t.Run("with custom template path", func(t *testing.T) {
		r := NewRegistry(nil, "/custom/template.html", "")

		if r == nil {
			t.Fatal("expected non-nil registry")
		}

		// HTML writer should be created with custom template path
		htmlWriter, ok := r.writers["html"]
		if !ok {
			t.Error("expected html writer to be registered")
		}
		if htmlWriter.Format() != "html" {
			t.Errorf("expected html format, got %s", htmlWriter.Format())
		}
	})
}

func TestRegistry_Get_Excel(t *testing.T) {
	r := NewRegistry(nil, "", "")

	writer, err := r.Get("excel")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if writer == nil {
		t.Fatal("expected non-nil writer")
	}
	if writer.Format() != "excel" {
		t.Errorf("expected format 'excel', got %q", writer.Format())
	}
}

func TestRegistry_Get_HTML(t *testing.T) {
	r := NewRegistry(nil, "", "")

	writer, err := r.Get("html")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if writer == nil {
		t.Fatal("expected non-nil writer")
	}
	if writer.Format() != "html" {
		t.Errorf("expected format 'html', got %q", writer.Format())
	}
}

func TestRegistry_Get_Unknown(t *testing.T) {
	r := NewRegistry(nil, "", "")

	writer, err := r.Get("pdf")

	if err == nil {
		t.Fatal("expected error for unknown format")
	}
	if writer != nil {
		t.Error("expected nil writer for unknown format")
	}

	// Error message should mention the unsupported format
	if !strings.Contains(err.Error(), "pdf") {
		t.Errorf("error message should mention the unsupported format 'pdf': %v", err)
	}

	// Error message should list supported formats
	if !strings.Contains(err.Error(), "excel") || !strings.Contains(err.Error(), "html") {
		t.Errorf("error message should list supported formats: %v", err)
	}
}

func TestRegistry_Get_CaseInsensitive(t *testing.T) {
	r := NewRegistry(nil, "", "")

	testCases := []struct {
		input    string
		expected string
	}{
		{"excel", "excel"},
		{"Excel", "excel"},
		{"EXCEL", "excel"},
		{"ExCeL", "excel"},
		{"html", "html"},
		{"HTML", "html"},
		{"Html", "html"},
		{" excel ", "excel"}, // with whitespace
		{" HTML ", "html"},   // with whitespace
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			writer, err := r.Get(tc.input)

			if err != nil {
				t.Fatalf("unexpected error for input %q: %v", tc.input, err)
			}
			if writer.Format() != tc.expected {
				t.Errorf("expected format %q, got %q", tc.expected, writer.Format())
			}
		})
	}
}

func TestRegistry_GetAll(t *testing.T) {
	r := NewRegistry(nil, "", "")

	formats := r.GetAll()

	if len(formats) != 2 {
		t.Errorf("expected 2 formats, got %d", len(formats))
	}

	// Should be sorted alphabetically
	expected := []string{"excel", "html"}
	for i, format := range expected {
		if formats[i] != format {
			t.Errorf("expected formats[%d] = %q, got %q", i, format, formats[i])
		}
	}
}

func TestRegistry_Has(t *testing.T) {
	r := NewRegistry(nil, "", "")

	testCases := []struct {
		format   string
		expected bool
	}{
		{"excel", true},
		{"html", true},
		{"pdf", false},
		{"Excel", true},   // case insensitive
		{"HTML", true},    // case insensitive
		{" excel ", true}, // with whitespace
		{"", false},
		{"   ", false},
	}

	for _, tc := range testCases {
		t.Run(tc.format, func(t *testing.T) {
			result := r.Has(tc.format)
			if result != tc.expected {
				t.Errorf("Has(%q) = %v, expected %v", tc.format, result, tc.expected)
			}
		})
	}
}

func TestRegistry_Get_EmptyFormat(t *testing.T) {
	r := NewRegistry(nil, "", "")

	writer, err := r.Get("")

	if err == nil {
		t.Fatal("expected error for empty format")
	}
	if writer != nil {
		t.Error("expected nil writer for empty format")
	}
}

func TestRegistry_Get_WhitespaceFormat(t *testing.T) {
	r := NewRegistry(nil, "", "")

	writer, err := r.Get("   ")

	if err == nil {
		t.Fatal("expected error for whitespace-only format")
	}
	if writer != nil {
		t.Error("expected nil writer for whitespace-only format")
	}
}

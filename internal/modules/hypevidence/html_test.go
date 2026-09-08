package hypevidence

import (
	"strings"
	"testing"
)

func TestNormalizeHTMLExcludesPresentationAndIsDeterministic(t *testing.T) {
	raw := []byte(`<html><head><style>ignore</style><script>ignore()</script></head><body><h1>Event &amp; Result</h1><p>Text <b>with</b> spaces.</p></body></html>`)
	want := "Event & Result Text with spaces."
	first, err := NormalizeHTML(raw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NormalizeHTML(raw)
	if err != nil || first != second || first != want {
		t.Fatalf("normalization = %q, %q; want %q", first, second, want)
	}
}

func TestReadAllBoundedRejectsOversizedContent(t *testing.T) {
	if _, err := ReadAllBounded(strings.NewReader("12345"), 4); err == nil {
		t.Fatal("oversized content accepted")
	}
}

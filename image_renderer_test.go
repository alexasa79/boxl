package boxl

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderPNGUsesDeclarationsCollectedInFirstPass(t *testing.T) {
	got, err := RenderPNG("a -> b\na[\"hello\"]")
	if err != nil {
		t.Fatal(err)
	}
	want, err := RenderPNG("a[\"hello\"] -> b")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("forward declaration produces a different diagram")
	}
}

func TestRenderPNGRejectsAmbiguousDeclarations(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{"conflicting text", "a[\"one\"]\na[\"two\"]", "conflicting text"},
		{"text and children", "a[\"one\"]\na[b]", "both text and nested"},
		{"two parents", "a[x]\nb[x]", "belongs to both"},
		{"boundary crossing", "a[x]\nx -> b", "crosses the boundary"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := RenderPNG(test.source)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want text %q", err, test.want)
			}
		})
	}
}

func TestRenderPNGAllowsConnectionCycles(t *testing.T) {
	if _, err := RenderPNG("a -> b\nb -> a"); err != nil {
		t.Fatal(err)
	}
}

package main

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"boxl"
)

func TestRunDefaultsToPNGFile(t *testing.T) {
	for _, mode := range []string{"inline", "stdin", "file"} {
		t.Run(mode, func(t *testing.T) {
			t.Chdir(t.TempDir())
			var args []string
			stdin := ""
			switch mode {
			case "inline":
				args = []string{"-e", "a -> b"}
			case "stdin":
				stdin = "a -> b"
			case "file":
				if err := os.WriteFile("diagram.boxl", []byte("a -> b"), 0o600); err != nil {
					t.Fatal(err)
				}
				args = []string{"diagram.boxl"}
			}
			var stdout, stderr bytes.Buffer
			if err := run(args, strings.NewReader(stdin), &stdout, &stderr); err != nil {
				t.Fatal(err)
			}
			output, err := os.ReadFile("out.png")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := png.Decode(bytes.NewReader(output)); err != nil {
				t.Fatalf("out.png is not a valid PNG: %v", err)
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q", stdout.String())
			}
		})
	}
}

func TestRunEmptyInlineSourceDoesNotReadStdin(t *testing.T) {
	t.Chdir(t.TempDir())
	var stdout, stderr bytes.Buffer
	if err := run([]string{"-e", ""}, nil, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunKittyImage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"-kitty", "-e", "a -> b"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(stdout.Bytes(), []byte("\x1b_Ga=T,f=100")) {
		t.Fatalf("stdout does not contain a Kitty image: %q", stdout.Bytes())
	}
}

func TestRunKittyImageWithMultilineText(t *testing.T) {
	source := "a['hello world'] -> b['second box\n    box language is awesome'] -> c['third box']"
	var stdout, stderr bytes.Buffer
	if err := run([]string{"-kitty", "-e", source}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(stdout.Bytes(), []byte("\x1b_Ga=T,f=100")) {
		t.Fatalf("stdout does not contain a Kitty image: %q", stdout.Bytes())
	}
}

func TestRunReadsAndWritesFiles(t *testing.T) {
	directory := t.TempDir()
	inputPath := filepath.Join(directory, "diagram.boxl")
	outputPath := filepath.Join(directory, "diagram.png")
	if err := os.WriteFile(inputPath, []byte("a[\"file\"]"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{"-o", outputPath, inputPath}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	output, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	want, err := boxl.RenderPNG("a[\"file\"]")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(output, want) {
		t.Fatal("output file does not match the rendered PNG")
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunRejectsInlineSourceAndFile(t *testing.T) {
	err := run([]string{"-e", "a", "diagram.boxl"}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "cannot be used together") {
		t.Fatalf("error = %v", err)
	}
}

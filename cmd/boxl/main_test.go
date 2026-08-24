package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunInlineSource(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"-e", "a -> b"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "-->") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunEmptyInlineSourceDoesNotReadStdin(t *testing.T) {
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
	outputPath := filepath.Join(directory, "diagram.txt")
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
	if !strings.Contains(string(output), "file") {
		t.Fatalf("output file = %q", output)
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

package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"boxl"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "boxl: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("boxl", flag.ContinueOnError)
	flags.SetOutput(stderr)
	inline := flags.String("e", "", "render inline box language source")
	kitty := flags.Bool("kitty", false, "display as an image using the Kitty graphics protocol")
	outputPath := flags.String("o", "", "write output to file")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "usage: boxl [-kitty] [-o output] [file]")
		fmt.Fprintln(stderr, "       boxl [-kitty] [-o output] -e 'a -> b'")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	inlineSet := false
	flags.Visit(func(option *flag.Flag) {
		if option.Name == "e" {
			inlineSet = true
		}
	})
	if flags.NArg() > 1 {
		return fmt.Errorf("expected at most one input file")
	}
	if inlineSet && flags.NArg() != 0 {
		return fmt.Errorf("-e and an input file cannot be used together")
	}

	input := stdin
	var inputFile *os.File
	if inlineSet {
		input = strings.NewReader(*inline)
	} else if flags.NArg() == 1 {
		var err error
		inputFile, err = os.Open(flags.Arg(0))
		if err != nil {
			return fmt.Errorf("open input: %w", err)
		}
		defer inputFile.Close()
		input = inputFile
	}

	var rendered bytes.Buffer
	transform := boxl.Transform
	if *kitty {
		transform = boxl.TransformKitty
		if *outputPath == "" {
			if cellSize, ok := detectKittyCellSize(stdout); ok {
				transform = func(input io.Reader, output io.Writer) error {
					return boxl.TransformKittyAtCellSize(input, output, cellSize)
				}
			}
		}
	}
	if err := transform(input, &rendered); err != nil {
		return err
	}
	if *outputPath == "" {
		_, err := rendered.WriteTo(stdout)
		return err
	}
	outputFile, err := os.Create(*outputPath)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	if _, err := rendered.WriteTo(outputFile); err != nil {
		outputFile.Close()
		return fmt.Errorf("write output: %w", err)
	}
	if err := outputFile.Close(); err != nil {
		return fmt.Errorf("close output: %w", err)
	}
	return nil
}

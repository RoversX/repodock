package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/roversx/repodock/internal/buildinfo"
	"github.com/roversx/repodock/internal/tui"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "repodock: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) > 0 {
		switch args[0] {
		case "version":
			_, err := fmt.Fprintln(stdout, buildinfo.LongWithCopyright())
			return err
		case "help":
			printUsage(stdout)
			return nil
		}
	}

	fs := flag.NewFlagSet("repodock", flag.ContinueOnError)
	fs.SetOutput(stderr)

	showVersion := fs.Bool("version", false, "print version and exit")
	showHelp := fs.Bool("help", false, "show help")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printUsage(stdout)
			return nil
		}
		return err
	}

	if *showHelp {
		printUsage(stdout)
		return nil
	}

	if *showVersion {
		_, err := fmt.Fprintln(stdout, buildinfo.LongWithCopyright())
		return err
	}

	if fs.NArg() > 0 {
		return fmt.Errorf("unknown argument: %s", fs.Arg(0))
	}

	return tui.Run()
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, "RepoDock %s\n\n", buildinfo.BinaryVersion())
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  repodock")
	fmt.Fprintln(w, "  repodock --version")
	fmt.Fprintln(w, "  repodock help")
}

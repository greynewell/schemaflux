package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/greynewell/pssg/internal/build"
	"github.com/greynewell/pssg/internal/config"
)

func main() {
	buildCmd := flag.NewFlagSet("build", flag.ExitOnError)
	configPath := buildCmd.String("config", "pssg.yaml", "Path to config file")
	force := buildCmd.Bool("force", false, "Force full rebuild (ignore cache)")

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: pssg <command> [flags]\n\nCommands:\n  build    Build the static site\n")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "build":
		buildCmd.Parse(os.Args[2:])

		absConfig, err := filepath.Abs(*configPath)
		if err != nil {
			log.Fatalf("Error resolving config path: %v", err)
		}

		cfg, err := config.Load(absConfig)
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}

		builder := build.NewBuilder(cfg, *force)
		if err := builder.Build(); err != nil {
			log.Fatalf("Build failed: %v", err)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"
	"charm.land/log/v2"
	"github.com/SayYoungMan/tfui/internal/ui"
	"github.com/SayYoungMan/tfui/pkg/terraform"
)

func main() {
	binary := flag.String("binary", "terraform", "path or name of the terraform binary")
	workdir := flag.String("dir", "", "directory to find resources (defaults to current directory)")
	flag.Parse()

	closeLog := setUpLogging()
	defer closeLog()

	if _, err := exec.LookPath(*binary); err != nil {
		fmt.Fprintf(os.Stderr, "%q not found in PATH\n", *binary)
		os.Exit(1)
	}

	if *workdir == "" {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error occurred during start up: %v\n", err)
			os.Exit(1)
		}
		*workdir = wd
	}

	log.Info("starting tfui", "binary", *binary, "workdir", *workdir)

	ctx, cancel := context.WithCancel(context.Background())
	runner := terraform.NewTerraformRunner(*workdir, *binary)
	ch := runner.StreamPlan(ctx)

	m := ui.NewModel(runner, ch, cancel)
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error occurred while running program: %v\n", err)
		os.Exit(1)
	}
}

func setUpLogging() func() {
	if os.Getenv("TFUI_DEBUG") != "1" {
		log.SetOutput(io.Discard)
		return func() {}
	}

	f, err := os.OpenFile("tfui-debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open debug log: %v\n", err)
		log.SetOutput(io.Discard)
		return func() {}
	}

	log.SetOutput(f)
	log.SetLevel(log.DebugLevel)
	log.SetReportTimestamp(true)
	log.SetReportCaller(true)
	log.SetTimeFormat("15:04:05.000")

	log.Info("=== tfui debug log started ===")

	return func() { f.Close() }
}

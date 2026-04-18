package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"
	"github.com/SayYoungMan/tfui/internal/ui"
	"github.com/SayYoungMan/tfui/pkg/terraform"
)

func main() {
	binary := flag.String("binary", "terraform", "path or name of the terraform binary")
	workdir := flag.String("dir", "", "directory to find resources (defaults to current directory)")
	flag.Parse()

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

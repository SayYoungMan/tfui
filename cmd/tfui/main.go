package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"
	"github.com/SayYoungMan/tfui/internal/ui"
	"github.com/SayYoungMan/tfui/pkg/terraform"
)

func main() {
	// TODO: Take the binary name from flags
	binary := "terraform"
	if _, err := exec.LookPath(binary); err != nil {
		fmt.Fprintf(os.Stderr, "%q not found in PATH\n", binary)
		os.Exit(1)
	}

	workdir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error occurred during start up: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	runner := terraform.NewTerraformRunner(workdir, "terraform")
	ch := runner.StreamPlan(ctx)

	m := ui.NewModel(runner, ch, cancel)
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error occurred while running program: %v\n", err)
		os.Exit(1)
	}
}

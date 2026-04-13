package terraform

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

const MB = 1024 * 1024

// Provides ability to override exec.CommandContext for mock testing
type CommandFactory func(ctx context.Context, name string, args ...string) *exec.Cmd

type TerraformRunner struct {
	binary     string
	workdir    string
	cmdFactory CommandFactory
}

func NewTerraformRunner(workdir string, binary string) *TerraformRunner {
	if binary == "" {
		binary = "terraform"
	}
	return &TerraformRunner{
		binary:     binary,
		workdir:    workdir,
		cmdFactory: exec.CommandContext,
	}
}

func (tr *TerraformRunner) StreamPlan(ctx context.Context) <-chan StreamEvent {
	ch := make(chan StreamEvent)

	go func() {
		defer close(ch)

		args := []string{"plan", "-json"}
		cmd := tr.cmdFactory(ctx, tr.binary, args...)
		cmd.Dir = tr.workdir

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			ch <- StreamEvent{Error: fmt.Errorf("failed to pipe stdout: %w", err)}
			return
		}

		if err := cmd.Start(); err != nil {
			ch <- StreamEvent{Error: fmt.Errorf("failed to terraform plan: %w", err)}
			return
		}

		parser := NewParser()
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, MB), MB)

		for scanner.Scan() {
			event, err := parser.ParseLine(scanner.Bytes())
			if err != nil {
				ch <- StreamEvent{Error: err}
				continue
			}
			if event != nil {
				ch <- *event
			}
		}
		if err := scanner.Err(); err != nil {
			ch <- StreamEvent{Error: fmt.Errorf("scanner error: %w", err)}
		}

		if err := cmd.Wait(); err != nil {
			ch <- StreamEvent{Error: fmt.Errorf("terraform plan exited with error: %w", err)}
		}
	}()

	return ch
}

// Runs a Terraform command and streams raw stdout/stderr
func (tr *TerraformRunner) streamOutput(ctx context.Context, args []string) <-chan string {
	ch := make(chan string)

	go func() {
		defer close(ch)

		cmd := tr.cmdFactory(ctx, tr.binary, args...)
		cmd.Dir = tr.workdir

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			ch <- fmt.Sprintf("failed to pipe stdout: %v", err)
		}
		cmd.Stderr = cmd.Stdout

		if err := cmd.Start(); err != nil {
			ch <- fmt.Sprintf("failed to run %s %s: %v", tr.binary, strings.Join(args, " "), err)
		}

		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, MB), MB)

		for scanner.Scan() {
			ch <- scanner.Text()
		}

		if err := cmd.Wait(); err != nil {
			ch <- fmt.Sprintf("%s %s exited with error: %v", tr.binary, strings.Join(args, " "), err)
		}
	}()

	return ch
}

func (tr *TerraformRunner) Plan(ctx context.Context, targets []string) <-chan string {
	args := []string{"plan"}
	for _, t := range targets {
		args = append(args, fmt.Sprintf("-target=%s", t))
	}
	return tr.streamOutput(ctx, args)
}

func (tr *TerraformRunner) Apply(ctx context.Context, targets []string) <-chan string {
	args := []string{"apply", "-auto-approve"}
	for _, t := range targets {
		args = append(args, fmt.Sprintf("-target=%s", t))
	}
	return tr.streamOutput(ctx, args)
}

func (tr *TerraformRunner) Destroy(ctx context.Context, targets []string) <-chan string {
	args := []string{"destroy", "-auto-approve"}
	for _, t := range targets {
		args = append(args, fmt.Sprintf("-target=%s", t))
	}
	return tr.streamOutput(ctx, args)
}

func (tr *TerraformRunner) Taint(ctx context.Context, targets []string) <-chan string {
	ch := make(chan string)

	go func() {
		defer close(ch)

		for _, t := range targets {
			for line := range tr.streamOutput(ctx, []string{"taint", t}) {
				ch <- line
			}
		}
	}()

	return ch
}

func (tr *TerraformRunner) Untaint(ctx context.Context, targets []string) <-chan string {
	ch := make(chan string)

	go func() {
		defer close(ch)

		for _, t := range targets {
			for line := range tr.streamOutput(ctx, []string{"untaint", t}) {
				ch <- line
			}
		}
	}()

	return ch
}

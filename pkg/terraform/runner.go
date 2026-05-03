package terraform

import (
	"bufio"
	"context"
	"fmt"
	"os"
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

func (tr *TerraformRunner) streamJsonEvents(ctx context.Context, args []string) <-chan StreamEvent {
	ch := make(chan StreamEvent)

	go func() {
		defer close(ch)

		cmd := tr.cmdFactory(ctx, tr.binary, args...)
		cmd.Dir = tr.workdir
		cmd.Cancel = func() error {
			return cmd.Process.Signal(os.Interrupt)
		}

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

func (tr *TerraformRunner) StreamPlan(ctx context.Context) <-chan StreamEvent {
	return tr.streamJsonEvents(ctx, []string{"plan", "-json"})
}

func (tr *TerraformRunner) Plan(ctx context.Context, targets []string) <-chan StreamEvent {
	args := []string{"plan", "-json"}
	for _, t := range targets {
		args = append(args, fmt.Sprintf("-target=%s", t))
	}
	return tr.streamJsonEvents(ctx, args)
}

func (tr *TerraformRunner) Apply(ctx context.Context, targets []string) <-chan StreamEvent {
	args := []string{"apply", "-auto-approve", "-json"}
	for _, t := range targets {
		args = append(args, fmt.Sprintf("-target=%s", t))
	}
	return tr.streamJsonEvents(ctx, args)
}

func (tr *TerraformRunner) Destroy(ctx context.Context, targets []string) <-chan StreamEvent {
	args := []string{"destroy", "-auto-approve", "-json"}
	for _, t := range targets {
		args = append(args, fmt.Sprintf("-target=%s", t))
	}
	return tr.streamJsonEvents(ctx, args)
}

func (tr *TerraformRunner) Taint(ctx context.Context, targets []string) <-chan StreamEvent {
	return tr.streamPerResource(ctx, "taint", targets)
}

func (tr *TerraformRunner) Untaint(ctx context.Context, targets []string) <-chan StreamEvent {
	return tr.streamPerResource(ctx, "untaint", targets)
}

func (tr *TerraformRunner) streamPerResource(ctx context.Context, command string, targets []string) <-chan StreamEvent {
	ch := make(chan StreamEvent)

	go func() {
		defer close(ch)

		for _, t := range targets {
			if ctx.Err() != nil {
				return
			}
			ch <- StreamEvent{
				Type:     "apply_start",
				Resource: &Resource{Address: t},
			}

			cmd := tr.cmdFactory(ctx, tr.binary, command, t)
			cmd.Dir = tr.workdir
			output, err := cmd.CombinedOutput()

			if len(output) > 0 {
				ch <- StreamEvent{Message: strings.TrimSpace(string(output))}
			}

			eventType := "apply_complete"
			if err != nil {
				eventType = "apply_errored"
			}
			ch <- StreamEvent{
				Type:     eventType,
				Resource: &Resource{Address: t},
			}
		}
	}()

	return ch
}

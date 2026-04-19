# tfui
Interactive TUI for performing Terraform workflows

![demo](./demo.gif)

## Install

### Go Install
```bash
go install github.com/SayYoungMan/tfui/cmd/tfui@latest
```
requires *Go 1.22+*


## Usage

Run from any directory containing Terraform configuration:
 
```bash
tfui
```

Or run targetting different directory:
```bash
tfui --dir <relative-path>
```

Run with Opentofu:
```bash
tfui --binary tofu
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--dir` | current directory | Working directory with Terraform resources |
| `--binary` | `terraform` | Path or name of the Terraform binary |


## Roadmap

| Feature | Status |
|---|---|
| Persistent resource state | 🔲 Planned |
| Diff viewer | 🔲 Planned |
| Workspace switcher | 🔲 Planned |
| Resource Detail Viewer | 🔲 Planned |
| Module tree view | 🔲 Planned |
| Per resource action tracker | 🔲 Planned |

Those are some features in mind but not in order of importance

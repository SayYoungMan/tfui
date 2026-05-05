# tfui
[![Build & Test](https://github.com/SayYoungMan/tfui/actions/workflows/build_test.yml/badge.svg?branch=main)](https://github.com/SayYoungMan/tfui/actions/workflows/build_test.yml)
![GitHub Release](https://img.shields.io/github/v/release/SayYoungMan/tfui)
[![Go Report Card](https://goreportcard.com/badge/github.com/SayYoungMan/tfui)](https://goreportcard.com/report/github.com/SayYoungMan/tfui)

Interactive TUI for performing Terraform workflows

![demo](./demo.gif)

## Install

### HomeBrew (Mac, Linux)
```bash
brew tap SayYoungMan/tap
brew install tfui
```

### Scoop (Windows)
```powershell
scoop bucket add SayYoungMan https://github.com/SayYoungMan/scoop-bucket
scoop install tfui
```

### Go Install
```bash
go install github.com/SayYoungMan/tfui/cmd/tfui@latest
```

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
| Module tree view | ✅ Done (v0.1.0) |
| Resource Detail Viewer | ✅ Done (v0.2.0) |
| Per resource progress tracker | ✅ Done (v0.3.0) |
| Persistent resource state | 🔲 Planned |
| Diff viewer | 🔲 Planned |
| Workspace switcher | 🔲 Planned |
| Stress test for large input | 🔲 Planned |
| Analytics Report | 🔲 Planned |
| Terragrunt Support | 🔲 Planned |
| History Viewer | 🔲 Planned |
| Resource Dependency View| 🔲 Planned |


Those are some features in mind but not in order of importance

<div align="center">

<p align="center"><b>English</b> | <a href="README.ja.md">日本語</a></p>

<h1 align="center">why</h1>

<h3 align="center">Shows the cause of a failed command and how to fix it, in Japanese.</h3>

<p align="center">A CLI that explains why your last command failed. It works offline and does not use AI. The messages are in Japanese.</p>

<p align="center">
  <a href="https://www.npmjs.com/package/@lapius/why"><img alt="npm" src="https://img.shields.io/npm/v/@lapius/why?style=flat-square&color=cb3837&logo=npm"></a>
  <a href="https://github.com/Lapius7/why/actions/workflows/release.yml"><img alt="Release workflow" src="https://img.shields.io/github/actions/workflow/status/Lapius7/why/release.yml?label=release&style=flat-square&logo=githubactions&logoColor=white"></a>
  <a href="https://go.dev"><img alt="Go" src="https://img.shields.io/badge/lang-Go-00add8?logo=go&logoColor=white&style=flat-square"></a>
  <a href="LICENSE"><img alt="License" src="https://img.shields.io/github/license/Lapius7/why?style=flat-square&color=97ca00"></a>
  <a href="https://github.com/Lapius7/why/stargazers"><img alt="Stars" src="https://img.shields.io/github/stars/Lapius7/why?style=flat-square&color=e3b341&logo=github"></a>
  <a href="https://github.com/Lapius7/why/commits/main"><img alt="Last commit" src="https://img.shields.io/github/last-commit/Lapius7/why?style=flat-square&color=3b82f6"></a>
</p>

<p align="center"><a href="#quick-start">Quick start</a> · <a href="#usage">Usage</a> · <a href="#rules">Rules</a> · <a href="#development">Development</a></p>

</div>

---

```
$ git push
fatal: not a git repository (or any of the parent directories): .git
$ why
✗ git push  (終了コード 128)
  ~/work · たった今
  [128] 致命的エラー（git の fatal など）、または exit に不正な値が渡された

原因  今いるディレクトリは git リポジトリではない
対処  → リポジトリのディレクトリに移動する（pwd で現在地を確認）
      → 新しく始めるなら: git init
```

## Features

- **Offline, no AI**: matches the exit code, the command and the output against YAML rules.
- **Your own rules**: add rules in `~/.config/why/rules/*.yaml`. A rule with the same `id` overrides a built-in one.
- **Shell hooks**: zsh, bash and fish. The hook only records failed commands.
- **Output capture**: reads output from tmux history, from `why -r`, or from a pipe.
- **Rule files**: built-in rules for build, docker, generic, git, node, python and system errors.

## Quick start

### pnpm (recommended)

```sh
pnpm add -g @lapius/why
```

> [!NOTE]
> The package is built for pnpm. Do not install it with both npm and pnpm, or the same command exists twice. Use only `pnpm add -g`.

This installs a prebuilt binary for Linux and macOS (x64 and arm64). Go is not needed. After installing, add the hook from the table below to your shell config. On first run the shim replaces itself with the native binary, so later runs do not go through node.

### Installer (no npm, no Go)

Downloads the prebuilt binary from GitHub Releases to `~/.local/bin` and adds the hook to your shell config.

```sh
curl -fsSL https://raw.githubusercontent.com/Lapius7/why/main/install.sh | sh
```

Options: `--no-hook` (do not edit the shell config) and `--uninstall`. Set the install path with `WHY_BIN` and the version with `WHY_VERSION=v0.1.1`.

### From source

`go install github.com/Lapius7/why/cmd/why@latest`. From a clone, `./install.sh` (or `just install`) builds from source and adds the hook.

### Hook

| Shell | Line to add |
|---|---|
| zsh (`~/.zshrc`) | `eval "$(why init zsh)"` |
| bash (`~/.bashrc`) | `eval "$(why init bash)"` |
| fish (`~/.config/fish/config.fish`) | `why init fish \| source` |

## Usage

| Command | What it does |
|---|---|
| `why` | Explain the last failed command |
| `why -r` | Re-run the last command after confirmation and explain it with its output |
| `why -- <cmd>` | Run the command and explain it if it fails (keeps the exit code) |
| `<cmd> 2>&1 \| why` | Explain output passed through a pipe |
| `why 137` | Explain what an exit code means |
| `why rules` | List the loaded rules |

### Getting the error output

A shell does not save stderr, so the hook knows only the command, the exit code and the directory. The output comes from one of these:

- **Inside tmux**: reads the output of the last command from the screen history.
- **`why -r`**: re-runs the command and captures it. It refuses commands that may have side effects, such as `rm`, `git push`, `sudo` and redirects (`--force` overrides).
- **Pipe or `why --`**: captures from the start.

A re-run uses English messages (`LC_MESSAGES=C.UTF-8`) so they match the rules more easily. The main rules also match Japanese messages.

## Rules

Built-in rules are in `rules/*.yaml`. Rules in `~/.config/why/rules/*.yaml` are added, and one with the same `id` overrides the built-in rule.

```yaml
- id: my-port
  prog: 'node|npm'            # program name (regex, full match)
  cmd: 'run dev'              # whole command line (regex)
  code: [1]                   # exit codes
  output: 'EADDRINUSE.*?:(?P<port>\d+)'   # output (regex, with (?m))
  example: "Error: listen EADDRINUSE: address already in use :::3000"
  priority: 1                 # display order adjustment (optional)
  cause: "ポート {{port}} が使用中"
  fix:
    - "ss -ltnp 'sport = :{{port}}'"
```

- A rule shows only when all of its conditions match. The score is 2 for `output` and 1 for each other condition, plus `priority`. The top 3 are shown.
- Named groups in `output`, and `{{prog}}`, `{{cmd}}` and `{{code}}`, work in `cause` and `fix`.
- A `fix` line with an empty variable is not shown.
- A built-in rule with `output` needs an `example`, and `go test` checks that it matches.

### Record file

`${XDG_STATE_HOME:-~/.local/state}/why/last-<shell PID>` holds the exit code, the cwd and the command. Files older than 7 days are deleted automatically.

## Safety

> [!NOTE]
> Not verified here: the npm and installer routes, the shell hooks, and tmux capture. A binary built from this source was run once with piped output, and it printed the cause and fix shown above. Windows is not covered by the prebuilt packages.

## Development

```sh
go test ./...
just npm-pack   # builds .tgz files in dist/npm/
```

## Release

Pushing a `v*` tag makes GitHub Actions build the binaries, publish to npm (`@lapius/why` and per-platform `@lapius/why-<os>-<cpu>`) and attach them to a GitHub Release.

```sh
git tag v0.1.0 && git push origin v0.1.0
```

## License

[MIT](LICENSE)

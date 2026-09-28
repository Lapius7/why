package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Lapius7/why"
	"github.com/Lapius7/why/internal/record"
	"github.com/Lapius7/why/internal/render"
	"github.com/Lapius7/why/internal/rules"
	"github.com/Lapius7/why/internal/runner"
)

var version = "0.1.3"

const usage = `why - 失敗したコマンドの原因と対処法を表示する

使い方:
  why                  直前に失敗したコマンドを説明
  why -r               直前のコマンドを確認後に再実行し、出力も含めて説明
  why -- <cmd> ...     コマンドを実行し、失敗したら説明（終了コードは引き継ぐ）
  <cmd> 2>&1 | why     パイプで受け取った出力を説明
  why <終了コード>      終了コードの意味を説明
  why init <shell>     シェルフックを出力（zsh / bash / fish）
  why rules            読み込まれているルールの一覧

オプション:
  -r, --rerun      直前のコマンドを再実行する
  -y, --yes        再実行の確認を省略する
      --force      副作用がありそうなコマンドでも再実行する
      --no-color   色を付けない
  -h, --help       このヘルプ
  -v, --version    バージョン

セットアップ（シェルの設定ファイルに追記）:
  zsh:  eval "$(why init zsh)"
  bash: eval "$(why init bash)"
  fish: why init fish | source

ユーザールール: ~/.config/why/rules/*.yaml（同じ id は組み込みを上書き）
`

func main() {
	os.Exit(run(os.Args[1:]))
}

type opts struct {
	rerun, yes, force, color bool
	code                     int
	argv                     []string
}

func run(args []string) int {
	if len(args) > 0 {
		switch args[0] {
		case "init":
			if len(args) < 2 {
				fmt.Fprintln(os.Stderr, "why init <zsh|bash|fish>")
				return 2
			}
			return cmdInit(args[1])
		case "rules":
			return cmdRules()
		}
	}

	o := opts{code: -1, color: colorEnabled()}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			o.argv = args[i+1:]
			i = len(args)
		case a == "-h" || a == "--help":
			fmt.Print(usage + "\n" + lapiusFooter())
			return 0
		case a == "-v" || a == "--version":
			fmt.Print("why ", version, "\n", lapiusFooter())
			return 0
		case a == "-r" || a == "--rerun":
			o.rerun = true
		case a == "-y" || a == "--yes":
			o.yes = true
		case a == "--force":
			o.force = true
		case a == "--no-color":
			o.color = false
		case isNumber(a):
			o.code, _ = strconv.Atoi(a)
		default:
			fmt.Fprintf(os.Stderr, "why: 不明な引数 %q（why -h でヘルプ）\n", a)
			return 2
		}
	}

	set, warns, err := rules.Load(why.Rules, userRulesDir())
	if err != nil {
		fmt.Fprintln(os.Stderr, "why: 組み込みルールの読み込みに失敗:", err)
		return 1
	}
	for _, w := range warns {
		fmt.Fprintln(os.Stderr, "why: ルールを無視:", w)
	}
	p := render.Printer{W: os.Stdout, Color: o.color}

	switch {
	case len(o.argv) > 0:
		return runDirect(o, set, p)
	case stdinPiped() && !o.rerun:
		b, _ := io.ReadAll(io.LimitReader(os.Stdin, 4<<20))
		out := rules.Normalize(string(b))
		rep := render.Report{Code: o.code, HasOutput: strings.TrimSpace(out) != ""}
		rep.Results = set.Match(rules.Input{Code: o.code, Output: out}, 3)
		p.Print(addHints(rep, true))
		return 0
	case o.code >= 0:
		rep := render.Report{Code: o.code}
		rep.Results = set.Match(rules.Input{Code: o.code}, 3)
		p.Print(rep)
		return 0
	}
	return explainLast(o, set, p)
}

func runDirect(o opts, set *rules.Set, p render.Printer) int {
	cmdline := quoteArgs(o.argv)
	res := runner.Exec(o.argv, "")
	if res.Code == 0 {
		return 0
	}
	wd, _ := os.Getwd()
	out := rules.Normalize(res.Output)
	rep := render.Report{Cmd: cmdline, Cwd: wd, Code: res.Code, HasOutput: strings.TrimSpace(out) != ""}
	rep.Results = set.Match(rules.Input{Cmd: cmdline, Code: res.Code, Output: out}, 3)
	fmt.Fprintln(os.Stdout)
	p.Print(addHints(rep, true))
	return res.Code
}

func explainLast(o opts, set *rules.Set, p render.Printer) int {
	dir := record.Dir()
	record.Prune(dir, 7*24*time.Hour)
	rec, err := record.Load(dir, shellPID())
	if err != nil {
		fmt.Fprintln(os.Stderr, "why: 失敗したコマンドの記録がありません。")
		fmt.Fprintln(os.Stderr, "  フックが未設定なら: eval \"$(why init "+shellName()+")\" を設定ファイルに追記")
		fmt.Fprintln(os.Stderr, "  その場で調べるなら: why -- <コマンド>  /  <コマンド> 2>&1 | why")
		return 1
	}

	in := rules.Input{Cmd: rec.Cmd, Code: rec.Code}
	rep := render.Report{Cmd: rec.Cmd, Cwd: rec.Cwd, Code: rec.Code, When: rec.Time}

	if o.rerun {
		if reason, bad := runner.Dangerous(rec.Cmd); bad && !o.force {
			fmt.Fprintf(os.Stderr, "why: 再実行しません（%s）: %s\n", reason, rec.Cmd)
			fmt.Fprintln(os.Stderr, "  それでも実行するなら why -r --force")
			return 1
		}
		if !o.yes && !confirm(fmt.Sprintf("再実行します: %s\n  場所: %s\nよろしいですか？ [y/N] ", rec.Cmd, rec.Cwd)) {
			return 1
		}
		res := runner.Shell(rec.Cmd, rec.Cwd)
		fmt.Fprintln(os.Stdout)
		if res.Code == 0 {
			fmt.Println("再実行したところ成功しました（一時的な問題か、環境が変わった可能性）。")
			return 0
		}
		in.Code, rep.Code, rep.When = res.Code, res.Code, time.Time{}
		in.Output = rules.Normalize(res.Output)
	} else if out := tmuxOutput(rec.Cmd); out != "" {
		in.Output = out
	}

	rep.HasOutput = strings.TrimSpace(in.Output) != ""
	rep.Results = set.Match(in, 3)
	p.Print(addHints(rep, o.rerun))
	return 0
}

// captured は出力を取り込み済み（再実行・直接実行・パイプ）かどうか。
func addHints(r render.Report, captured bool) render.Report {
	if !captured && !r.HasOutput && r.Cmd != "" {
		r.Hints = append(r.Hints, "出力も含めて調べるには why -r（再実行）か <コマンド> 2>&1 | why")
	}
	if len(r.Results) == 0 && r.HasOutput {
		r.Hints = append(r.Hints, "よく出るエラーなら "+userRulesDir()+"/*.yaml にルールを追加できます")
	}
	return r
}

// tmuxOutput は tmux 内なら画面の履歴から直前のコマンド以降の出力を取り出す。
func tmuxOutput(cmd string) string {
	if os.Getenv("TMUX") == "" {
		return ""
	}
	args := []string{"capture-pane", "-p", "-J", "-S", "-300"}
	if pane := os.Getenv("TMUX_PANE"); pane != "" {
		args = append(args, "-t", pane)
	}
	b, err := exec.Command("tmux", args...).Output()
	if err != nil {
		return ""
	}
	s := rules.Normalize(string(b))
	// 最後の「why」実行行より前で、コマンドが最後に現れた位置以降を出力とみなす
	first := strings.SplitN(cmd, "\n", 2)[0]
	end := strings.LastIndex(s, "\n")
	if end < 0 {
		return ""
	}
	i := strings.LastIndex(s[:end], first)
	if i < 0 {
		return ""
	}
	out := s[i+len(first) : end]
	if j := strings.LastIndex(out, "\n"); j >= 0 {
		out = out[:j] // why を打ったプロンプト行を除く
	}
	return out
}

func confirm(prompt string) bool {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, "why: 端末がないため確認できません（-y で確認を省略）")
		return false
	}
	defer tty.Close()
	fmt.Fprint(tty, prompt)
	line, _ := bufio.NewReader(tty).ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "y" || line == "yes"
}

func cmdInit(sh string) int {
	b, err := why.Shell.ReadFile("shell/why." + sh)
	if err != nil {
		fmt.Fprintf(os.Stderr, "why: 未対応のシェル %q（zsh / bash / fish）\n", sh)
		return 2
	}
	os.Stdout.Write(b)
	return 0
}

func cmdRules() int {
	set, warns, err := rules.Load(why.Rules, userRulesDir())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	for _, w := range warns {
		fmt.Fprintln(os.Stderr, "why: ルールを無視:", w)
	}
	src := ""
	for _, r := range set.Rules {
		if r.Source != src {
			src = r.Source
			fmt.Printf("\n# %s\n", src)
		}
		fmt.Printf("  %-28s %s\n", r.ID, r.Cause)
	}
	fmt.Printf("\n合計 %d 件（ユーザールール: %s）\n", len(set.Rules), userRulesDir())
	return 0
}

func userRulesDir() string {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "why", "rules")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "why", "rules")
}

func stdinPiped() bool {
	st, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice == 0 && st.Mode()&os.ModeNamedPipe != 0 || st.Mode().IsRegular()
}

func colorEnabled() bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	st, err := os.Stdout.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

func shellName() string {
	s := filepath.Base(os.Getenv("SHELL"))
	switch s {
	case "zsh", "bash", "fish":
		return s
	}
	return "zsh"
}

// shellPID は why を起動したシェルの PID を返す。
// npm の JS シム経由だと親は node になるため、シムが WHY_PPID で渡す。
func shellPID() int {
	if n, err := strconv.Atoi(os.Getenv("WHY_PPID")); err == nil && n > 0 {
		return n
	}
	return os.Getppid()
}

func isNumber(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func quoteArgs(argv []string) string {
	q := make([]string, len(argv))
	for i, a := range argv {
		if a == "" || strings.ContainsAny(a, " \t\n'\"\\$`*?[]{}()<>|&;#~!") {
			q[i] = "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
		} else {
			q[i] = a
		}
	}
	return strings.Join(q, " ")
}

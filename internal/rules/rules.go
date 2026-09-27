// Package rules は YAML で書かれたルールを読み込み、失敗したコマンドと照合する。
package rules

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Rule は 1 つの既知の失敗パターン。
// prog / cmd / code / output のうち指定したものがすべて一致したときに当てはまる。
type Rule struct {
	ID       string   `yaml:"id"`
	Prog     string   `yaml:"prog"`   // プログラム名の正規表現（完全一致）
	Cmd      string   `yaml:"cmd"`    // コマンドライン全体の正規表現
	Code     []int    `yaml:"code"`   // 終了コード
	Output   string   `yaml:"output"` // 出力の正規表現（(?m) 付きで評価）
	Priority int      `yaml:"priority"`
	Cause    string   `yaml:"cause"`
	Fix      []string `yaml:"fix"`
	Example  string   `yaml:"example"` // output が一致すべき出力例（テスト用）

	Source string `yaml:"-"`

	progRe, cmdRe, outRe *regexp.Regexp
}

type Input struct {
	Cmd    string
	Code   int // 不明なら -1
	Output string
}

type Result struct {
	Rule  *Rule
	Cause string
	Fix   []string
	Score int
}

func (r *Rule) compile() error {
	var err error
	if r.ID == "" {
		return fmt.Errorf("id がない")
	}
	if r.Cause == "" {
		return fmt.Errorf("%s: cause がない", r.ID)
	}
	if r.Prog == "" && r.Cmd == "" && len(r.Code) == 0 && r.Output == "" {
		return fmt.Errorf("%s: 条件（prog/cmd/code/output）が 1 つもない", r.ID)
	}
	if r.Prog != "" {
		if r.progRe, err = regexp.Compile(`^(?:` + r.Prog + `)$`); err != nil {
			return fmt.Errorf("%s: prog: %w", r.ID, err)
		}
	}
	if r.Cmd != "" {
		if r.cmdRe, err = regexp.Compile(r.Cmd); err != nil {
			return fmt.Errorf("%s: cmd: %w", r.ID, err)
		}
	}
	if r.Output != "" {
		if r.outRe, err = regexp.Compile(`(?m)` + r.Output); err != nil {
			return fmt.Errorf("%s: output: %w", r.ID, err)
		}
	}
	return nil
}

// MatchOutput は output 条件だけを s に当てる（テスト用）。
func (r *Rule) MatchOutput(s string) bool {
	return r.outRe != nil && r.outRe.MatchString(s)
}

type Set struct {
	Rules []*Rule
}

func parse(data []byte, source string) ([]*Rule, error) {
	var rs []*Rule
	if err := yaml.Unmarshal(data, &rs); err != nil {
		return nil, fmt.Errorf("%s: %w", source, err)
	}
	for _, r := range rs {
		r.Source = source
		if err := r.compile(); err != nil {
			return nil, fmt.Errorf("%s: %w", source, err)
		}
	}
	return rs, nil
}

// Load は組み込みルールとユーザールール（userDir/*.yaml）を読み込む。
// 同じ id のユーザールールは組み込みルールを上書きする。
// ユーザールールのエラーは warn に返し、読み込みは続ける。
func Load(builtin fs.FS, userDir string) (*Set, []error, error) {
	var warns []error
	byID := map[string]int{}
	set := &Set{}
	add := func(rs []*Rule) {
		for _, r := range rs {
			if i, ok := byID[r.ID]; ok {
				set.Rules[i] = r
				continue
			}
			byID[r.ID] = len(set.Rules)
			set.Rules = append(set.Rules, r)
		}
	}

	names, err := fs.Glob(builtin, "rules/*.yaml")
	if err != nil {
		return nil, nil, err
	}
	sort.Strings(names)
	for _, n := range names {
		b, err := fs.ReadFile(builtin, n)
		if err != nil {
			return nil, nil, err
		}
		rs, err := parse(b, "builtin:"+path.Base(n))
		if err != nil {
			return nil, nil, err
		}
		add(rs)
	}

	if userDir != "" {
		files, _ := filepath.Glob(filepath.Join(userDir, "*.yaml"))
		more, _ := filepath.Glob(filepath.Join(userDir, "*.yml"))
		files = append(files, more...)
		sort.Strings(files)
		for _, f := range files {
			b, err := os.ReadFile(f)
			if err != nil {
				warns = append(warns, err)
				continue
			}
			rs, err := parse(b, f)
			if err != nil {
				warns = append(warns, err)
				continue
			}
			add(rs)
		}
	}
	return set, warns, nil
}

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]|\x1b\][^\x07]*\x07`)

// Normalize は照合前に出力から ANSI エスケープを除き、改行を揃える。
func Normalize(s string) string {
	s = ansiRe.ReplaceAllString(s, "")
	return strings.ReplaceAll(s, "\r\n", "\n")
}

// Match は当てはまるルールをスコア順に最大 limit 件返す。
func (s *Set) Match(in Input, limit int) []Result {
	prog := Program(in.Cmd)
	var out []Result
	for _, r := range s.Rules {
		vars := map[string]string{"prog": prog, "cmd": in.Cmd}
		if in.Code >= 0 {
			vars["code"] = strconv.Itoa(in.Code)
		}
		score := r.Priority
		if r.progRe != nil {
			if prog == "" || !r.progRe.MatchString(prog) {
				continue
			}
			score++
		}
		if r.cmdRe != nil {
			m := r.cmdRe.FindStringSubmatch(in.Cmd)
			if in.Cmd == "" || m == nil {
				continue
			}
			capture(vars, r.cmdRe, m)
			score++
		}
		if len(r.Code) > 0 {
			if in.Code < 0 || !contains(r.Code, in.Code) {
				continue
			}
			score++
		}
		if r.outRe != nil {
			m := r.outRe.FindStringSubmatch(in.Output)
			if in.Output == "" || m == nil {
				continue
			}
			capture(vars, r.outRe, m)
			score += 2
		}
		out = append(out, Result{
			Rule:  r,
			Cause: expand(r.Cause, vars),
			Fix:   expandFix(r.Fix, vars),
			Score: score,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func capture(vars map[string]string, re *regexp.Regexp, m []string) {
	for i, name := range re.SubexpNames() {
		if name != "" && m[i] != "" {
			vars[name] = m[i]
		}
	}
}

func contains(xs []int, x int) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

var varRe = regexp.MustCompile(`\{\{\s*(\w+)\s*\}\}`)

func expand(s string, vars map[string]string) string {
	return varRe.ReplaceAllStringFunc(s, func(m string) string {
		return vars[varRe.FindStringSubmatch(m)[1]]
	})
}

// expandFix は値が空の変数を含む対処行を捨てる（意味のないコマンドを出さないため）。
func expandFix(fix []string, vars map[string]string) []string {
	var out []string
	for _, f := range fix {
		ok := true
		for _, m := range varRe.FindAllStringSubmatch(f, -1) {
			if vars[m[1]] == "" {
				ok = false
				break
			}
		}
		if ok {
			out = append(out, expand(f, vars))
		}
	}
	return out
}

var assignRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)

// Program はコマンドラインから実際に動くプログラム名を取り出す。
// 環境変数の代入や sudo / time などの前置きは読み飛ばす。
func Program(cmd string) string {
	skip := map[string]bool{
		"sudo": true, "doas": true, "command": true, "exec": true, "time": true,
		"nohup": true, "env": true, "nice": true, "builtin": true, "noglob": true,
	}
	for _, f := range strings.Fields(cmd) {
		if assignRe.MatchString(f) || skip[f] || strings.HasPrefix(f, "-") {
			continue
		}
		f = strings.Trim(f, `"'(`)
		return filepath.Base(f)
	}
	return ""
}

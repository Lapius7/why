// Package render は診断結果を端末向けに整形する。
package render

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Lapius7/why/internal/exitcode"
	"github.com/Lapius7/why/internal/rules"
)

type Report struct {
	Cmd       string
	Cwd       string
	Code      int // 不明なら -1
	When      time.Time
	HasOutput bool
	Results   []rules.Result
	Hints     []string
}

type Printer struct {
	W     io.Writer
	Color bool
}

func (p Printer) c(code, s string) string {
	if !p.Color {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func (p Printer) Print(r Report) {
	w := p.W
	if r.Cmd != "" {
		head := p.c("1;31", "✗ ") + p.c("1", r.Cmd)
		if r.Code >= 0 {
			head += p.c("2", fmt.Sprintf("  (終了コード %d)", r.Code))
		}
		fmt.Fprintln(w, head)
		var meta []string
		if r.Cwd != "" {
			meta = append(meta, shortHome(r.Cwd))
		}
		if !r.When.IsZero() {
			meta = append(meta, ago(time.Since(r.When)))
		}
		if len(meta) > 0 {
			fmt.Fprintln(w, "  "+p.c("2", strings.Join(meta, " · ")))
		}
	}
	if r.Code > 0 {
		fmt.Fprintf(w, "  %s %s\n", p.c("33", fmt.Sprintf("[%d]", r.Code)), exitcode.Describe(r.Code))
	}

	for i, res := range r.Results {
		fmt.Fprintln(w)
		label := "原因"
		if i > 0 {
			label = "候補"
		}
		fmt.Fprintf(w, "%s  %s\n", p.c("1;36", label), res.Cause)
		for j, f := range res.Fix {
			prefix := "      "
			if j == 0 {
				prefix = p.c("1;32", "対処") + "  "
			}
			fmt.Fprintf(w, "%s%s %s\n", prefix, p.c("32", "→"), f)
		}
	}

	if len(r.Results) == 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, p.c("2", "該当するルールはありませんでした。"))
	}
	if len(r.Hints) > 0 {
		fmt.Fprintln(w)
		for _, h := range r.Hints {
			fmt.Fprintln(w, p.c("2", "ヒント: "+h))
		}
	}
}

func shortHome(p string) string {
	home, err := os.UserHomeDir()
	if err == nil && home != "" && strings.HasPrefix(p, home) {
		return "~" + strings.TrimPrefix(p, home)
	}
	return p
}

func ago(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "たった今"
	case d < time.Hour:
		return fmt.Sprintf("%d分前", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d時間前", int(d.Hours()))
	}
	return fmt.Sprintf("%d日前", int(d.Hours()/24))
}

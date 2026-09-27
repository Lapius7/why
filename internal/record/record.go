// Package record はシェルフックが書き出した「直前に失敗したコマンド」の記録を読む。
//
// フォーマット（改行区切り）: 1行目=終了コード, 2行目=cwd, 3行目以降=コマンド
// ファイル名は last-<シェルのPID>。
package record

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Record struct {
	Code int
	Cwd  string
	Cmd  string
	Time time.Time
	Path string
}

// Dir は記録ディレクトリ。シェル側のフックと同じ規則で決める。
func Dir() string {
	if d := os.Getenv("XDG_STATE_HOME"); d != "" {
		return filepath.Join(d, "why")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", "why")
}

// Load は親シェル（ppid）の記録を返す。無ければ最も新しい記録を返す。
func Load(dir string, ppid int) (*Record, error) {
	if r, err := parseFile(filepath.Join(dir, "last-"+strconv.Itoa(ppid))); err == nil {
		return r, nil
	}
	files, _ := filepath.Glob(filepath.Join(dir, "last-*"))
	var newest *Record
	for _, f := range files {
		r, err := parseFile(f)
		if err != nil {
			continue
		}
		if newest == nil || r.Time.After(newest.Time) {
			newest = r
		}
	}
	if newest == nil {
		return nil, os.ErrNotExist
	}
	return newest, nil
}

func parseFile(path string) (*Record, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	r, err := Parse(string(b))
	if err != nil {
		return nil, err
	}
	r.Time = st.ModTime()
	r.Path = path
	return r, nil
}

func Parse(s string) (*Record, error) {
	parts := strings.SplitN(s, "\n", 3)
	if len(parts) < 3 {
		return nil, errors.New("記録の形式が不正")
	}
	code, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return nil, err
	}
	return &Record{
		Code: code,
		Cwd:  parts[1],
		Cmd:  strings.TrimRight(parts[2], "\n"),
	}, nil
}

// Prune は古い記録（終了済みシェルの残骸）を消す。
func Prune(dir string, maxAge time.Duration) {
	files, _ := filepath.Glob(filepath.Join(dir, "last-*"))
	for _, f := range files {
		if st, err := os.Stat(f); err == nil && time.Since(st.ModTime()) > maxAge {
			os.Remove(f)
		}
	}
}

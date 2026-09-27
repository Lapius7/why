// Package runner はコマンドを実行し、画面に流しつつ出力を取り込む。
package runner

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"syscall"
)

const maxOutput = 1 << 20

type Result struct {
	Code   int
	Output string
}

// tail は末尾 maxOutput バイトだけを保持する（エラーは末尾に出ることが多い）。
type tail struct {
	mu  sync.Mutex
	buf []byte
}

func (t *tail) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf = append(t.buf, p...)
	if len(t.buf) > 2*maxOutput {
		t.buf = append([]byte(nil), t.buf[len(t.buf)-maxOutput:]...)
	}
	return len(p), nil
}

func (t *tail) String() string {
	if len(t.buf) > maxOutput {
		return string(t.buf[len(t.buf)-maxOutput:])
	}
	return string(t.buf)
}

// Exec は argv を直接実行する。
func Exec(argv []string, dir string) Result {
	return run(exec.Command(argv[0], argv[1:]...), argv[0], dir)
}

// Shell は cmdline をユーザーのシェル（$SHELL -c）で実行する。
func Shell(cmdline, dir string) Result {
	sh := os.Getenv("SHELL")
	if sh == "" {
		sh = "/bin/sh"
	}
	return run(exec.Command(sh, "-c", cmdline), sh, dir)
}

func run(c *exec.Cmd, name, dir string) Result {
	var t tail
	c.Dir = dir
	c.Env = messagesInEnglish(os.Environ())
	c.Stdin = os.Stdin
	c.Stdout = io.MultiWriter(os.Stdout, &t)
	c.Stderr = io.MultiWriter(os.Stderr, &t)
	err := c.Run()
	if err == nil {
		return Result{Code: 0, Output: t.String()}
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			return Result{Code: 128 + int(ws.Signal()), Output: t.String()}
		}
		return Result{Code: ee.ExitCode(), Output: t.String()}
	}
	// 起動自体に失敗した場合はシェルと同じ終了コード・メッセージに寄せる
	if errors.Is(err, exec.ErrNotFound) {
		msg := fmt.Sprintf("%s: command not found\n", name)
		fmt.Fprint(os.Stderr, msg)
		return Result{Code: 127, Output: msg}
	}
	msg := fmt.Sprintf("%s: %v\n", name, err)
	fmt.Fprint(os.Stderr, msg)
	return Result{Code: 126, Output: msg}
}

// messagesInEnglish はルールと照合しやすいようにメッセージだけ英語にする。
// 文字コードなど他のロケール設定は変えない。
func messagesInEnglish(env []string) []string {
	out := make([]string, 0, len(env)+2)
	for _, e := range env {
		switch {
		case strings.HasPrefix(e, "LANGUAGE="), strings.HasPrefix(e, "LC_MESSAGES="):
			continue
		case strings.HasPrefix(e, "LC_ALL="):
			// LC_ALL は LC_MESSAGES より優先されるので LANG に移す
			if v := strings.TrimPrefix(e, "LC_ALL="); v != "" {
				out = append(out, "LANG="+v)
			}
			continue
		}
		out = append(out, e)
	}
	return append(out, "LC_MESSAGES=C.UTF-8")
}

var dangerous = []struct {
	re     *regexp.Regexp
	reason string
}{
	{regexp.MustCompile(`(^|[;&|(\s])(rm|rmdir|mv|dd|shred|truncate|mkfs\S*|kill|pkill|killall|reboot|shutdown|poweroff)(\s|$)`), "ファイル削除・移動・プロセス停止などの副作用がある"},
	{regexp.MustCompile(`(^|[;&|(\s])sudo(\s|$)`), "sudo で実行される"},
	{regexp.MustCompile(`\bgit\s+(push|commit|reset|clean|rebase|merge|pull|checkout|switch|restore|stash|tag|cherry-pick|revert|am)\b`), "git の履歴・作業ツリーを変更する"},
	{regexp.MustCompile(`\bdocker(\s+compose|-compose)?\s+(rm|rmi|run|up|down|stop|kill|prune|system|volume|network)\b`), "コンテナやイメージの状態を変更する"},
	{regexp.MustCompile(`\b(npm|pnpm|yarn)\s+(publish|unpublish|deprecate)\b`), "パッケージを公開・変更する"},
	{regexp.MustCompile(`\b(apt|apt-get|dpkg|snap|brew)\s+(install|remove|purge|upgrade|autoremove|uninstall)\b`), "パッケージをインストール・削除する"},
	{regexp.MustCompile(`\bcurl\b.*(-X\s*(POST|PUT|DELETE|PATCH)|--data|-d\s)|\bhttps?\s+(POST|PUT|DELETE|PATCH)\b`), "外部に書き込みリクエストを送る"},
	{regexp.MustCompile(`(^|[^0-9&>])>[^&>]|>>`), "リダイレクトでファイルを上書き・追記する"},
}

// Dangerous は再実行すると副作用がありそうなら理由を返す。
func Dangerous(cmdline string) (string, bool) {
	for _, d := range dangerous {
		if d.re.MatchString(cmdline) {
			return d.reason, true
		}
	}
	return "", false
}

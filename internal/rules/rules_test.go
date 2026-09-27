package rules_test

import (
	"strings"
	"testing"

	"github.com/Lapius7/why"
	"github.com/Lapius7/why/internal/rules"
)

func load(t *testing.T) *rules.Set {
	t.Helper()
	set, _, err := rules.Load(why.Rules, "")
	if err != nil {
		t.Fatal(err)
	}
	return set
}

// output 条件を持つルールは example を必須にし、自分の example に一致することを確かめる。
func TestBuiltinExamples(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range load(t).Rules {
		if seen[r.ID] {
			t.Errorf("id が重複: %s", r.ID)
		}
		seen[r.ID] = true
		if r.Output == "" {
			continue
		}
		if r.Example == "" {
			t.Errorf("%s: output ルールに example がない", r.ID)
			continue
		}
		if !r.MatchOutput(r.Example) {
			t.Errorf("%s: example に一致しない: %q", r.ID, r.Example)
		}
	}
}

func TestMatch(t *testing.T) {
	set := load(t)
	cases := []struct {
		name  string
		in    rules.Input
		top   string
		cause string
	}{
		{"not found", rules.Input{Cmd: "gti status", Code: 127}, "command-not-found", "コマンド「gti」が見つからない"},
		{"python", rules.Input{Cmd: "python app.py", Code: 127}, "python-not-found", ""},
		{"sudo prefix", rules.Input{Cmd: "sudo FOO=1 apt install x", Code: 100, Output: "E: Unable to locate package x"}, "apt-unable-to-locate", "パッケージ「x」が見つからない"},
		{"push", rules.Input{Cmd: "git push", Code: 1, Output: " ! [rejected]        main -> main (non-fast-forward)\nerror: failed to push some refs"}, "git-push-rejected", ""},
		{"docker port", rules.Input{Cmd: "docker run -p 8080:80 nginx", Code: 125, Output: "Error response from daemon: driver failed programming external connectivity: Bind for 0.0.0.0:8080 failed: port is already allocated"}, "docker-port-allocated", ""},
		{"docker socket", rules.Input{Cmd: "docker ps", Code: 1, Output: "permission denied while trying to connect to the Docker daemon socket at unix:///var/run/docker.sock: Get ...: dial unix /var/run/docker.sock: connect: permission denied"}, "docker-socket-permission", ""},
		{"crlf", rules.Input{Cmd: "./run.sh", Code: 126, Output: "bash: ./run.sh: /bin/bash\r: bad interpreter: No such file or directory"}, "crlf-shebang", ""},
		{"inotify", rules.Input{Cmd: "npm run dev", Code: 1, Output: "Error: ENOSPC: System limit for number of file watchers reached, watch '/app'"}, "inotify-limit", ""},
		{"pipe only", rules.Input{Code: -1, Output: "ModuleNotFoundError: No module named 'cv2.foo'"}, "py-module-not-found", "Python モジュール「cv2」が入っていない（または別の Python 環境で動いている）"},
		{"oom", rules.Input{Cmd: "make -j16", Code: 137}, "oom-killed", ""},
		{"docker oom", rules.Input{Cmd: "docker compose up", Code: 137}, "docker-oom", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := set.Match(c.in, 3)
			if len(res) == 0 {
				t.Fatalf("一致なし")
			}
			if res[0].Rule.ID != c.top {
				ids := []string{}
				for _, r := range res {
					ids = append(ids, r.Rule.ID)
				}
				t.Fatalf("先頭 = %v, want %s", ids, c.top)
			}
			if c.cause != "" && res[0].Cause != c.cause {
				t.Errorf("cause = %q, want %q", res[0].Cause, c.cause)
			}
		})
	}
}

func TestFixDropsEmptyVars(t *testing.T) {
	res := load(t).Match(rules.Input{Code: -1, Output: "listen: address already in use"}, 1)
	if len(res) == 0 || res[0].Rule.ID != "port-in-use" {
		t.Fatalf("port-in-use に一致しない: %+v", res)
	}
	for _, f := range res[0].Fix {
		if f == "" || strings.Contains(f, "{{") || strings.Contains(f, "sport = :'") {
			t.Errorf("空の変数を含む行が残っている: %q", f)
		}
	}
}

func TestProgram(t *testing.T) {
	cases := map[string]string{
		"git push":                       "git",
		"sudo apt update":                "apt",
		"FOO=1 BAR=2 /usr/bin/python3 x": "python3",
		"time make -j4":                  "make",
		"":                               "",
	}
	for in, want := range cases {
		if got := rules.Program(in); got != want {
			t.Errorf("Program(%q) = %q, want %q", in, got, want)
		}
	}
}

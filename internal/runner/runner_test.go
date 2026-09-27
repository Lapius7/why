package runner

import "testing"

func TestDangerous(t *testing.T) {
	bad := []string{"rm -rf build", "git push -f", "sudo make install", "echo x > a.txt", "docker compose down", "cd x && rm a", "apt install foo"}
	ok := []string{"npm run build", "make test", "go build ./... 2>&1", "git status", "ls 2>/dev/null", "grep -r armor ."}
	for _, c := range bad {
		if _, d := Dangerous(c); !d {
			t.Errorf("危険と判定されない: %q", c)
		}
	}
	for _, c := range ok {
		if r, d := Dangerous(c); d {
			t.Errorf("誤って危険と判定: %q (%s)", c, r)
		}
	}
}

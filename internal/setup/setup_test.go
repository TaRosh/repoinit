package setup

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TaRosh/repoinit/internal/keys"
)

const repoURL = "https://github.com/TaRosh/parser"

// gitOut — значение из git в папке dir (для проверок).
func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := git(context.Background(), dir, args...)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(out)
}

func TestRunFreshDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new project") // папки ещё нет, и в пути пробел

	res, err := Run(context.Background(), repoURL, dir)
	if err != nil {
		t.Fatal(err)
	}

	checks := []struct {
		name string
		ok   bool
	}{
		{"git repo created", exists(filepath.Join(dir, ".git"))},
		{"private key created", exists(filepath.Join(dir, keys.PrivateFile))},
		{"public key returned", strings.HasPrefix(res.PublicKey, "ssh-ed25519 ") && strings.HasSuffix(res.PublicKey, "deploy:TaRosh/parser")},
		{"branch is main", gitOut(t, dir, "symbolic-ref", "--short", "HEAD") == "main"},
		{"remote is ssh", gitOut(t, dir, "remote", "get-url", "origin") == "git@github.com:TaRosh/parser.git"},
		{"ssh command uses key", gitOut(t, dir, "config", "core.sshCommand") == SSHCommand(dir)},
		{"5 steps reported", len(res.Steps) == 5},
	}
	for _, c := range checks {
		if !c.ok {
			t.Errorf("%s: failed (result %+v)", c.name, res)
		}
	}

	// Ключи защищены: git их не видит, даже если сделать "git add -A".
	if out := gitOut(t, dir, "status", "--porcelain"); strings.Contains(out, "deploy_key") {
		t.Errorf("keys are visible to git:\n%s", out)
	}
}

func TestRunIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	first, err := Run(ctx, repoURL, dir)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Run(ctx, "git@github.com:TaRosh/parser.git", dir)
	if err != nil {
		t.Fatal(err)
	}

	if first.PublicKey != second.PublicKey {
		t.Error("key was regenerated on second run")
	}
	ignore, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if n := strings.Count(string(ignore), "deploy_key.pub"); n != 1 {
		t.Errorf("deploy_key.pub appears %d times in .gitignore", n)
	}
	for _, want := range []string{"уже есть", "используются существующие", "уже защищает"} {
		if !strings.Contains(strings.Join(second.Steps, "\n"), want) {
			t.Errorf("second run steps do not mention %q: %v", want, second.Steps)
		}
	}
}

func TestRunReplacesHTTPSRemote(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	gitOut(t, dir, "init", "-b", "main")
	gitOut(t, dir, "remote", "add", "origin", "https://github.com/TaRosh/old.git")

	if _, err := Run(ctx, repoURL, dir); err != nil {
		t.Fatal(err)
	}
	if got := gitOut(t, dir, "remote", "get-url", "origin"); got != "git@github.com:TaRosh/parser.git" {
		t.Errorf("origin = %q", got)
	}
}

func TestRunInvalidInput(t *testing.T) {
	tests := []struct {
		name, url, dir string
	}{
		{"not github", "https://gitlab.com/a/b", "x"},
		{"no repo name", "https://github.com/TaRosh", "x"},
		{"empty dir", repoURL, "  "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := t.TempDir()
			dir := tt.dir
			if strings.TrimSpace(dir) != "" {
				dir = filepath.Join(base, dir)
			}
			if _, err := Run(context.Background(), tt.url, dir); err == nil {
				t.Fatal("expected error")
			}
			// При ошибке ввода ничего не создаётся.
			if entries, _ := os.ReadDir(base); len(entries) != 0 {
				t.Errorf("files created on invalid input: %v", entries)
			}
		})
	}
}

func TestCheckAccess(t *testing.T) {
	ctx := context.Background()

	// "Сервер" — локальный bare-репозиторий: так проверяем ls-remote без сети.
	server := t.TempDir()
	gitOut(t, server, "init", "--bare")

	ok := t.TempDir()
	gitOut(t, ok, "init")
	gitOut(t, ok, "remote", "add", "origin", server)

	noRemote := t.TempDir()
	gitOut(t, noRemote, "init")

	tests := []struct {
		name    string
		dir     string
		wantErr bool
	}{
		{"remote reachable", ok, false},
		{"no origin", noRemote, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckAccess(ctx, tt.dir)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), "git ls-remote") {
				t.Errorf("error should name the git command: %v", err)
			}
		})
	}
}

func TestSSHCommand(t *testing.T) {
	got := SSHCommand(filepath.Join("/home", "my projects", "bot"))
	want := `ssh -i "/home/my projects/bot/deploy_key" -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestGitNotFound(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // git в PATH нет
	if _, err := git(context.Background(), t.TempDir(), "status"); err == nil || !strings.Contains(err.Error(), "git не найден") {
		t.Fatalf("err = %v", err)
	}
	if _, err := exec.LookPath("git"); err == nil {
		t.Fatal("PATH override did not work")
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

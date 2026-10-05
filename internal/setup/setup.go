// Package setup настраивает папку проекта на работу с репозиторием GitHub
// через deploy-ключ: git init, ключи, .gitignore, remote и core.sshCommand.
package setup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/TaRosh/repoinit/internal/ghurl"
	"github.com/TaRosh/repoinit/internal/gitignore"
	"github.com/TaRosh/repoinit/internal/keys"
)

// Result — итог настройки для показа пользователю.
type Result struct {
	Repo      ghurl.Repo
	Dir       string
	PublicKey string
	Steps     []string // что было сделано, по порядку
}

// Run выполняет настройку. Повторный запуск безопасен: уже сделанные шаги
// не ломаются (ключ не перегенерируется, строки в .gitignore не дублируются).
func Run(ctx context.Context, rawURL, dir string) (Result, error) {
	repo, err := ghurl.Parse(rawURL)
	if err != nil {
		return Result{}, err
	}
	if strings.TrimSpace(dir) == "" {
		return Result{}, errors.New("не выбрана папка проекта")
	}
	dir, err = filepath.Abs(dir)
	if err != nil {
		return Result{}, err
	}
	res := Result{Repo: repo, Dir: dir}
	step := func(format string, a ...any) { res.Steps = append(res.Steps, fmt.Sprintf(format, a...)) }

	// 1. Папка и git-репозиторий.
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return res, fmt.Errorf("создать папку: %w", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		step("git-репозиторий уже есть")
	} else {
		if _, err := git(ctx, dir, "init", "-b", "main"); err != nil {
			return res, err
		}
		step("git init (ветка main)")
	}

	// 2. Ключи — до .gitignore нельзя коммитить, поэтому сразу и туда, и туда.
	pub, created, err := keys.Ensure(dir, "deploy:"+repo.FullName())
	if err != nil {
		return res, err
	}
	res.PublicKey = pub
	if created {
		step("созданы ключи %s и %s", keys.PrivateFile, keys.PublicFile)
	} else {
		step("ключи уже есть — используются существующие")
	}

	added, err := gitignore.Ensure(filepath.Join(dir, ".gitignore"),
		"Ключи доступа к репозиторию — не коммитить!", keys.PrivateFile, keys.PublicFile)
	if err != nil {
		return res, err
	}
	if len(added) > 0 {
		step("в .gitignore добавлено: %s", strings.Join(added, ", "))
	} else {
		step(".gitignore уже защищает ключи")
	}

	// 3. Remote origin на SSH-адрес.
	remotes, err := git(ctx, dir, "remote")
	if err != nil {
		return res, err
	}
	if hasLine(remotes, "origin") {
		_, err = git(ctx, dir, "remote", "set-url", "origin", repo.SSHURL())
	} else {
		_, err = git(ctx, dir, "remote", "add", "origin", repo.SSHURL())
	}
	if err != nil {
		return res, err
	}
	step("remote origin → %s", repo.SSHURL())

	// 4. Только для ЭТОГО репозитория git ходит в GitHub с этим ключом.
	if _, err := git(ctx, dir, "config", "core.sshCommand", SSHCommand(dir)); err != nil {
		return res, err
	}
	step("git config core.sshCommand → ключ из папки проекта")

	return res, nil
}

// SSHCommand — команда ssh для git config core.sshCommand.
//   - путь с прямыми слешами и в кавычках: git выполняет её через sh,
//     а обратные слеши и пробелы в пути Windows sh бы испортил;
//   - IdentitiesOnly=yes — не пробовать другие ключи из ~/.ssh: GitHub
//     пустит по первому подходящему и может открыть не тот репозиторий;
//   - StrictHostKeyChecking=accept-new — при первом подключении запомнить
//     ключ сервера github.com без вопроса в консоли (которой у нас нет).
func SSHCommand(dir string) string {
	key := filepath.ToSlash(filepath.Join(dir, keys.PrivateFile))
	return fmt.Sprintf(`ssh -i "%s" -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new`, key)
}

// CheckAccess проверяет, пускает ли GitHub с этим ключом (git ls-remote).
// Имеет смысл после того, как ключ добавлен в Deploy keys.
func CheckAccess(ctx context.Context, dir string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	_, err := git(ctx, dir, "ls-remote", "origin")
	return err
}

// git запускает git в папке dir и возвращает stdout.
// При ошибке в тексте будет stderr git — там самое понятное объяснение.
func git(ctx context.Context, dir string, args ...string) (string, error) {
	// Без shell: аргументы передаются git как есть, подстановка команд невозможна.
	// Значения из ввода пользователя проверены ghurl.Parse и не начинаются с "-",
	// поэтому git не примет их за свои флаги.
	cmd := exec.CommandContext(ctx, "git", args...) //nolint:gosec // см. комментарий выше
	cmd.Dir = dir
	// Запретить git спрашивать логин/пароль: окна для ввода у нас нет,
	// процесс бы просто завис.
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	hideWindow(cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", errors.New("git не найден — установите Git for Windows: https://git-scm.com")
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", args[0], msg)
	}
	return stdout.String(), nil
}

func hasLine(s, want string) bool {
	for line := range strings.Lines(s) {
		if strings.TrimSpace(line) == want {
			return true
		}
	}
	return false
}

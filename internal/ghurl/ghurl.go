// Package ghurl разбирает ссылки на репозитории GitHub в любом привычном виде.
package ghurl

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Repo — репозиторий GitHub: владелец и имя.
type Repo struct {
	Owner string
	Name  string
}

var (
	// Логин GitHub: буквы, цифры и дефис, не начинается с дефиса, до 39 символов.
	ownerRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,38}$`)
	// Имя репозитория: буквы, цифры, '.', '_' и '-'.
	nameRe = regexp.MustCompile(`^[A-Za-z0-9._-]{1,100}$`)

	ErrNotGitHub = errors.New("это не ссылка на GitHub")
	ErrBadPath   = errors.New("в ссылке нет владельца и имени репозитория")
)

// Parse принимает ссылку в любом из видов:
//
//	https://github.com/owner/repo(.git)  — из адресной строки или кнопки Code
//	https://github.com/owner/repo/tree/main — ссылка на ветку или файл
//	github.com/owner/repo                 — без схемы
//	git@github.com:owner/repo.git         — SSH из кнопки Code
//	ssh://git@github.com/owner/repo.git
func Parse(raw string) (Repo, error) {
	s := strings.TrimSpace(raw)

	var path string
	switch {
	case strings.HasPrefix(s, "git@github.com:"):
		path = strings.TrimPrefix(s, "git@github.com:")
	default:
		// Убираем схему и пользователя: https://, http://, ssh://git@
		for _, p := range []string{"https://", "http://", "ssh://"} {
			s = strings.TrimPrefix(s, p)
		}
		s = strings.TrimPrefix(s, "git@")
		s = strings.TrimPrefix(s, "www.")
		host, rest, _ := strings.Cut(s, "/")
		if !strings.EqualFold(host, "github.com") {
			return Repo{}, ErrNotGitHub
		}
		path = rest
	}

	// Отбрасываем "?tab=..." и "#readme" из ссылок, скопированных из браузера.
	path, _, _ = strings.Cut(path, "?")
	path, _, _ = strings.Cut(path, "#")

	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		return Repo{}, ErrBadPath
	}
	r := Repo{Owner: parts[0], Name: strings.TrimSuffix(parts[1], ".git")}

	if !ownerRe.MatchString(r.Owner) {
		return Repo{}, fmt.Errorf("недопустимое имя владельца %q", r.Owner)
	}
	if !nameRe.MatchString(r.Name) || r.Name == "." || r.Name == ".." {
		return Repo{}, fmt.Errorf("недопустимое имя репозитория %q", r.Name)
	}
	return r, nil
}

// FullName — "owner/repo".
func (r Repo) FullName() string { return r.Owner + "/" + r.Name }

// SSHURL — адрес для git remote при доступе по deploy-ключу.
func (r Repo) SSHURL() string { return "git@github.com:" + r.FullName() + ".git" }

// DeployKeysURL — страница добавления deploy-ключа на GitHub.
func (r Repo) DeployKeysURL() string {
	return "https://github.com/" + r.FullName() + "/settings/keys/new"
}

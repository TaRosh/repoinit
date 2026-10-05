// Package keys создаёт SSH-ключи для deploy-доступа к репозиторию.
package keys

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

const (
	PrivateFile = "deploy_key"
	PublicFile  = "deploy_key.pub"
)

// Generate создаёт пару ключей Ed25519: приватный в формате OpenSSH
// (как у ssh-keygen) и публичный одной строкой "ssh-ed25519 AAAA... comment".
//
// Ed25519 — современный алгоритм: ключ короткий, быстрый и рекомендуется
// GitHub вместо RSA.
func Generate(comment string) (private, public []byte, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate key: %w", err)
	}

	block, err := ssh.MarshalPrivateKey(priv, comment)
	if err != nil {
		return nil, nil, fmt.Errorf("encode private key: %w", err)
	}

	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return nil, nil, fmt.Errorf("encode public key: %w", err)
	}
	line := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPub)))
	if comment != "" {
		line += " " + comment
	}
	return pem.EncodeToMemory(block), []byte(line + "\n"), nil
}

// Ensure возвращает публичный ключ из папки dir, создавая пару, если её нет.
// Существующий ключ НЕ перезаписывается: он мог быть уже добавлен на GitHub.
// created == true, если ключ создан сейчас.
func Ensure(dir, comment string) (public string, created bool, err error) {
	privPath := filepath.Join(dir, PrivateFile)
	pubPath := filepath.Join(dir, PublicFile)

	privExists, err := exists(privPath)
	if err != nil {
		return "", false, err
	}
	if privExists {
		pub, err := os.ReadFile(pubPath) //nolint:gosec // путь собран из выбранной пользователем папки
		if err != nil {
			return "", false, fmt.Errorf("приватный ключ есть, а %s не прочитать: %w", PublicFile, err)
		}
		return strings.TrimSpace(string(pub)), false, nil
	}

	priv, pub, err := Generate(comment)
	if err != nil {
		return "", false, err
	}
	// 0600 — приватный ключ может читать только владелец (ssh на Linux
	// откажется работать с ключом, доступным другим пользователям).
	if err := os.WriteFile(privPath, priv, 0o600); err != nil {
		return "", false, fmt.Errorf("save private key: %w", err)
	}
	if err := os.WriteFile(pubPath, pub, 0o644); err != nil { //nolint:gosec // публичный ключ не секрет
		return "", false, fmt.Errorf("save public key: %w", err)
	}
	return strings.TrimSpace(string(pub)), true, nil
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

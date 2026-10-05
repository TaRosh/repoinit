package keys

import (
	"bytes"
	"crypto/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestGenerateKeysArePair(t *testing.T) {
	priv, pub, err := Generate("deploy:TaRosh/parser")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(string(pub), "ssh-ed25519 ") || !strings.HasSuffix(string(pub), " deploy:TaRosh/parser\n") {
		t.Errorf("public key format: %q", pub)
	}
	if !bytes.HasPrefix(priv, []byte("-----BEGIN OPENSSH PRIVATE KEY-----")) {
		t.Errorf("private key is not in OpenSSH format")
	}

	// Главная проверка: ключи действительно пара. Подписываем приватным,
	// проверяем публичным — так же делает сервер GitHub при подключении.
	signer, err := ssh.ParsePrivateKey(priv)
	if err != nil {
		t.Fatalf("parse private key: %v", err)
	}
	pubKey, _, _, _, err := ssh.ParseAuthorizedKey(pub)
	if err != nil {
		t.Fatalf("parse public key: %v", err)
	}
	data := []byte("challenge")
	sig, err := signer.Sign(rand.Reader, data)
	if err != nil {
		t.Fatal(err)
	}
	if err := pubKey.Verify(data, sig); err != nil {
		t.Fatalf("public key does not match private key: %v", err)
	}
}

func TestGenerateUniqueKeys(t *testing.T) {
	_, pub1, _ := Generate("")
	_, pub2, _ := Generate("")
	if bytes.Equal(pub1, pub2) {
		t.Fatal("two generated keys are equal")
	}
}

func TestEnsure(t *testing.T) {
	dir := t.TempDir()

	pub1, created, err := Ensure(dir, "c")
	if err != nil || !created {
		t.Fatalf("first Ensure: created=%v err=%v", created, err)
	}

	info, err := os.Stat(filepath.Join(dir, PrivateFile))
	if err != nil {
		t.Fatal(err)
	}
	// На Windows права доступа устроены иначе (ACL), проверка имеет смысл в Linux/CI.
	if perm := info.Mode().Perm(); perm&0o077 != 0 && os.PathSeparator == '/' {
		t.Errorf("private key permissions = %o, want 600", perm)
	}

	// Повторный вызов отдаёт тот же ключ и ничего не перезаписывает.
	pub2, created, err := Ensure(dir, "c")
	if err != nil || created {
		t.Fatalf("second Ensure: created=%v err=%v", created, err)
	}
	if pub1 != pub2 {
		t.Error("existing key was replaced")
	}
}

func TestEnsureBrokenPair(t *testing.T) {
	dir := t.TempDir()
	// Приватный ключ есть, публичного нет — молча генерировать новый нельзя.
	if err := os.WriteFile(filepath.Join(dir, PrivateFile), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Ensure(dir, "c"); err == nil {
		t.Fatal("expected error when public key is missing")
	}
}

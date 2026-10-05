// Package gitignore добавляет строки в .gitignore без дублей.
package gitignore

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// Ensure дописывает в файл path недостающие строки entries под заголовком header.
// Существующее содержимое и перевод строк (LF или CRLF) сохраняются.
// Возвращает добавленные строки (пусто, если всё уже было).
func Ensure(path, header string, entries ...string) (added []string, err error) {
	data, err := os.ReadFile(path) //nolint:gosec // путь задаёт вызывающий код
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	// Множество (map с пустыми значениями) существующих строк: проверка
	// "уже есть?" — O(1) на строку вместо поиска по всему файлу.
	present := make(map[string]struct{})
	for line := range strings.Lines(string(data)) {
		present[strings.TrimSpace(line)] = struct{}{}
	}
	for _, e := range entries {
		if _, ok := present[e]; !ok {
			added = append(added, e)
			present[e] = struct{}{} // защита от дублей внутри самих entries
		}
	}
	if len(added) == 0 {
		return nil, nil
	}

	nl := "\n"
	if bytes.Contains(data, []byte("\r\n")) {
		nl = "\r\n"
	}

	var b strings.Builder
	b.Write(data)
	if len(data) > 0 {
		if !bytes.HasSuffix(data, []byte("\n")) {
			b.WriteString(nl)
		}
		b.WriteString(nl) // пустая строка-разделитель перед нашим блоком
	}
	if header != "" {
		b.WriteString("# " + header + nl)
	}
	for _, e := range added {
		b.WriteString(e + nl)
	}

	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil { //nolint:gosec // .gitignore не секрет
		return nil, fmt.Errorf("write %s: %w", path, err)
	}
	return added, nil
}

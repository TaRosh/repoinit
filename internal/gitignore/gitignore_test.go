package gitignore

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestEnsure(t *testing.T) {
	keys := []string{"deploy_key", "deploy_key.pub"}

	tests := []struct {
		name      string
		existing  *string // nil — файла нет
		entries   []string
		want      string
		wantAdded []string
	}{
		{
			name:      "no file",
			existing:  nil,
			entries:   keys,
			want:      "# keys\ndeploy_key\ndeploy_key.pub\n",
			wantAdded: keys,
		},
		{
			name:      "append to existing",
			existing:  ptr("/bin/\n"),
			entries:   keys,
			want:      "/bin/\n\n# keys\ndeploy_key\ndeploy_key.pub\n",
			wantAdded: keys,
		},
		{
			name:      "existing without trailing newline",
			existing:  ptr("/bin/"),
			entries:   keys,
			want:      "/bin/\n\n# keys\ndeploy_key\ndeploy_key.pub\n",
			wantAdded: keys,
		},
		{
			name:      "already present — file untouched",
			existing:  ptr("deploy_key\ndeploy_key.pub\n"),
			entries:   keys,
			want:      "deploy_key\ndeploy_key.pub\n",
			wantAdded: nil,
		},
		{
			name:      "partially present, with spaces and CRLF",
			existing:  ptr("  deploy_key  \r\n.env\r\n"),
			entries:   keys,
			want:      "  deploy_key  \r\n.env\r\n\r\n# keys\r\ndeploy_key.pub\r\n",
			wantAdded: []string{"deploy_key.pub"},
		},
		{
			name:      "duplicates in entries",
			existing:  nil,
			entries:   []string{"a", "a"},
			want:      "# keys\na\n",
			wantAdded: []string{"a"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), ".gitignore")
			if tt.existing != nil {
				if err := os.WriteFile(path, []byte(*tt.existing), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			added, err := Ensure(path, "keys", tt.entries...)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(added, tt.wantAdded) {
				t.Errorf("added = %q, want %q", added, tt.wantAdded)
			}

			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("file:\n%q\nwant:\n%q", got, tt.want)
			}
		})
	}
}

func ptr(s string) *string { return &s }

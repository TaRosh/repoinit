package ghurl

import (
	"errors"
	"testing"
)

func TestParse(t *testing.T) {
	want := Repo{Owner: "TaRosh", Name: "tgBotRecord"}

	tests := []struct {
		in      string
		want    Repo
		wantErr bool
		wantIs  error // если задано — конкретная ошибка
	}{
		{"https://github.com/TaRosh/tgBotRecord", want, false, nil},
		{"https://github.com/TaRosh/tgBotRecord.git", want, false, nil},
		{"https://github.com/TaRosh/tgBotRecord/", want, false, nil},
		{"  https://github.com/TaRosh/tgBotRecord  ", want, false, nil},
		{"https://github.com/TaRosh/tgBotRecord/tree/main/cmd", want, false, nil},
		{"https://github.com/TaRosh/tgBotRecord?tab=readme-ov-file", want, false, nil},
		{"https://github.com/TaRosh/tgBotRecord#readme", want, false, nil},
		{"https://www.github.com/TaRosh/tgBotRecord", want, false, nil},
		{"http://GitHub.com/TaRosh/tgBotRecord", want, false, nil},
		{"github.com/TaRosh/tgBotRecord", want, false, nil},
		{"git@github.com:TaRosh/tgBotRecord.git", want, false, nil},
		{"ssh://git@github.com/TaRosh/tgBotRecord.git", want, false, nil},
		{"https://github.com/my-org/repo.name_v2", Repo{Owner: "my-org", Name: "repo.name_v2"}, false, nil},

		{"", Repo{}, true, ErrNotGitHub},
		{"https://gitlab.com/TaRosh/tgBotRecord", Repo{}, true, ErrNotGitHub},
		{"https://github.com.evil.ru/TaRosh/repo", Repo{}, true, ErrNotGitHub},
		{"https://github.com/TaRosh", Repo{}, true, ErrBadPath},
		{"https://github.com/", Repo{}, true, ErrBadPath},
		{"https://github.com/-bad/repo", Repo{}, true, nil},
		{"https://github.com/TaRosh/re po", Repo{}, true, nil},
		{"https://github.com/TaRosh/..", Repo{}, true, nil},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := Parse(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
				t.Fatalf("err = %v, want %v", err, tt.wantIs)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestRepoURLs(t *testing.T) {
	r := Repo{Owner: "TaRosh", Name: "parser"}
	tests := []struct{ got, want string }{
		{r.FullName(), "TaRosh/parser"},
		{r.SSHURL(), "git@github.com:TaRosh/parser.git"},
		{r.DeployKeysURL(), "https://github.com/TaRosh/parser/settings/keys/new"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("got %q, want %q", tt.got, tt.want)
		}
	}
}

package paths

import (
	"path/filepath"
	"testing"
)

func TestNewLayout(t *testing.T) {
	home := filepath.Join("tmp", "codex-switch")
	layout := NewLayout(home)

	if layout.Home != home {
		t.Fatalf("home is %q, want %q", layout.Home, home)
	}

	if layout.AccountsDir != filepath.Join(home, "accounts") {
		t.Fatalf("accounts dir is %q, want %q", layout.AccountsDir, filepath.Join(home, "accounts"))
	}

	if layout.SharedDir != filepath.Join(home, "shared") {
		t.Fatalf("shared dir is %q, want %q", layout.SharedDir, filepath.Join(home, "shared"))
	}

	if layout.RuntimeDir != filepath.Join(home, "runtime") {
		t.Fatalf("runtime dir is %q, want %q", layout.RuntimeDir, filepath.Join(home, "runtime"))
	}

	if layout.CurrentHomeDir != filepath.Join(home, "runtime", "current-home") {
		t.Fatalf("current home dir is %q, want %q", layout.CurrentHomeDir, filepath.Join(home, "runtime", "current-home"))
	}

	if layout.StateDir != filepath.Join(home, "state") {
		t.Fatalf("state dir is %q, want %q", layout.StateDir, filepath.Join(home, "state"))
	}

	if layout.BinDir != filepath.Join(home, "bin") {
		t.Fatalf("bin dir is %q, want %q", layout.BinDir, filepath.Join(home, "bin"))
	}

	if layout.RegistryPath != filepath.Join(home, "state", "accounts.json") {
		t.Fatalf("registry path is %q, want %q", layout.RegistryPath, filepath.Join(home, "state", "accounts.json"))
	}

	if layout.WrapperPath != filepath.Join(home, "bin", "codex") {
		t.Fatalf("wrapper path is %q, want %q", layout.WrapperPath, filepath.Join(home, "bin", "codex"))
	}
}

func TestLayoutAccountPaths(t *testing.T) {
	home := filepath.Join("tmp", "codex-switch")
	layout := NewLayout(home)

	tag := "test"
	if layout.AccountDir(tag) != filepath.Join(layout.AccountsDir, tag) {
		t.Fatalf("account dir for tag %q is %q, want %q", tag, layout.AccountDir(tag), filepath.Join(layout.AccountsDir, tag))
	}

	if layout.AccountAuthPath(tag) != filepath.Join(layout.AccountDir(tag), "auth.json") {
		t.Fatalf("account auth path for tag %q is %q, want %q", tag, layout.AccountAuthPath(tag), filepath.Join(layout.AccountDir(tag), "auth.json"))
	}
}

func TestValidateTag(t *testing.T) {
	cases := []struct {
		tag     string
		wantErr bool
	}{
		{"work", false},
		{"personal", false},
		{"work_2", false},
		{"!work-2", true},
		{"team/project", true},
		{"../secret", true},
		{"", true},
	}

	for _, c := range cases {
		t.Run(c.tag, func(t *testing.T) {
			err := ValidateTag(c.tag)
			if gotErr := err != nil; gotErr != c.wantErr {
				t.Fatalf("ValidateTag(%q) error = %v, wantErr %v", c.tag, gotErr, c.wantErr)
			}
		})
	}
}

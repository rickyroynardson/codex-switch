package paths

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
)

const EnvHome = "CODEX_SWITCH_HOME"

var tagPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type Layout struct {
	Home           string
	AccountsDir    string
	SharedDir      string
	RuntimeDir     string
	CurrentHomeDir string
	StateDir       string
	BinDir         string

	RegistryPath string
	WrapperPath  string
}

func (l Layout) AccountDir(tag string) string {
	return filepath.Join(l.AccountsDir, tag)
}

func (l Layout) AccountAuthPath(tag string) string {
	return filepath.Join(l.AccountDir(tag), "auth.json")
}

// Get default home directory for codex-switch, which is ~/.codex-switch (default). Can be overridden by CODEX_SWITCH_HOME env var.
func DefaultHome() (string, error) {
	if v := os.Getenv(EnvHome); v != "" {
		return v, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, ".codex-switch"), nil
}

func NewLayout(home string) Layout {
	return Layout{
		Home:           home,
		AccountsDir:    filepath.Join(home, "accounts"),
		SharedDir:      filepath.Join(home, "shared"),
		RuntimeDir:     filepath.Join(home, "runtime"),
		CurrentHomeDir: filepath.Join(home, "runtime", "current-home"),
		StateDir:       filepath.Join(home, "state"),
		BinDir:         filepath.Join(home, "bin"),
		RegistryPath:   filepath.Join(home, "state", "accounts.json"),
		WrapperPath:    filepath.Join(home, "bin", "codex"),
	}
}

// Get the default layout for codex-switch.
func DefaultLayout() (Layout, error) {
	home, err := DefaultHome()
	if err != nil {
		return Layout{}, err
	}
	return NewLayout(home), nil
}

// Validate given tag with safe pattern.
func ValidateTag(tag string) error {
	if tag == "" {
		return errors.New("tag cannot be empty")
	}

	if !tagPattern.MatchString(tag) {
		return errors.New("tag may only contain letters, numbers, . (dot), _ (underscore), and - (dash), and must start with a letter or number")
	}

	return nil
}

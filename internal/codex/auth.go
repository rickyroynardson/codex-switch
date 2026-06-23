package codex

import (
	"os"
	"path/filepath"
)

const FileAuthConfig = `cli_auth_credentials_store = "file"`

func EnsureFileAuthConfig(codexHome string) error {
	if err := os.MkdirAll(codexHome, 0700); err != nil {
		return err
	}

	configPath := filepath.Join(codexHome, "config.toml")
	if err := os.WriteFile(configPath, []byte(FileAuthConfig), 0600); err != nil {
		return err
	}

	return nil
}

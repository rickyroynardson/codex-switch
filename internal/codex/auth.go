package codex

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const FileAuthConfig = `cli_auth_credentials_store = "file"`

type authFile struct {
	Tokens struct {
		IDToken string `json:"id_token"`
	} `json:"tokens"`
}

type idTokenClaims struct {
	Email string `json:"email"`
}

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

func ReadEmailFromAuthFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var auth authFile
	if err := json.Unmarshal(b, &auth); err != nil {
		return "", err
	}

	if auth.Tokens.IDToken == "" {
		return "", nil
	}

	parts := strings.Split(auth.Tokens.IDToken, ".")
	if len(parts) < 2 {
		return "", nil
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", err
	}

	var claims idTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", err
	}

	return claims.Email, nil
}

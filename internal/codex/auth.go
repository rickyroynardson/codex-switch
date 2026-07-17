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

// EnsureFileAuthConfig guarantees config.toml carries the file-auth line
// without discarding any existing config. The account dir is the CODEX_HOME
// Codex actually runs against, so truncating it here would wipe the user's
// real settings (MCP servers, trust, model). Idempotent: rewrites only when
// the file-auth line is missing or set to something else.
func EnsureFileAuthConfig(codexHome string) error {
	if err := os.MkdirAll(codexHome, 0700); err != nil {
		return err
	}

	configPath := filepath.Join(codexHome, "config.toml")

	existing, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	updated := ensureFileAuthLine(string(existing))
	if string(existing) == updated {
		return nil
	}

	return os.WriteFile(configPath, []byte(updated), 0600)
}

// ensureFileAuthLine returns contents with the file-auth line as a top-level
// key, replacing any existing cli_auth_credentials_store setting and keeping
// everything else. Prepending keeps it above any [table], where a top-level
// key must live.
func ensureFileAuthLine(contents string) string {
	var kept []string
	for _, line := range strings.Split(contents, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "cli_auth_credentials_store") {
			continue
		}
		kept = append(kept, line)
	}

	body := strings.TrimSpace(strings.Join(kept, "\n"))
	if body == "" {
		return FileAuthConfig + "\n"
	}

	return FileAuthConfig + "\n" + body + "\n"
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

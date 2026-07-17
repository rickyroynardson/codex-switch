package codex

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureFileAuthConfigCreatesHome(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "codex-home")

	err := EnsureFileAuthConfig(dir)
	assert.NoError(t, err)

	info, err := os.Stat(dir)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestEnsureFileAuthConfigWritesConfig(t *testing.T) {
	dir := t.TempDir()

	err := EnsureFileAuthConfig(dir)
	assert.NoError(t, err)

	b, err := os.ReadFile(filepath.Join(dir, "config.toml"))
	assert.NoError(t, err)
	assert.Equal(t, FileAuthConfig+"\n", string(b))
}

func TestEnsureFileAuthConfigPreservesExistingConfig(t *testing.T) {
	dir := t.TempDir()

	existing := `model = "gpt-5"

[projects."/tmp/my-project"]
trust_level = "trusted"
`
	err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(existing), 0600)
	assert.NoError(t, err)

	err = EnsureFileAuthConfig(dir)
	assert.NoError(t, err)

	b, err := os.ReadFile(filepath.Join(dir, "config.toml"))
	assert.NoError(t, err)
	assert.Contains(t, string(b), FileAuthConfig)
	assert.Contains(t, string(b), `model = "gpt-5"`)
	assert.Contains(t, string(b), `trust_level = "trusted"`)
}

func TestEnsureFileAuthConfigReplacesConflictingAuthStore(t *testing.T) {
	dir := t.TempDir()

	err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(`cli_auth_credentials_store = "keychain"`+"\n"+`model = "gpt-5"`), 0600)
	assert.NoError(t, err)

	err = EnsureFileAuthConfig(dir)
	assert.NoError(t, err)

	b, err := os.ReadFile(filepath.Join(dir, "config.toml"))
	assert.NoError(t, err)
	assert.NotContains(t, string(b), "keychain")
	assert.Contains(t, string(b), FileAuthConfig)
	assert.Contains(t, string(b), `model = "gpt-5"`)
}

func TestEnsureFileAuthConfigIdempotent(t *testing.T) {
	dir := t.TempDir()

	err := EnsureFileAuthConfig(dir)
	assert.NoError(t, err)

	first, err := os.ReadFile(filepath.Join(dir, "config.toml"))
	assert.NoError(t, err)

	err = EnsureFileAuthConfig(dir)
	assert.NoError(t, err)

	second, err := os.ReadFile(filepath.Join(dir, "config.toml"))
	assert.NoError(t, err)
	assert.Equal(t, string(first), string(second))
}

func TestReadEmailFromAuthFile(t *testing.T) {
	dir := t.TempDir()
	authPath := filepath.Join(dir, "auth.json")

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"email": "work@mail.com"}`))
	idToken := header + "." + payload + "."

	authJSON := `{"tokens": {"id_token": "` + idToken + `"}}`
	writeAuthJSON(t, authPath, authJSON)

	email, err := ReadEmailFromAuthFile(authPath)
	require.NoError(t, err)
	assert.Equal(t, "work@mail.com", email)
}

func TestReadEmailFromAuthFileReturnsEmptyWhenNoIDToken(t *testing.T) {
	dir := t.TempDir()
	authPath := filepath.Join(dir, "auth.json")

	authJSON := `{"tokens": {}}`
	writeAuthJSON(t, authPath, authJSON)

	email, err := ReadEmailFromAuthFile(authPath)
	require.NoError(t, err)
	assert.Empty(t, email)
}

func TestReadEmailFromAuthFileReturnsErrorWhenMalformedToken(t *testing.T) {
	dir := t.TempDir()
	authPath := filepath.Join(dir, "auth.json")

	authJSON := `{"tokens": {"id_token": "malformed.token"}}`
	writeAuthJSON(t, authPath, authJSON)

	email, err := ReadEmailFromAuthFile(authPath)
	require.Error(t, err)
	assert.Empty(t, email)
}

func writeAuthJSON(t *testing.T, path, json string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("create auth dir: %v", err)
	}

	if err := os.WriteFile(path, []byte(json), 0600); err != nil {
		t.Fatalf("write auth file: %v", err)
	}
}

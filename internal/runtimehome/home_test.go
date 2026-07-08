package runtimehome

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rickyroynardson/codex-switch/internal/codex"
	"github.com/rickyroynardson/codex-switch/internal/paths"
	"github.com/rickyroynardson/codex-switch/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsSharedEntryReturnsTrueForAllowedEntries(t *testing.T) {
	cases := []string{"config.toml", "sessions", "skills", "plugins"}
	for _, name := range cases {
		if !IsSharedEntry(name) {
			t.Errorf("expecting %q to be shared entry", name)
		}
	}
}

func TestIsSharedEntryReturnsFalseForSensitiveOrUnknownEntries(t *testing.T) {
	cases := []string{"auth.json", "mcp_oauth.json", "log", "unknown"}
	for _, name := range cases {
		if IsSharedEntry(name) {
			t.Errorf("expecting %q to not be shared entry", name)
		}
	}
}

func writeAuthFile(t *testing.T, authPath, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(authPath), 0700); err != nil {
		t.Fatalf("create auth dir: %v", err)
	}

	if err := os.WriteFile(authPath, []byte(contents), 0600); err != nil {
		t.Fatalf("write auth file: %v", err)
	}
}

func TestAssembleCopiesAuthFile(t *testing.T) {
	layout := paths.NewLayout(t.TempDir())
	authPath := layout.AccountAuthPath("work")
	writeAuthFile(t, authPath, `{"token":"work"}`)

	account := state.Account{
		Tag:      "work",
		AuthPath: authPath,
	}

	err := Assemble(layout, account)
	assert.NoError(t, err)

	b, err := os.ReadFile(filepath.Join(layout.CurrentHomeDir, "auth.json"))
	assert.NoError(t, err)
	assert.Equal(t, `{"token":"work"}`, string(b))
}

func TestAssembleLinksSharedEntries(t *testing.T) {
	layout := paths.NewLayout(t.TempDir())
	authPath := layout.AccountAuthPath("work")
	writeAuthFile(t, authPath, `{}`)

	if err := os.MkdirAll(layout.SharedDir, 0700); err != nil {
		t.Fatalf("created shared dir: %v", err)
	}

	sharedConfig := filepath.Join(layout.SharedDir, "config.toml")
	if err := os.WriteFile(sharedConfig, []byte("shared config"), 0600); err != nil {
		t.Fatalf("write shared config: %v", err)
	}

	sharedSkills := filepath.Join(layout.SharedDir, "skills")
	if err := os.MkdirAll(sharedSkills, 0700); err != nil {
		t.Fatalf("create shared skills: %v", err)
	}

	account := state.Account{
		Tag:      "work",
		AuthPath: authPath,
	}

	err := Assemble(layout, account)
	assert.NoError(t, err)

	configLink := filepath.Join(layout.CurrentHomeDir, "config.toml")
	configInfo, err := os.Lstat(configLink)
	assert.NoError(t, err)
	assert.NotZero(t, configInfo.Mode()&os.ModeSymlink)

	configTarget, err := os.Readlink(configLink)
	assert.NoError(t, err)
	assert.Equal(t, sharedConfig, configTarget)

	skillsLink := filepath.Join(layout.CurrentHomeDir, "skills")
	skillsInfo, err := os.Lstat(skillsLink)
	assert.NoError(t, err)
	assert.NotZero(t, skillsInfo.Mode()&os.ModeSymlink)

	skillsTarget, err := os.Readlink(skillsLink)
	assert.NoError(t, err)
	assert.Equal(t, sharedSkills, skillsTarget)
}

func TestAssembleSkipsMissingSharedEntries(t *testing.T) {
	layout := paths.NewLayout(t.TempDir())
	authPath := layout.AccountAuthPath("work")
	writeAuthFile(t, authPath, `{}`)

	account := state.Account{
		Tag:      "work",
		AuthPath: authPath,
	}

	err := Assemble(layout, account)
	assert.NoError(t, err)

	_, err = os.Lstat(filepath.Join(layout.CurrentHomeDir, "auth.json"))
	assert.NoError(t, err)

	_, err = os.Lstat(filepath.Join(layout.CurrentHomeDir, "plugins"))
	assert.True(t, os.IsNotExist(err))
}

func TestAssembleReplacesExistingCurrentHome(t *testing.T) {
	layout := paths.NewLayout(t.TempDir())
	authPath := layout.AccountAuthPath("work")
	writeAuthFile(t, authPath, `{"token":"new"}`)

	if err := os.MkdirAll(layout.CurrentHomeDir, 0700); err != nil {
		t.Fatalf("create existing current home: %v", err)
	}

	stalePath := filepath.Join(layout.CurrentHomeDir, "stale.txt")
	if err := os.WriteFile(stalePath, []byte("stale"), 0600); err != nil {
		t.Fatalf("write stale file: %v", err)
	}

	account := state.Account{
		Tag:      "work",
		AuthPath: authPath,
	}

	err := Assemble(layout, account)
	assert.NoError(t, err)

	_, err = os.Stat(stalePath)
	assert.True(t, os.IsNotExist(err))

	b, err := os.ReadFile(filepath.Join(layout.CurrentHomeDir, "auth.json"))
	assert.NoError(t, err)
	assert.Equal(t, `{"token":"new"}`, string(b))
}

func TestAssembleReturnsErrorWhenAuthMissing(t *testing.T) {
	layout := paths.NewLayout(t.TempDir())

	account := state.Account{
		Tag:      "work",
		AuthPath: layout.AccountAuthPath("work"),
	}

	err := Assemble(layout, account)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "account auth file not found")
}

func TestAssembleCreatesRequiredSharedDirs(t *testing.T) {
	layout := paths.NewLayout(t.TempDir())
	authPath := layout.AccountAuthPath("work")
	writeAuthFile(t, authPath, `{}`)

	account := state.Account{
		Tag:      "work",
		AuthPath: authPath,
	}

	err := Assemble(layout, account)
	require.NoError(t, err)

	for _, name := range []string{"sessions", "archived_sessions"} {
		sharedPath := filepath.Join(layout.SharedDir, name)

		sharedInfo, err := os.Stat(sharedPath)
		require.NoError(t, err)
		assert.True(t, sharedInfo.IsDir())

		runtimePath := filepath.Join(layout.CurrentHomeDir, name)

		runtimeInfo, err := os.Lstat(runtimePath)
		require.NoError(t, err)
		assert.NotZero(t, runtimeInfo.Mode()&os.ModeSymlink)

		target, err := os.Readlink(runtimePath)
		require.NoError(t, err)
		assert.Equal(t, sharedPath, target)
	}
}

func TestAssembleCreatesSharedConfigFile(t *testing.T) {
	layout := paths.NewLayout(t.TempDir())
	authPath := layout.AccountAuthPath("work")
	writeAuthFile(t, authPath, `{}`)

	account := state.Account{
		Tag:      "work",
		AuthPath: authPath,
	}

	err := Assemble(layout, account)
	require.NoError(t, err)

	sharedConfig := filepath.Join(layout.SharedDir, "config.toml")
	b, err := os.ReadFile(sharedConfig)
	require.NoError(t, err)
	assert.Equal(t, codex.FileAuthConfig+"\n", string(b))

	runtimeConfig := filepath.Join(layout.CurrentHomeDir, "config.toml")
	info, err := os.Lstat(runtimeConfig)
	require.NoError(t, err)
	assert.NotZero(t, info.Mode()&os.ModeSymlink)

	target, err := os.Readlink(runtimeConfig)
	require.NoError(t, err)
	assert.Equal(t, sharedConfig, target)
}

func TestAssembleDoesNotOverwriteExistingSharedConfigFile(t *testing.T) {
	layout := paths.NewLayout(t.TempDir())
	authPath := layout.AccountAuthPath("work")
	writeAuthFile(t, authPath, `{}`)

	if err := os.MkdirAll(layout.SharedDir, 0700); err != nil {
		t.Fatalf("create shared dir: %v", err)
	}

	sharedConfig := filepath.Join(layout.SharedDir, "config.toml")
	existingConfig := codex.FileAuthConfig + `

[projects."/tmp/my-project"]
trust_level = "trusted"
`

	if err := os.WriteFile(sharedConfig, []byte(existingConfig), 0600); err != nil {
		t.Fatalf("write shared config: %v", err)
	}

	account := state.Account{
		Tag:      "work",
		AuthPath: authPath,
	}

	err := Assemble(layout, account)
	require.NoError(t, err)

	b, err := os.ReadFile(sharedConfig)
	require.NoError(t, err)
	assert.Equal(t, existingConfig, string(b))

	runtimeConfig := filepath.Join(layout.CurrentHomeDir, "config.toml")
	target, err := os.Readlink(runtimeConfig)
	require.NoError(t, err)
	assert.Equal(t, sharedConfig, target)
}

func TestImportSharedStateCopiesSharedEntities(t *testing.T) {
	layout := paths.NewLayout(t.TempDir())
	sourceHome := filepath.Join(t.TempDir(), ".codex")

	if err := os.MkdirAll(filepath.Join(sourceHome, "sessions"), 0700); err != nil {
		t.Fatalf("failed create source sessions: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceHome, "config.toml"), []byte(`model="gpt-5"`), 0600); err != nil {
		t.Fatalf("failed create source config.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceHome, "sessions", "one.jsonl"), []byte("{}\n"), 0600); err != nil {
		t.Fatalf("failed create source session: %v", err)
	}

	err := ImportSharedState(layout, sourceHome)
	assert.NoError(t, err)

	config, err := os.ReadFile(filepath.Join(layout.SharedDir, "config.toml"))
	assert.NoError(t, err)
	assert.Contains(t, string(config), `model="gpt-5"`)

	session, err := os.ReadFile(filepath.Join(layout.SharedDir, "sessions", "one.jsonl"))
	assert.NoError(t, err)
	assert.Equal(t, "{}\n", string(session))
}

func TestImportSharedStateSkipsSensitiveEntries(t *testing.T) {
	layout := paths.NewLayout(t.TempDir())
	sourceHome := filepath.Join(t.TempDir(), ".codex")

	if err := os.MkdirAll(sourceHome, 0700); err != nil {
		t.Fatalf("failed create source home: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceHome, "auth.json"), []byte(`secret`), 0600); err != nil {
		t.Fatalf("failed write auth: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceHome, "mcp_oauth.json"), []byte(`secret`), 0600); err != nil {
		t.Fatalf("failed write mcp auth: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceHome, "logs_2.sqlite"), []byte(`logs`), 0600); err != nil {
		t.Fatalf("failed write logs: %v", err)
	}

	err := ImportSharedState(layout, sourceHome)
	assert.NoError(t, err)

	_, err = os.Stat(filepath.Join(layout.SharedDir, "auth.json"))
	assert.True(t, os.IsNotExist(err))

	_, err = os.Stat(filepath.Join(layout.SharedDir, "mcp_oauth.json"))
	assert.True(t, os.IsNotExist(err))

	_, err = os.Stat(filepath.Join(layout.SharedDir, "logs_2.sqlite"))
	assert.True(t, os.IsNotExist(err))
}

func TestImportSharedStateIgnoresMissingSource(t *testing.T) {
	layout := paths.NewLayout(t.TempDir())
	sourceHome := filepath.Join(t.TempDir(), ".codex-missing")

	err := ImportSharedState(layout, sourceHome)
	assert.NoError(t, err)
}

func TestPersistSharedStateCopiesRuntimeCreatedSharedFile(t *testing.T) {
	layout := paths.NewLayout(t.TempDir())

	if err := os.MkdirAll(layout.CurrentHomeDir, 0700); err != nil {
		t.Fatalf("failed create current home: %v", err)
	}

	runtimePath := filepath.Join(layout.CurrentHomeDir, "AGENTS.md")
	if err := os.WriteFile(runtimePath, []byte("shared instructions"), 0600); err != nil {
		t.Fatalf("failed write runtime shared file: %v", err)
	}

	err := PersistSharedState(layout)
	assert.NoError(t, err)

	b, err := os.ReadFile(filepath.Join(layout.SharedDir, "AGENTS.md"))
	assert.NoError(t, err)
	assert.Equal(t, "shared instructions", string(b))
}

func TestPersistSharedStateCopiesRuntimeCreatedSharedDir(t *testing.T) {
	layout := paths.NewLayout(t.TempDir())

	runtimePromptsDir := filepath.Join(layout.CurrentHomeDir, "prompts")
	if err := os.MkdirAll(runtimePromptsDir, 0700); err != nil {
		t.Fatalf("failed create runtime prompts dir: %v", err)
	}

	runtimePromptPath := filepath.Join(runtimePromptsDir, "REVIEW.md")
	if err := os.WriteFile(runtimePromptPath, []byte("review prompt"), 0600); err != nil {
		t.Fatalf("failed write runtime prompt: %v", err)
	}

	err := PersistSharedState(layout)
	assert.NoError(t, err)

	b, err := os.ReadFile(filepath.Join(layout.SharedDir, "prompts", "REVIEW.md"))
	assert.NoError(t, err)
	assert.Equal(t, "review prompt", string(b))
}

func TestNormalizeSharedConfigAddsFileAuthToEmptyConfig(t *testing.T) {
	got := normalizeSharedConfigContents("")
	assert.Equal(t, `cli_auth_credentials_store = "file"`, got)
}

func TestNormalizeSharedConfigPreservesTopLevelConfig(t *testing.T) {
	input := `model = "gpt-5"`
	got := normalizeSharedConfigContents(input)

	assert.Equal(t, `model = "gpt-5"
cli_auth_credentials_store = "file"`, got)
}

func TestNormalizeSharedConfigInsertsFileAuthBeforeTables(t *testing.T) {
	input := `model = "gpt-5"

[projects."/tmp/project"]
trust_level = "trusted"`

	got := normalizeSharedConfigContents(input)

	assert.Equal(t, `model = "gpt-5"
cli_auth_credentials_store = "file"

[projects."/tmp/project"]
trust_level = "trusted"`, got)
}

func TestNormalizeSharedConfigReplacesExistingAuthStore(t *testing.T) {
	input := `cli_auth_credentials_store = "keychain"
model = "gpt-5"`

	got := normalizeSharedConfigContents(input)

	assert.Equal(t, `model = "gpt-5"
cli_auth_credentials_store = "file"`, got)
}

func TestAssembleNormalizesExistingSharedConfigFile(t *testing.T) {
	layout := paths.NewLayout(t.TempDir())
	authPath := layout.AccountAuthPath("work")
	writeAuthFile(t, authPath, `{}`)

	if err := os.MkdirAll(layout.SharedDir, 0700); err != nil {
		t.Fatalf("failed create shared dir: %v", err)
	}

	sharedConfig := filepath.Join(layout.SharedDir, "config.toml")
	existingConfig := `model = "gpt-5"

[projects."/tmp/my-project"]
trust_level = "trusted"
`

	if err := os.WriteFile(sharedConfig, []byte(existingConfig), 0600); err != nil {
		t.Fatalf("failed write shared config: %v", err)
	}

	account := state.Account{
		Tag:      "work",
		AuthPath: authPath,
	}

	err := Assemble(layout, account)
	assert.NoError(t, err)

	b, err := os.ReadFile(sharedConfig)
	assert.NoError(t, err)

	assert.Equal(t, `model = "gpt-5"
cli_auth_credentials_store = "file"

[projects."/tmp/my-project"]
trust_level = "trusted"
`, string(b))
}

func TestImportSharedStateNormalizesConfig(t *testing.T) {
	layout := paths.NewLayout(t.TempDir())
	sourceHome := filepath.Join(t.TempDir(), ".codex")

	if err := os.MkdirAll(sourceHome, 0700); err != nil {
		t.Fatalf("create source home: %v", err)
	}

	sourceConfig := `model = "gpt-5"

[projects."/tmp/project"]
trust_level = "trusted"
`
	if err := os.WriteFile(filepath.Join(sourceHome, "config.toml"), []byte(sourceConfig), 0600); err != nil {
		t.Fatalf("write source config: %v", err)
	}

	err := ImportSharedState(layout, sourceHome)
	require.NoError(t, err)

	b, err := os.ReadFile(filepath.Join(layout.SharedDir, "config.toml"))
	require.NoError(t, err)

	assert.Equal(t, `model = "gpt-5"
cli_auth_credentials_store = "file"

[projects."/tmp/project"]
trust_level = "trusted"
`, string(b))
}

func TestPersistSharedStateNormalizesRuntimeConfig(t *testing.T) {
	layout := paths.NewLayout(t.TempDir())

	if err := os.MkdirAll(layout.CurrentHomeDir, 0700); err != nil {
		t.Fatalf("create current home: %v", err)
	}

	runtimeConfig := `cli_auth_credentials_store = "keychain"
model = "gpt-5"
`
	if err := os.WriteFile(filepath.Join(layout.CurrentHomeDir, "config.toml"), []byte(runtimeConfig), 0600); err != nil {
		t.Fatalf("write runtime config: %v", err)
	}

	err := PersistSharedState(layout)
	require.NoError(t, err)

	b, err := os.ReadFile(filepath.Join(layout.SharedDir, "config.toml"))
	require.NoError(t, err)

	assert.Equal(t, `model = "gpt-5"
cli_auth_credentials_store = "file"
`, string(b))
}

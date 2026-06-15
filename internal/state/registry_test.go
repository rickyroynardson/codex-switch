package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	if registry.ActiveTag != "" {
		t.Fatalf("new registry active tag should be empty")
	}
	if registry.Accounts == nil {
		t.Fatalf("new registry accounts should be declared (not nil)")
	}
	if len(registry.Accounts) != 0 {
		t.Fatalf("new registry accounts should be empty")
	}
}

func TestValidAuthState(t *testing.T) {
	cases := []struct {
		state     AuthState
		wantValid bool
	}{
		{AuthStateUnknown, true},
		{AuthStateReady, true},
		{AuthStateNeedsLogin, true},
		{AuthStateInvalid, true},
		{AuthState("other"), false},
		{AuthState(""), false},
		{AuthState("banana"), false},
	}

	for _, c := range cases {
		t.Run(string(c.state), func(t *testing.T) {
			if v := c.state.Valid(); v != c.wantValid {
				t.Fatalf("expected %s to be valid: %t, got: %t", c.state, c.wantValid, v)
			}
		})
	}
}

func TestRegistryJSON(t *testing.T) {
	r := NewRegistry()
	r.ActiveTag = "test"
	r.Accounts = []Account{
		{
			Tag:       "test",
			AuthPath:  "/tmp/auth.json",
			Email:     "me@test.com",
			AuthState: AuthStateReady,
			CreatedAt: "2026-06-12T00:00:00Z",
			UpdatedAt: "2026-06-12T00:00:00Z",
		},
	}

	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("failed to json marshal registry: %v", err)
	}

	unmarshaled := Registry{}
	if err := json.Unmarshal(b, &unmarshaled); err != nil {
		t.Fatalf("failed to json unmarshal registry: %v", err)
	}

	if !reflect.DeepEqual(unmarshaled, r) {
		t.Fatalf("unmarshaled registry %#v, want %#v", unmarshaled, r)
	}
}

func tempPath(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	return path
}

func TestLoadRegistryMissingFileReturnsEmptyRegistry(t *testing.T) {
	path := tempPath(t, "accounts.json")
	registry, err := LoadRegistry(path)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := NewRegistry()
	if !reflect.DeepEqual(registry, expected) {
		t.Fatalf("unexpected registry: %#v, want %#v", registry, expected)
	}
}

func TestLoadRegistryReadsFile(t *testing.T) {
	path := tempPath(t, "accounts.json")
	if err := os.WriteFile(path, []byte(`{"activeTag":"test","accounts":[]}`), 0600); err != nil {
		t.Fatalf("failed to write registry: %v", err)
	}
	registry, err := LoadRegistry(path)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if registry.ActiveTag != "test" {
		t.Fatalf("registry.ActiveTag got %s, want test", registry.ActiveTag)
	}

	if registry.Accounts == nil {
		t.Fatalf("registry.Accounts should be declared (not nil)")
	}

	if len(registry.Accounts) != 0 {
		t.Fatalf("len(registry.Accounts) = %d, want 0", len(registry.Accounts))
	}
}

func TestLoadRegistryInvalidJSON(t *testing.T) {
	path := tempPath(t, "accounts.json")
	if err := os.WriteFile(path, []byte("data"), 0600); err != nil {
		t.Fatalf("failed to write registry: %v", err)
	}
	_, err := LoadRegistry(path)

	if err == nil {
		t.Fatalf("expecting error, got nil")
	}
}

func TestSaveRegistryWritesFile(t *testing.T) {
	registry := NewRegistry()
	registry.ActiveTag = "test"

	path := tempPath(t, "accounts.json")

	err := SaveRegistry(path, registry)
	if err != nil {
		t.Fatalf("failed to save registry: %v", err)
	}

	r, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("failed to load registry: %v", err)
	}

	if !reflect.DeepEqual(r, registry) {
		t.Fatalf("registry got %#v, want %#v", r, registry)
	}
}

func TestSaveRegistryCreatesParentDirectory(t *testing.T) {
	registry := NewRegistry()
	registry.ActiveTag = "test"

	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "accounts.json")
	if err := SaveRegistry(path, registry); err != nil {
		t.Fatalf("failed to save registry: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected registry file to be exists: %v", err)
	}
}

func TestSaveRegistryNormalizesNilAccounts(t *testing.T) {
	registry := NewRegistry()
	registry.Accounts = nil

	path := tempPath(t, "accounts.json")
	if err := SaveRegistry(path, registry); err != nil {
		t.Fatalf("failed to save registry: %v", err)
	}

	r, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("failed to load registry: %v", err)
	}

	if r.Accounts == nil {
		t.Fatalf("registry.Accounts should be declared (not nil)")
	}

	if len(r.Accounts) != 0 {
		t.Fatalf("len(registry.Accounts) = %d, want 0", len(r.Accounts))
	}
}

func TestRegistryFindAccount(t *testing.T) {
	cases := []struct {
		tag       string
		wantFound bool
	}{
		{tag: "test-1", wantFound: true},
		{tag: "test-2", wantFound: true},
		{tag: "other", wantFound: false},
	}

	registry := NewRegistry()
	registry.Accounts = []Account{
		{Tag: "test-1"},
		{Tag: "test-2"},
	}

	for _, c := range cases {
		t.Run(c.tag, func(t *testing.T) {
			_, gotFound := registry.FindAccount(c.tag)
			if gotFound != c.wantFound {
				t.Fatalf("FindAccount(%s) got found: %t, want: %t", c.tag, gotFound, c.wantFound)
			}
		})
	}
}

func TestRegistryActiveAccount(t *testing.T) {
	registry := NewRegistry()
	registry.Accounts = []Account{
		{Tag: "test"},
	}

	if _, ok := registry.ActiveAccount(); ok {
		t.Fatalf("expected to not find active account")
	}

	registry.ActiveTag = "test"
	acc, ok := registry.ActiveAccount()
	if !ok {
		t.Fatalf("expected to find active account")
	}
	if acc.Tag != "test" {
		t.Fatalf("active account tag got %s, want test", acc.Tag)
	}

	registry.ActiveTag = "other"
	if _, ok := registry.ActiveAccount(); ok {
		t.Fatalf("expected to not find active account with tag other")
	}
}

func TestRegistryUpsertAccountAddsNewAccount(t *testing.T) {
	registry := NewRegistry()
	acc := Account{
		Tag: "test",
	}

	registry.UpsertAccount(acc)

	a, ok := registry.FindAccount(acc.Tag)
	assert.True(t, ok)

	assert.Len(t, registry.Accounts, 1)
	assert.Equal(t, acc.Tag, a.Tag)
	assert.Equal(t, a.Tag, registry.ActiveTag)
}

func TestRegistryUpsertAccountUpdatesExistingAccount(t *testing.T) {
	registry := NewRegistry()
	createdAt := time.Now().String()
	acc := Account{
		Tag:       "test",
		AuthPath:  "/tmp/auth.json",
		AuthState: AuthStateNeedsLogin,
		CreatedAt: createdAt,
		UpdatedAt: time.Now().String(),
	}
	registry.UpsertAccount(acc)

	acc = Account{
		Tag:       "test",
		AuthPath:  "/tmp/new-auth.json",
		Email:     "me@test.com",
		AuthState: AuthStateReady,
		CreatedAt: time.Now().Add(time.Hour).String(),
		UpdatedAt: time.Now().String(),
	}
	registry.UpsertAccount(acc)

	activeAcc, ok := registry.ActiveAccount()
	assert.True(t, ok)

	assert.Len(t, registry.Accounts, 1)
	assert.Equal(t, "/tmp/new-auth.json", activeAcc.AuthPath)
	assert.Equal(t, "me@test.com", activeAcc.Email)
	assert.Equal(t, AuthStateReady, activeAcc.AuthState)
	assert.Equal(t, createdAt, activeAcc.CreatedAt, "CreatedAt should not be updated")
	assert.Equal(t, acc.UpdatedAt, activeAcc.UpdatedAt)
}

func TestRegistryUpsertAccountDoesNotReplaceActiveTag(t *testing.T) {
	registry := NewRegistry()
	registry.ActiveTag = "test"
	registry.Accounts = []Account{
		{Tag: "test"},
	}

	acc := Account{
		Tag: "work",
	}
	registry.UpsertAccount(acc)

	assert.Len(t, registry.Accounts, 2)
	assert.Equal(t, "test", registry.ActiveTag)
}

func TestRegistrySetActiveTag(t *testing.T) {
	t.Run("sets active tag when account exists", func(t *testing.T) {
		registry := NewRegistry()
		registry.Accounts = []Account{
			{Tag: "work"},
			{Tag: "personal"},
		}

		err := registry.SetActiveTag("work")
		assert.NoError(t, err)
		assert.Equal(t, "work", registry.ActiveTag)
	})

	t.Run("return error if tag doest not exists", func(t *testing.T) {
		registry := NewRegistry()
		registry.Accounts = []Account{
			{Tag: "work"},
		}
		registry.ActiveTag = "work"

		err := registry.SetActiveTag("other")
		assert.Error(t, err)
		assert.Equal(t, "work", registry.ActiveTag)
	})
}

func TestRegistryRemoveAccount(t *testing.T) {
	t.Run("removes non-active account", func(t *testing.T) {
		registry := NewRegistry()
		registry.Accounts = []Account{
			{Tag: "work"},
			{Tag: "personal"},
		}
		registry.ActiveTag = "personal"

		err := registry.RemoveAccount("work")
		assert.NoError(t, err)
		assert.Len(t, registry.Accounts, 1)
		a, ok := registry.FindAccount("work")
		assert.False(t, ok)
		assert.Equal(t, Account{}, a)
		_, ok = registry.FindAccount("personal")
		assert.True(t, ok)
	})

	t.Run("returns error for missing account", func(t *testing.T) {
		registry := NewRegistry()

		err := registry.RemoveAccount("work")
		assert.Error(t, err)
	})

	t.Run("returns error for active account", func(t *testing.T) {
		registry := NewRegistry()
		acc := Account{
			Tag: "work",
		}
		registry.UpsertAccount(acc)

		err := registry.RemoveAccount("work")
		assert.Error(t, err)

		_, ok := registry.FindAccount("work")
		assert.True(t, ok)
	})
}

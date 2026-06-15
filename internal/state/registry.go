package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

type AuthState string

const (
	AuthStateUnknown    AuthState = "unknown"
	AuthStateReady      AuthState = "ready"
	AuthStateNeedsLogin AuthState = "needs_login"
	AuthStateInvalid    AuthState = "invalid"
)

type Account struct {
	Tag       string    `json:"tag"`
	AuthPath  string    `json:"authPath"`
	Email     string    `json:"email,omitempty"`
	AuthState AuthState `json:"authState"`
	CreatedAt string    `json:"createdAt"`
	UpdatedAt string    `json:"updatedAt"`
}

type Registry struct {
	ActiveTag string    `json:"activeTag"`
	Accounts  []Account `json:"accounts"`
}

func NewRegistry() Registry {
	return Registry{
		Accounts: []Account{},
	}
}

func (r Registry) FindAccount(tag string) (Account, bool) {
	for _, a := range r.Accounts {
		if a.Tag == tag {
			return a, true
		}
	}
	return Account{}, false
}

func (r Registry) ActiveAccount() (Account, bool) {
	if r.ActiveTag == "" {
		return Account{}, false
	}
	return r.FindAccount(r.ActiveTag)
}

func (r *Registry) UpsertAccount(acc Account) {
	if r.Accounts == nil {
		r.Accounts = []Account{}
	}

	for i, a := range r.Accounts {
		if a.Tag == acc.Tag {
			acc.CreatedAt = a.CreatedAt
			r.Accounts[i] = acc
			return
		}
	}
	r.Accounts = append(r.Accounts, acc)
	if r.ActiveTag == "" {
		r.ActiveTag = acc.Tag
	}
}

func (r *Registry) SetActiveTag(tag string) error {
	if _, ok := r.FindAccount(tag); !ok {
		return fmt.Errorf("unknown account tag: %s", tag)
	}

	r.ActiveTag = tag
	return nil
}

func (r *Registry) RemoveAccount(tag string) error {
	if r.ActiveTag == tag {
		return errors.New("cannot remove active account")
	}

	for i, a := range r.Accounts {
		if a.Tag == tag {
			// r.Accounts = append(r.Accounts[:i], r.Accounts[i+1:]...)
			r.Accounts = slices.Delete(r.Accounts, i, i+1)
			return nil
		}
	}

	return fmt.Errorf("account with tag: %s not found", tag)
}

func LoadRegistry(path string) (Registry, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return NewRegistry(), nil
		}
		return Registry{}, err
	}

	var r Registry
	if err := json.Unmarshal(b, &r); err != nil {
		return Registry{}, err
	}

	if r.Accounts == nil {
		r.Accounts = []Account{}
	}

	return r, nil
}

func SaveRegistry(path string, registry Registry) error {
	if registry.Accounts == nil {
		registry.Accounts = []Account{}
	}

	b, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	tmp := fmt.Sprintf("%s.%d.tmp", path, os.Getpid())
	defer func() {
		_ = os.Remove(tmp)
	}()

	if err := os.WriteFile(tmp, append(b, '\n'), 0600); err != nil {
		return err
	}

	return os.Rename(tmp, path)
}

func (s AuthState) Valid() bool {
	switch s {
	case AuthStateUnknown, AuthStateReady, AuthStateNeedsLogin, AuthStateInvalid:
		return true
	default:
		return false
	}
}

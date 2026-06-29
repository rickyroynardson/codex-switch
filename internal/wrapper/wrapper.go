package wrapper

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func FindRealCodex(wrapperPath, pathEnv string) (string, error) {
	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == "" {
			continue
		}

		candidate := filepath.Join(dir, "codex")

		same, err := samePath(candidate, wrapperPath)
		if err != nil {
			return "", err
		}
		if same {
			continue
		}

		if isExecutable(candidate) {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("real codex executable not found on PATH")
}

func samePath(a, b string) (bool, error) {
	aReal, aErr := filepath.EvalSymlinks(a)
	bReal, bErr := filepath.EvalSymlinks(b)

	if os.IsNotExist(aErr) || os.IsNotExist(bErr) {
		return filepath.Clean(a) == filepath.Clean(b), nil
	}

	if aErr != nil {
		return false, aErr
	}
	if bErr != nil {
		return false, bErr
	}

	return filepath.Clean(aReal) == filepath.Clean(bReal), nil
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	if info.IsDir() {
		return false
	}

	return info.Mode()&0111 != 0
}

func Install(wrapperPath, realCodexPath, launcher string) error {
	if wrapperPath == "" {
		return fmt.Errorf("wrapper path is required")
	}

	if realCodexPath == "" {
		return fmt.Errorf("real codex path is required")
	}

	if launcher == "" {
		launcher = "codex-switch"
	}

	if err := os.MkdirAll(filepath.Dir(wrapperPath), 0700); err != nil {
		return err
	}

	contents := renderWrapperScript(realCodexPath, launcher)

	if err := os.WriteFile(wrapperPath, []byte(contents), 0755); err != nil {
		return err
	}

	return nil
}

func renderWrapperScript(realCodexPath, launcher string) string {
	return fmt.Sprintf(`#!/bin/sh
export CODEX_SWITCH_REAL_CODEX=%s
exec %s proxy "$@"
`, shellQuote(realCodexPath), shellQuote(launcher))
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

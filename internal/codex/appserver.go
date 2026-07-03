package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

const defaultAppServerTimeout = 30 * time.Second

type AppServerRequestOptions struct {
	CodexHome    string
	CodexCommand string
	Method       string
	Params       any
	Timeout      time.Duration
}

type AppServerRequester func(opts AppServerRequestOptions, out any) error

type appServerEnvelope struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  json.RawMessage `json:"error,omitempty"`
}

func RequestAppServer(opts AppServerRequestOptions, out any) error {
	if opts.CodexHome == "" {
		return errors.New("codex home is required")
	}
	if opts.Method == "" {
		return errors.New("app-server method is required")
	}

	if err := EnsureFileAuthConfig(opts.CodexHome); err != nil {
		return err
	}

	command := opts.CodexCommand
	if command == "" {
		command = "codex"
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = defaultAppServerTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, "app-server")
	cmd.Env = append(os.Environ(), "CODEX_HOME="+opts.CodexHome)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	stderrDone := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(stderr)
		stderrDone <- string(b)
	}()

	defer func() {
		_ = stdin.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}()

	writeJSONLine := func(v any) error {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		b = append(b, '\n')
		_, err = stdin.Write(b)
		return err
	}

	if err := writeJSONLine(map[string]any{
		"id":     1,
		"method": "initialize",
		"params": map[string]any{
			"clientInfo": map[string]any{
				"name":    "codex-switch",
				"title":   nil,
				"version": "1.0.0",
			},
			"capabilities": map[string]any{
				"experimentalApi": true,
			},
		},
	}); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 74*1024), 1024*1024)

	requestSent := false
	for scanner.Scan() {
		var msg appServerEnvelope
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}

		if len(msg.Error) > 0 {
			return fmt.Errorf("codex app-server request failed: %s", string(msg.Error))
		}

		if msg.ID == 1 && !requestSent {
			params := opts.Params
			if params == nil {
				params = map[string]any{}
			}

			if err := writeJSONLine(map[string]any{
				"id":     2,
				"method": opts.Method,
				"params": params,
			}); err != nil {
				return err
			}

			requestSent = true
			continue
		}

		if msg.ID == 2 {
			if out == nil {
				return nil
			}
			return json.Unmarshal(msg.Result, out)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if ctx.Err() != nil {
		return fmt.Errorf("codex app-server timed out after %s", timeout)
	}

	select {
	case stderrText := <-stderrDone:
		if strings.TrimSpace(stderrText) != "" {
			return errors.New(strings.TrimSpace(stderrText))
		}
	default:
	}

	return errors.New("codex app-server exited before responding")
}

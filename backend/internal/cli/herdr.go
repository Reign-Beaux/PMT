package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// herdr son las operaciones de herdr que usa pmt. La real llama al comando
// herdr y lee su JSON; las pruebas usan una falsa.
type herdr interface {
	Workspaces(ctx context.Context) ([]herdrWorkspace, error)
	CreateWorkspace(ctx context.Context, cwd, label string) (herdrWorkspace, herdrTab, string, error)
	Tabs(ctx context.Context, workspace string) ([]herdrTab, error)
	CreateTab(ctx context.Context, workspace, cwd, label string) (herdrTab, string, error)
	RenameTab(ctx context.Context, tab, label string) error
	FocusTab(ctx context.Context, tab string) error
	Panes(ctx context.Context, workspace string) ([]herdrPane, error)
	// Busy indica si en el pane corre algo en primer plano además del shell.
	Busy(ctx context.Context, pane string) (bool, error)
	// Run escribe un comando en el shell del pane y lo ejecuta.
	Run(ctx context.Context, pane, command string) error
	Agents(ctx context.Context) ([]herdrAgent, error)
	RenameAgent(ctx context.Context, pane, name string) error
}

type herdrWorkspace struct {
	ID    string `json:"workspace_id"`
	Label string `json:"label"`
}

type herdrTab struct {
	ID        string `json:"tab_id"`
	Label     string `json:"label"`
	Workspace string `json:"workspace_id"`
}

type herdrPane struct {
	ID        string `json:"pane_id"`
	Tab       string `json:"tab_id"`
	Workspace string `json:"workspace_id"`
	// Agent es la herramienta que herdr detectó en el pane (claude, codex…).
	Agent string `json:"agent"`
}

type herdrAgent struct {
	Name      string `json:"name"`
	Pane      string `json:"pane_id"`
	Workspace string `json:"workspace_id"`
	Agent     string `json:"agent"`
}

// herdrCLI es herdr por su comando.
type herdrCLI struct{ bin string }

func (h herdrCLI) call(ctx context.Context, out any, args ...string) error {
	cmd := exec.CommandContext(ctx, h.bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	runErr := cmd.Run()
	// Algunos comandos (pane run) no imprimen nada cuando salen bien.
	if runErr == nil && out == nil && len(bytes.TrimSpace(stdout.Bytes())) == 0 {
		return nil
	}
	var resp struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		if runErr != nil {
			return fmt.Errorf("herdr %s: %v: %s", args[0], runErr, strings.TrimSpace(stderr.String()))
		}
		return fmt.Errorf("herdr %s: respuesta inesperada", args[0])
	}
	if resp.Error != nil {
		return fmt.Errorf("herdr %s %s: %s", args[0], args[1], resp.Error.Message)
	}
	if runErr != nil {
		return fmt.Errorf("herdr %s: %v", args[0], runErr)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(resp.Result, out)
}

func (h herdrCLI) Workspaces(ctx context.Context) ([]herdrWorkspace, error) {
	var r struct {
		Workspaces []herdrWorkspace `json:"workspaces"`
	}
	err := h.call(ctx, &r, "workspace", "list")
	return r.Workspaces, err
}

func (h herdrCLI) CreateWorkspace(ctx context.Context, cwd, label string) (herdrWorkspace, herdrTab, string, error) {
	var r struct {
		Workspace herdrWorkspace `json:"workspace"`
		Tab       herdrTab       `json:"tab"`
		RootPane  herdrPane      `json:"root_pane"`
	}
	err := h.call(ctx, &r, "workspace", "create", "--cwd", cwd, "--label", label, "--no-focus")
	return r.Workspace, r.Tab, r.RootPane.ID, err
}

func (h herdrCLI) Tabs(ctx context.Context, workspace string) ([]herdrTab, error) {
	var r struct {
		Tabs []herdrTab `json:"tabs"`
	}
	err := h.call(ctx, &r, "tab", "list", "--workspace", workspace)
	return r.Tabs, err
}

func (h herdrCLI) CreateTab(ctx context.Context, workspace, cwd, label string) (herdrTab, string, error) {
	var r struct {
		Tab      herdrTab  `json:"tab"`
		RootPane herdrPane `json:"root_pane"`
	}
	err := h.call(ctx, &r, "tab", "create", "--workspace", workspace, "--cwd", cwd, "--label", label, "--no-focus")
	return r.Tab, r.RootPane.ID, err
}

func (h herdrCLI) RenameTab(ctx context.Context, tab, label string) error {
	return h.call(ctx, nil, "tab", "rename", tab, label)
}

func (h herdrCLI) FocusTab(ctx context.Context, tab string) error {
	return h.call(ctx, nil, "tab", "focus", tab)
}

func (h herdrCLI) Panes(ctx context.Context, workspace string) ([]herdrPane, error) {
	var r struct {
		Panes []herdrPane `json:"panes"`
	}
	err := h.call(ctx, &r, "pane", "list", "--workspace", workspace)
	return r.Panes, err
}

func (h herdrCLI) Busy(ctx context.Context, pane string) (bool, error) {
	var r struct {
		Info struct {
			ShellPID   int `json:"shell_pid"`
			Foreground []struct {
				PID int `json:"pid"`
			} `json:"foreground_processes"`
		} `json:"process_info"`
	}
	if err := h.call(ctx, &r, "pane", "process-info", "--pane", pane); err != nil {
		return false, err
	}
	for _, p := range r.Info.Foreground {
		if p.PID != r.Info.ShellPID {
			return true, nil
		}
	}
	return false, nil
}

func (h herdrCLI) Run(ctx context.Context, pane, command string) error {
	return h.call(ctx, nil, "pane", "run", pane, command)
}

func (h herdrCLI) Agents(ctx context.Context) ([]herdrAgent, error) {
	var r struct {
		Agents []herdrAgent `json:"agents"`
	}
	err := h.call(ctx, &r, "agent", "list")
	return r.Agents, err
}

func (h herdrCLI) RenameAgent(ctx context.Context, pane, name string) error {
	return h.call(ctx, nil, "agent", "rename", pane, name)
}

// errHerdrMissing indica que no hay comando herdr.
var errHerdrMissing = errors.New("no encuentro el comando herdr")

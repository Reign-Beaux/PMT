package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultServer es el PMT de producción.
const DefaultServer = "https://pmt.saulantonio.com"

// normalizeServer exige http o https con host y quita la barra final.
func normalizeServer(raw string) (string, error) {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("servidor inválido %q: usa una URL como https://pmt.ejemplo.com", raw)
	}
	return raw, nil
}

// apiError es una respuesta de error del servidor.
type apiError struct {
	Status  int
	Message string
}

func (e *apiError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("el servidor respondió %d", e.Status)
	}
	return e.Message
}

// isStatus indica si err es una respuesta del servidor con ese status.
func isStatus(err error, status int) bool {
	var apiErr *apiError
	return errors.As(err, &apiErr) && apiErr.Status == status
}

// client habla con la API de un servidor PMT.
type client struct {
	http   *http.Client
	server string
	token  string
	agent  string
}

// do manda una petición JSON y decodifica la respuesta en out si no es nil.
func (c client) do(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		raw, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.server+path, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.agent)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("no se pudo conectar con %s: %w", c.server, unwrapURLError(err))
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("leer respuesta de %s: %w", c.server, err)
	}
	if resp.StatusCode >= 300 {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(raw, &e)
		return &apiError{Status: resp.StatusCode, Message: e.Error}
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("respuesta inesperada de %s: ¿es un servidor PMT?", c.server)
	}
	return nil
}

// getText pide un texto (Markdown) con GET.
func (c client) getText(ctx context.Context, path string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.server+path, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "text/markdown, text/plain")
	req.Header.Set("User-Agent", c.agent)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("no se pudo conectar con %s: %w", c.server, unwrapURLError(err))
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("leer respuesta de %s: %w", c.server, err)
	}
	if resp.StatusCode >= 300 {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(raw, &e)
		return "", &apiError{Status: resp.StatusCode, Message: e.Error}
	}
	return string(raw), nil
}

func unwrapURLError(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Err
	}
	return err
}

// loginStart es la respuesta de POST /api/cli/login.
type loginStart struct {
	Code       string    `json:"code"`
	PollSecret string    `json:"poll_secret"`
	URL        string    `json:"url"`
	ExpiresAt  time.Time `json:"expires_at"`
	Interval   int       `json:"interval"`
}

// loginPoll es la respuesta de POST /api/cli/login/poll.
type loginPoll struct {
	Status   string `json:"status"`
	Token    string `json:"token"`
	TokenID  string `json:"token_id"`
	Username string `json:"username"`
}

// me es la respuesta de GET /api/auth/me.
type me struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	Email    string `json:"email"`
}

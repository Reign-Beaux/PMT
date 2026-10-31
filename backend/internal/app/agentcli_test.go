package app

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"pmt/internal/features/agents"
	"pmt/internal/features/agents/commands/create"
	"pmt/internal/features/agents/domain"
)

type fakeAgents struct{}

func (fakeAgents) Create(_ context.Context, agent domain.Agent, _ []byte) (domain.Agent, error) {
	return agent, nil
}

func TestAgentCommandUsage(t *testing.T) {
	for _, args := range [][]string{nil, {"borrar"}, {"create", "sobra"}, {"create", "--otra", "x"}} {
		err := runAgentCommand(context.Background(), agents.Admin{}, args, &bytes.Buffer{})
		if err == nil || !strings.HasPrefix(err.Error(), "uso:") {
			t.Errorf("%v: error = %v", args, err)
		}
	}
}

func TestAgentCreateWritesOnlyTheToken(t *testing.T) {
	var out bytes.Buffer
	admin := agents.Admin{Create: create.New(fakeAgents{})}

	err := runAgentCommand(context.Background(), admin, []string{"create", "--name", "dev", "--profile", "DevBackend"}, &out)

	if err != nil {
		t.Fatal(err)
	}
	if token := strings.TrimSpace(out.String()); len(token) != 64 || strings.ContainsAny(token, " \n") {
		t.Errorf("salida de %d caracteres", len(token))
	}
}

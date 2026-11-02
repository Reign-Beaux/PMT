package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"text/tabwriter"

	"pmt/internal/features/agents"
	"pmt/internal/features/agents/commands/create"
	"pmt/internal/kernel"
	"pmt/internal/platform/postgres"
)

const agentUsage = `uso:
  pmt-server agent create --name NOMBRE --profile PERFIL
  pmt-server agent rotate --name NOMBRE
  pmt-server agent revoke --name NOMBRE
  pmt-server agent list

create y rotate escriben solo el token en la salida estándar. No se puede volver a obtener.`

// RunAgent administra los agentes. Lee la configuración sin escribirla.
func RunAgent(args []string, stdout io.Writer) error {
	cfg, err := LoadConfig("config.yaml")
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()
	pool, err := postgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	return runAgentCommand(ctx, agents.NewAdmin(pool), args, stdout)
}

func runAgentCommand(ctx context.Context, admin agents.Admin, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New(agentUsage)
	}

	flags := flag.NewFlagSet("agent "+args[0], flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	name := flags.String("name", "", "nombre del agente")
	profile := flags.String("profile", "", "perfil del agente")
	if err := flags.Parse(args[1:]); err != nil || flags.NArg() > 0 {
		return errors.New(agentUsage)
	}

	switch args[0] {
	case "create":
		result, err := admin.Create.Handle(ctx, create.Command{Name: *name, Profile: kernel.Profile(*profile)})
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, result.Token)
		return err
	case "rotate":
		token, err := admin.Rotate.Handle(ctx, *name)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, token)
		return err
	case "revoke":
		return admin.Revoke.Handle(ctx, *name)
	case "list":
		items, err := admin.List.Handle(ctx)
		if err != nil {
			return err
		}
		table := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(table, "NOMBRE\tPERFIL\tREVOCADO\tCREADO")
		for _, item := range items {
			fmt.Fprintf(table, "%s\t%s\t%t\t%s\n", item.Name, item.Profile, item.Revoked, item.CreatedAt.Format("2006-01-02 15:04"))
		}
		return table.Flush()
	}
	return errors.New(agentUsage)
}

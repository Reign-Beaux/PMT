package cli

import (
	"bufio"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
)

// errInterrupted corta una pregunta por Ctrl+C (contexto cancelado).
var errInterrupted = errors.New("cancelado con Ctrl+C: no guardé nada")

// errNoInput corta una pregunta porque la entrada se acabó (Ctrl+D).
var errNoInput = errors.New("cancelado: se acabó la entrada (Ctrl+D) sin respuesta; no guardé nada")

// prompter lee las respuestas a las preguntas línea por línea sin bloquear al
// que pregunta: la lectura de stdin va en una goroutine, y cada pregunta espera
// la línea o el fin del contexto, lo que llegue primero. Así Ctrl+C corta
// aunque stdin siga esperando.
type prompter struct {
	ctx   context.Context
	r     io.Reader
	once  sync.Once
	lines chan lineResult
}

type lineResult struct {
	line string
	err  error
}

func newPrompter(ctx context.Context, r io.Reader) *prompter {
	return &prompter{ctx: ctx, r: r, lines: make(chan lineResult)}
}

// start arranca la lectura con la primera pregunta: si no se pregunta nada,
// stdin no se toca.
func (p *prompter) start() {
	go func() {
		in := bufio.NewReader(p.r)
		for {
			line, err := in.ReadString('\n')
			if line != "" {
				select {
				case p.lines <- lineResult{line: line}:
				case <-p.ctx.Done():
					return
				}
			}
			if err != nil {
				select {
				case p.lines <- lineResult{err: err}:
				case <-p.ctx.Done():
				}
				return
			}
		}
	}()
}

// readLine devuelve la siguiente respuesta sin espacios alrededor. Con el
// contexto cancelado devuelve errInterrupted; sin más entrada, errNoInput.
func (p *prompter) readLine() (string, error) {
	if p.ctx.Err() != nil {
		return "", errInterrupted
	}
	p.once.Do(p.start)
	select {
	case <-p.ctx.Done():
		return "", errInterrupted
	case r, ok := <-p.lines:
		if !ok || r.err != nil {
			// La goroutine ya terminó: las siguientes preguntas también cortan.
			p.lines = closedLines
			if r.err != nil && !errors.Is(r.err, io.EOF) {
				return "", r.err
			}
			return "", errNoInput
		}
		return strings.TrimSpace(r.line), nil
	}
}

// closedLines es un canal cerrado: tras el fin de la entrada, toda lectura corta.
var closedLines = func() chan lineResult {
	c := make(chan lineResult)
	close(c)
	return c
}()

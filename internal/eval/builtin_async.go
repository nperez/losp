// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez

package eval

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"nickandperla.net/losp/internal/expr"
)

func builtinAsync(e *Evaluator, argsRaw string) (expr.Expr, error) {
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return expr.Empty{}, nil
	}
	if len(args) < 1 {
		return expr.Empty{}, nil
	}

	name := args[0]

	// Verify expression exists
	stored := e.namespace.Get(name)
	if stored.IsEmpty() {
		return expr.Empty{}, nil
	}

	h := e.asyncRegistry.Register(false, 0)
	forked := e.forkForAsync()

	e.asyncRegistry.wg.Add(1)
	go func() {
		defer e.asyncRegistry.wg.Done()
		defer close(h.done)
		result, err := forked.execute(name, "")
		if err != nil {
			h.err = err
			return
		}
		h.result = strings.TrimSpace(result.String())
	}()

	return expr.Text{Value: h.id}, nil
}

func builtinAwait(e *Evaluator, argsRaw string) (expr.Expr, error) {
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return expr.Empty{}, nil
	}
	if len(args) < 1 {
		return expr.Empty{}, nil
	}

	id := args[0]
	h := e.asyncRegistry.Get(id)
	if h == nil {
		return expr.Empty{}, nil
	}

	<-h.done

	if h.err != nil || h.result == "" {
		return expr.Empty{}, nil
	}
	return expr.Text{Value: h.result}, nil
}

func builtinCheck(e *Evaluator, argsRaw string) (expr.Expr, error) {
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return expr.Text{Value: "FALSE"}, nil
	}
	if len(args) < 1 {
		return expr.Text{Value: "FALSE"}, nil
	}

	id := args[0]
	h := e.asyncRegistry.Get(id)
	if h == nil {
		return expr.Text{Value: "FALSE"}, nil
	}

	select {
	case <-h.done:
		return expr.Text{Value: "TRUE"}, nil
	default:
		return expr.Text{Value: "FALSE"}, nil
	}
}

func builtinTimer(e *Evaluator, argsRaw string) (expr.Expr, error) {
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return expr.Empty{}, nil
	}
	if len(args) < 2 {
		return expr.Empty{}, nil
	}

	ms, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return expr.Empty{}, nil
	}
	name := args[1]

	// Verify expression exists
	stored := e.namespace.Get(name)
	if stored.IsEmpty() {
		return expr.Empty{}, nil
	}

	duration := time.Duration(ms) * time.Millisecond
	h := e.asyncRegistry.Register(true, duration)
	forked := e.forkForAsync()

	e.asyncRegistry.wg.Add(1)
	h.timer = time.AfterFunc(duration, func() {
		defer e.asyncRegistry.wg.Done()
		defer close(h.done)
		result, err := forked.execute(name, "")
		if err != nil {
			h.err = err
			return
		}
		h.result = strings.TrimSpace(result.String())
	})

	return expr.Text{Value: h.id}, nil
}

func builtinTicks(e *Evaluator, argsRaw string) (expr.Expr, error) {
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return expr.Text{Value: "0"}, nil
	}
	if len(args) < 1 {
		return expr.Text{Value: "0"}, nil
	}

	id := args[0]
	h := e.asyncRegistry.Get(id)
	if h == nil {
		return expr.Text{Value: "0"}, nil
	}

	// Non-timer or already completed: return 0
	if !h.isTimer {
		return expr.Text{Value: "0"}, nil
	}

	select {
	case <-h.done:
		return expr.Text{Value: "0"}, nil
	default:
	}

	remaining := time.Until(h.fireAt)
	if remaining < 0 {
		remaining = 0
	}
	return expr.Text{Value: fmt.Sprintf("%d", remaining.Milliseconds())}, nil
}

func builtinSleep(e *Evaluator, argsRaw string) (expr.Expr, error) {
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return expr.Empty{}, nil
	}
	if len(args) < 1 {
		return expr.Empty{}, nil
	}

	ms, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return expr.Empty{}, nil
	}

	time.Sleep(time.Duration(ms) * time.Millisecond)
	return expr.Empty{}, nil
}

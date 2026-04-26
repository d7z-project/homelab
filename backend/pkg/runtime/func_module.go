package runtime

import (
	"context"

	"github.com/go-chi/chi/v5"
)

type FuncModule struct {
	ModuleName string
	OnInit     func(deps ModuleDeps) error
	Routes     func(r chi.Router)
	OnStart    func(ctx context.Context) error
	OnStop     func(ctx context.Context) error
}

func (m FuncModule) Name() string {
	return m.ModuleName
}

func (m FuncModule) Init(deps ModuleDeps) error {
	if m.OnInit != nil {
		return m.OnInit(deps)
	}
	return nil
}

func (m FuncModule) RegisterRoutes(r chi.Router) {
	if m.Routes != nil {
		m.Routes(r)
	}
}

func (m FuncModule) Start(ctx context.Context) error {
	if m.OnStart != nil {
		return m.OnStart(ctx)
	}
	return nil
}

func (m FuncModule) Stop(ctx context.Context) error {
	if m.OnStop != nil {
		return m.OnStop(ctx)
	}
	return nil
}

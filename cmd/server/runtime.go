package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/application"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/chaincheck"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/httpapi"
	"benzhi-project-f76842ad-2ac3-482b-8891-76d6c790b0dd/internal/persistence"
)

type runtime struct {
	store       *persistence.Store
	server      *http.Server
	listener    net.Listener
	serveResult chan error
}

func newRuntime(cfg config) (*runtime, error) {
	store, err := persistence.Open(cfg.databasePath)
	if err != nil {
		return nil, err
	}
	service := application.New(store, chaincheck.New())
	api := httpapi.New(service)
	listener, err := net.Listen("tcp", cfg.address)
	if err != nil {
		store.Close()
		return nil, fmt.Errorf("监听 %s: %w", cfg.address, err)
	}
	server := &http.Server{Handler: api.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	return &runtime{store: store, server: server, listener: listener, serveResult: make(chan error, 1)}, nil
}

func (r *runtime) start() {
	go func() {
		err := r.server.Serve(r.listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		r.serveResult <- err
	}()
}

func (r *runtime) shutdown(ctx context.Context) error {
	serverErr := r.server.Shutdown(ctx)
	if serverErr != nil {
		_ = r.server.Close()
	}
	serveErr := <-r.serveResult
	storeErr := r.store.Close()
	if serverErr != nil {
		return serverErr
	}
	if serveErr != nil {
		return serveErr
	}
	return storeErr
}

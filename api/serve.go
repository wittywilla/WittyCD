// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package api

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/wittywilla/wittymodule"
	"github.com/wittywilla/wittymodule/host"
	"github.com/wittywilla/wittymodule/sdui"
	"github.com/wittywilla/wittymodule/service"
)

// Dual is one process answering as either of the two things a server starts wittycd-module as:
//
//   - a helper with one open image, as it always was (Helper), started for each disc played,
//     and never given a network;
//   - WittyCD's service (module API 5): the server's first call is then service.MethodStart,
//     with where its host API is. It listens on this machine only, with a secret of its own,
//     for the routes the server passes on, and answers the SDUI and lifecycle calls.
//
// wittymodule/service.Main runs a service only, and answers every other call with
// "unknown-method", so it can't also be the helper a server asks about one image: this is that
// part of it, with the same guard (a WittyModule gap: a service with helper calls of its own).
type Dual struct {
	Helper wittymodule.Handler

	mu      sync.Mutex
	started *service.Started
	host    *host.Client
}

// Handle is the helper protocol's handler, for wittymodule.Serve.
func (d *Dual) Handle(method string, params json.RawMessage) (any, []byte, error) {
	switch {
	case method == service.MethodStart:
		return d.start(params)
	case strings.HasPrefix(method, "lifecycle."):
		return nil, nil, nil // nothing to set up: WittyCD keeps no data of its own
	}
	answers := sdui.Answers{Items: func(req sdui.ItemsRequest) (sdui.ItemsAnswer, error) {
		h := d.hostClient()
		if h == nil {
			return sdui.ItemsAnswer{}, nil
		}
		return Items(h, req)
	}}
	if res, blob, err, ok := answers.Handle(method, params); ok {
		return res, blob, err
	}
	return d.Helper(method, params)
}

func (d *Dual) hostClient() *host.Client {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.host
}

// start says where the service is, listening the first time. The server asks again now and
// then, with nothing, to see the process is there: what it said at the start stands.
func (d *Dual) start(params json.RawMessage) (any, []byte, error) {
	var p service.StartParams
	if len(params) > 0 && string(params) != "null" {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, nil, err
		}
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if p.Host != "" {
		d.host = host.NewAt(p.Host, p.HostToken, p.Offers)
	}
	if d.started == nil {
		st, err := d.listen()
		if err != nil {
			return nil, nil, err
		}
		d.started = st
	}
	return d.started, nil, nil
}

// listen serves the routes on this machine only, behind the start's secret.
func (d *Dual) listen() (*service.Started, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(secret)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	srv := &http.Server{Handler: Guard(token, d.Mux()), ReadHeaderTimeout: 30 * time.Second}
	go func() { _ = srv.Serve(l) }()
	return &service.Started{URL: "http://" + l.Addr().String(), Token: token}, nil
}

// Mux is the service's handler: WittyCD's routes under /http, as the server passes them.
func (d *Dual) Mux() http.Handler {
	own := http.NewServeMux()
	Routes(own, hostLater{d})
	mux := http.NewServeMux()
	mux.Handle("/http/", http.StripPrefix("/http", own))
	return mux
}

// hostLater is the host API as the server last said where it is.
type hostLater struct{ d *Dual }

func (h hostLater) Do(ctx context.Context, on host.On, method, path string, body, out any) error {
	c := h.d.hostClient()
	if c == nil {
		return host.ErrNoHost
	}
	return c.Do(ctx, on, method, path, body, out)
}

// Guard lets in only whoever was told the start's secret, which is the server: a port on this
// machine can be reached by anything else running on it.
func Guard(token string, next http.Handler) http.Handler {
	want := []byte(token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get(service.TokenHeader)), want) != 1 {
			http.Error(w, "no", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

// Package api is WittyCD's own HTTP API and SDUI source (module API 5): finding which album an
// audio CD is. WittyMediaServer passes an administrator's request for /api/modules/wittycd/...
// here once it has checked who is asking and that they may see the disc (integration.json).
//
// The lookups themselves are the server's, through its host API: the process that reads disc
// images never has a network, so the server asks MusicBrainz with the disc's table of contents
// and fetches the cover, as its library scan does. This is the route a person (or a TV's page)
// reaches, and what it shows.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/wittywilla/wittycd"
	"github.com/wittywilla/wittymodule/host"
	"github.com/wittywilla/wittymodule/sdui"
	core "github.com/wittywilla/wittymodule/sdui/core"
	"github.com/wittywilla/wittymodule/service"
)

// Host is the part of the host API this uses (host.Client).
type Host interface {
	Do(ctx context.Context, on host.On, method, path string, body, out any) error
}

// Routes adds WittyCD's routes, written without the /http prefix the server passes them under.
func Routes(mux *http.ServeMux, h Host) {
	mux.Handle("GET /cds/{id}/matches", service.Admins(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := discID(w, r)
		if !ok {
			return
		}
		var out json.RawMessage
		if err := h.Do(r.Context(), host.For(r), "GET", fmt.Sprintf("/cds/%d/matches", id), nil, &out); err != nil {
			writeHostError(w, err)
			return
		}
		writeRaw(w, out)
	})))
	mux.Handle("POST /cds/{id}/identify", service.Admins(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := discID(w, r)
		if !ok {
			return
		}
		var b struct {
			ReleaseID string `json:"releaseId"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&b); err != nil {
			writeError(w, 400, "expected a JSON body with releaseId")
			return
		}
		var out json.RawMessage
		if err := h.Do(r.Context(), host.For(r), "PUT", fmt.Sprintf("/cds/%d/release", id), map[string]string{"releaseId": b.ReleaseID}, &out); err != nil {
			writeHostError(w, err)
			return
		}
		writeRaw(w, out)
	})))
}

func discID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, 404, "no such disc")
		return 0, false
	}
	return id, true
}

func writeRaw(w http.ResponseWriter, b json.RawMessage) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(b)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// writeHostError passes on what the server said, with its status: it words its refusals for
// whoever asked.
func writeHostError(w http.ResponseWriter, err error) {
	var he *host.Error
	if errors.As(err, &he) {
		writeError(w, he.Status, he.Message)
		return
	}
	if errors.Is(err, host.ErrNotOffered) || errors.Is(err, host.ErrNoHost) {
		writeError(w, 501, "This server can't find out which album a disc is for WittyCD: it needs a newer server.")
		return
	}
	writeError(w, 502, "The server couldn't be asked just now.")
}

// match is a MusicBrainz release as the server sends it.
type match struct {
	ReleaseID string `json:"releaseId"`
	Title     string `json:"title"`
	Artist    string `json:"artist"`
	Date      string `json:"date"`
	Country   string `json:"country"`
	Label     string `json:"label"`
	Disc      int    `json:"disc"`
	Discs     int    `json:"discs"`
	Exact     bool   `json:"exact"`
}

// Items answers the server's sdui.items call for WittyCD's source.
func Items(h Host, req sdui.ItemsRequest) (sdui.ItemsAnswer, error) {
	if req.Source.Type() != wittycd.SourceMatches || !req.Viewer.Admin || req.Viewer.User <= 0 || h == nil {
		return sdui.ItemsAnswer{}, nil
	}
	id, err := strconv.ParseInt(fmt.Sprint(req.Source["item"]), 10, 64)
	if err != nil || id <= 0 {
		return sdui.ItemsAnswer{}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	var got struct {
		Matches []match `json:"matches"`
	}
	if err := h.Do(ctx, host.On{User: req.Viewer.User}, "GET", fmt.Sprintf("/cds/%d/matches", id), nil, &got); err != nil {
		var he *host.Error
		if errors.As(err, &he) && (he.Status == 403 || he.Status == 404) {
			return sdui.ItemsAnswer{}, nil
		}
		return sdui.ItemsAnswer{}, err
	}
	item := strconv.FormatInt(id, 10)
	var rows []sdui.Item
	for _, m := range got.Matches {
		disc := ""
		if m.Discs > 1 {
			disc = fmt.Sprintf("disc %d of %d", m.Disc, m.Discs)
		}
		row := sdui.Item{ID: "release-" + m.ReleaseID, Kind: "view", Title: m.Title, Icon: "disc",
			Subtitle: words(m.Artist, year(m.Date), m.Country, m.Label, disc),
			Action:   &sdui.Action{Type: core.ActCommand, Op: wittycd.CommandIdentify, Item: item, Params: map[string]any{"release": m.ReleaseID}}}
		if m.Exact {
			row.Badges = []string{"Exact"}
		}
		rows = append(rows, row)
	}
	if len(rows) > 0 {
		rows = append(rows, sdui.Item{ID: "own", Kind: "view", Title: "Use the disc's own names", Icon: "disc",
			Subtitle: "What the disc and its sheet say, from the next scan.",
			Action:   &sdui.Action{Type: core.ActCommand, Op: wittycd.CommandIdentify, Item: item, Params: map[string]any{"release": ""}}})
	}
	if req.Offset > 0 {
		if req.Offset >= len(rows) {
			rows = nil
		} else {
			rows = rows[req.Offset:]
		}
	}
	more := false
	if req.Limit > 0 && len(rows) > req.Limit {
		rows, more = rows[:req.Limit], true
	}
	return sdui.ItemsAnswer{Items: rows, More: more}, nil
}

func year(date string) string {
	if len(date) >= 4 {
		return date[:4]
	}
	return ""
}

func words(parts ...string) string {
	out := parts[:0:0]
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, " · ")
}

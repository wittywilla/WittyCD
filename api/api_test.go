// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wittywilla/wittycd"
	"github.com/wittywilla/wittymodule/host"
	"github.com/wittywilla/wittymodule/sdui"
	"github.com/wittywilla/wittymodule/service"
)

// fakeHost is the server's host API for a disc, 5: matches for anyone it calls an
// administrator (user 1), and the disc's view after a release is chosen.
func fakeHost(t *testing.T, asked *[]string) (*host.Client, string) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Witty-Host", "1")
		if r.Header.Get(host.HeaderToken) != "tok" {
			w.WriteHeader(403)
			return
		}
		b, _ := io.ReadAll(r.Body)
		*asked = append(*asked, r.Method+" "+r.URL.Path+" "+r.Header.Get(host.HeaderOnUser)+" "+string(b))
		if r.Header.Get(host.HeaderOnUser) != "1" {
			w.WriteHeader(403)
			_, _ = w.Write([]byte(`{"error":"Only an administrator can do that."}`))
			return
		}
		switch r.Method + " " + r.URL.Path {
		case "GET /host/v1/cds/5/matches":
			_, _ = w.Write([]byte(`{"itemId":5,"discId":"x","matches":[{"releaseId":"r1","title":"Album","artist":"Band","date":"1999-03-02","discs":2,"disc":1,"exact":true}],"attribution":"MusicBrainz"}`))
		case "PUT /host/v1/cds/5/release":
			_, _ = w.Write([]byte(`{"id":5,"title":"Album"}`))
		default:
			w.WriteHeader(404)
			_, _ = w.Write([]byte(`{"error":"no such disc"}`))
		}
	}))
	t.Cleanup(srv.Close)
	return host.NewAt(srv.URL, "tok", nil), srv.URL
}

func call(t *testing.T, h http.Handler, user string, admin bool, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if user != "" {
		r.Header.Set(service.HeaderUser, user)
	}
	if admin {
		r.Header.Set(service.HeaderAdmin, "1")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestRoutes(t *testing.T) {
	var asked []string
	mux := http.NewServeMux()
	h, _ := fakeHost(t, &asked)
	Routes(mux, h)

	w := call(t, mux, "1", true, "GET", "/cds/5/matches", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"releaseId":"r1"`) {
		t.Fatalf("matches: %d %s", w.Code, w.Body)
	}
	w = call(t, mux, "1", true, "POST", "/cds/5/identify", `{"releaseId":"r1"}`)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"title":"Album"`) {
		t.Fatalf("identify: %d %s", w.Code, w.Body)
	}
	if want := `PUT /host/v1/cds/5/release 1 {"releaseId":"r1"}`; asked[len(asked)-1] != want {
		t.Errorf("asked the host %q, want %q", asked[len(asked)-1], want)
	}
	// The server's refusals pass on as they are.
	if w := call(t, mux, "1", true, "GET", "/cds/9/matches", ""); w.Code != 404 || !strings.Contains(w.Body.String(), "no such disc") {
		t.Errorf("another disc: %d %s", w.Code, w.Body)
	}
	if w := call(t, mux, "2", true, "GET", "/cds/5/matches", ""); w.Code != 403 {
		t.Errorf("someone the server says no to: %d", w.Code)
	}
	// Checked here too: only an administrator, and a body that says which release.
	if w := call(t, mux, "1", false, "GET", "/cds/5/matches", ""); w.Code != 403 {
		t.Errorf("not an administrator: %d", w.Code)
	}
	if w := call(t, mux, "1", true, "POST", "/cds/5/identify", `nonsense`); w.Code != 400 {
		t.Errorf("an unreadable body: %d", w.Code)
	}
	if w := call(t, mux, "1", true, "GET", "/cds/x/matches", ""); w.Code != 404 {
		t.Errorf("not an id: %d", w.Code)
	}
}

func TestItems(t *testing.T) {
	var asked []string
	h, _ := fakeHost(t, &asked)
	req := sdui.ItemsRequest{Source: sdui.Source{"type": wittycd.SourceMatches, "item": "5"}, Limit: 50, Viewer: sdui.Viewer{User: 1, Admin: true}}
	ans, err := Items(h, req)
	if err != nil || len(ans.Items) != 2 {
		t.Fatalf("items: %+v %v", ans, err)
	}
	m := ans.Items[0]
	if m.Title != "Album" || m.Subtitle != "Band · 1999 · disc 1 of 2" || m.Action == nil || m.Action.Op != wittycd.CommandIdentify || m.Action.Item != "5" || m.Action.Params["release"] != "r1" {
		t.Errorf("a match: %+v %+v", m, m.Action)
	}
	if own := ans.Items[1]; own.Action == nil || own.Action.Params["release"] != "" {
		t.Errorf("the disc's own names: %+v", own)
	}
	// Nobody but an administrator, and nothing for another source.
	req.Viewer.Admin = false
	if ans, _ := Items(h, req); len(ans.Items) != 0 {
		t.Errorf("for someone not an administrator: %+v", ans)
	}
	req.Viewer = sdui.Viewer{User: 2, Admin: true}
	if ans, err := Items(h, req); len(ans.Items) != 0 || err != nil {
		t.Errorf("the server refused: %+v %v", ans, err)
	}
}

// The one program is a helper for one image, and WittyCD's service when started as one.
func TestDual(t *testing.T) {
	var helped []string
	d := &Dual{Helper: func(method string, _ json.RawMessage) (any, []byte, error) {
		helped = append(helped, method)
		return "image", nil, nil
	}}
	if res, _, err := d.Handle("open", json.RawMessage(`{"path":"x.cue"}`)); err != nil || res != "image" || len(helped) != 1 {
		t.Fatalf("a helper call: %v %v %v", res, err, helped)
	}
	var asked []string
	_, base := fakeHost(t, &asked)
	params, _ := json.Marshal(service.StartParams{Host: base, HostToken: "tok"})
	res, _, err := d.Handle(service.MethodStart, params)
	st, ok := res.(*service.Started)
	if err != nil || !ok || !strings.HasPrefix(st.URL, "http://127.0.0.1:") || st.Token == "" {
		t.Fatalf("starting: %+v %v", res, err)
	}
	// Asked again with nothing, it is the same service.
	if again, _, _ := d.Handle(service.MethodStart, nil); again.(*service.Started) != st {
		t.Error("a second start made another service")
	}
	if _, _, err := d.Handle("lifecycle.install", json.RawMessage(`{"to":"0.3.0"}`)); err != nil {
		t.Errorf("install: %v", err)
	}
	// Its routes, behind the start's secret.
	get := func(tok string) int {
		r, _ := http.NewRequest("GET", st.URL+"/http/cds/5/matches", nil)
		r.Header.Set(service.TokenHeader, tok)
		r.Header.Set(service.HeaderUser, "1")
		r.Header.Set(service.HeaderAdmin, "1")
		resp, err := http.DefaultClient.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	if c := get("forged"); c != 403 {
		t.Errorf("a forged secret: %d", c)
	}
	if c := get(st.Token); c != 200 {
		t.Errorf("the server: %d", c)
	}
	// Its source, through the host it was told of.
	b, _ := json.Marshal(sdui.ItemsRequest{Source: sdui.Source{"type": wittycd.SourceMatches, "item": 5}, Limit: 10, Viewer: sdui.Viewer{User: 1, Admin: true}})
	ans, _, err := d.Handle(sdui.MethodItems, b)
	if a, ok := ans.(sdui.ItemsAnswer); err != nil || !ok || len(a.Items) != 2 {
		t.Errorf("sdui.items: %+v %v", ans, err)
	}
	if len(helped) != 1 {
		t.Errorf("the service's calls reached the image helper: %v", helped)
	}
}

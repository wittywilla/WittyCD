// Command wittycd-module is WittyCD as a helper process: the only way a server reaches it. One
// process has one open image, and answers one call at a time. Samples go a chunk at a time,
// and FLAC is encoded here, so whatever an image holds meets nothing but this process's limits.
// See WittyModule for the protocol.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"sync"

	"github.com/wittywilla/wittycd"
	"github.com/wittywilla/wittymodule"
)

func main() {
	h := &helper{streams: map[int]*wittycd.Stream{}, flacs: map[int]*flac{}}
	wittymodule.Main(wittycd.ManifestJSON, h.handle)
}

// maxRead is more than a server asks for at once.
const maxRead = 8 << 20

// flac is one program being encoded, read through a pipe.
type flac struct {
	r *io.PipeReader
}

type helper struct {
	mu      sync.Mutex
	disc    *wittycd.Disc
	streams map[int]*wittycd.Stream
	flacs   map[int]*flac
	next    int
}

// opened is the answer to "open": what a disc says about itself, with what the server would
// otherwise have to work out.
type opened struct {
	*wittycd.Disc
	TotalBytes  int64       `json:"totalBytes"`
	DurationSec float64     `json:"durationSec"`
	TOC         wittycd.TOC `json:"toc"`
}

func (h *helper) handle(method string, raw json.RawMessage) (any, []byte, error) {
	var p struct {
		Path       string            `json:"path"`
		Foreign    map[string]string `json:"foreign"`
		Tracks     []int             `json:"tracks"`
		Whole      bool              `json:"whole"`
		Loops      int               `json:"loops"`
		Deemphasis bool              `json:"deemphasis"`
		ID         int               `json:"id"`
		Off        int64             `json:"off"`
		N          int               `json:"n"`
		Start      int64             `json:"start"`
		From       int               `json:"from"`
		Sectors    int               `json:"sectors"`
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, nil, &wittymodule.CallError{Code: "bad-call", Message: "the call is unreadable"}
		}
	}
	if method == "open" {
		// A sound file this package can't read is made into FLAC by the server, which has FFmpeg.
		// This asks for what's missing, and is asked again with it.
		var missing []string
		wittycd.ForeignAudio = func(path string) (string, error) {
			if f, ok := p.Foreign[path]; ok {
				return f, nil
			}
			missing = append(missing, path)
			return "", errors.New("the server hasn't made this yet")
		}
		d, err := wittycd.Open(p.Path)
		if len(missing) > 0 {
			return nil, nil, &wittymodule.CallError{Code: "needs-flac", Message: strings.Join(missing, "\n")}
		}
		if err != nil {
			return nil, nil, coded(err)
		}
		h.disc = d
		return opened{Disc: d, TotalBytes: d.TotalBytes(), DurationSec: d.DurationSec(), TOC: d.TOC()}, nil, nil
	}
	if h.disc == nil {
		return nil, nil, &wittymodule.CallError{Code: "bad-call", Message: "no image is open"}
	}
	switch method {
	case "diagnose":
		return h.diagnose()
	case "stream":
		tracks := p.Tracks
		if p.Whole {
			tracks = nil
		}
		st, err := h.disc.NewStream(tracks, p.Loops, p.Deemphasis)
		if err != nil {
			return nil, nil, coded(err)
		}
		h.mu.Lock()
		defer h.mu.Unlock()
		h.next++
		h.streams[h.next] = st
		return struct {
			ID      int   `json:"id"`
			Samples int64 `json:"samples"`
			Size    int64 `json:"size"`
		}{h.next, st.Samples(), st.Size()}, nil, nil
	case "read":
		st := h.stream(p.ID)
		if st == nil || p.N < 0 || p.N > maxRead {
			return nil, nil, &wittymodule.CallError{Code: "bad-call", Message: "no such program"}
		}
		buf := make([]byte, p.N)
		n, err := st.ReadAt(buf, p.Off)
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, nil, coded(err)
		}
		return nil, buf[:n], nil
	case "flac":
		st := h.stream(p.ID)
		if st == nil {
			return nil, nil, &wittymodule.CallError{Code: "bad-call", Message: "no such program"}
		}
		r, w := io.Pipe()
		go func() { _ = w.CloseWithError(st.WriteFLAC(w, p.Start)) }()
		h.mu.Lock()
		defer h.mu.Unlock()
		h.next++
		h.flacs[h.next] = &flac{r: r}
		return struct {
			ID int `json:"id"`
		}{h.next}, nil, nil
	case "flacnext":
		h.mu.Lock()
		f := h.flacs[p.ID]
		h.mu.Unlock()
		if f == nil {
			return nil, nil, nil
		}
		buf := make([]byte, 256<<10)
		n, err := io.ReadFull(f.r, buf)
		if err != nil {
			h.dropFlac(p.ID)
			if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
				return nil, nil, coded(err)
			}
		}
		return nil, buf[:n], nil
	case "flacstop":
		h.dropFlac(p.ID)
		return nil, nil, nil
	case "graphics":
		b, err := h.disc.ReadGraphics(p.From, p.Sectors)
		if err != nil {
			return nil, nil, coded(err)
		}
		return nil, b, nil
	}
	return nil, nil, &wittymodule.CallError{Code: "unknown-method", Message: fmt.Sprintf("this version has no %q", method)}
}

// diagnose reads every track, reporting as it goes.
func (h *helper) diagnose() (any, []byte, error) {
	j := &wittymodule.DiagJob{}
	d := h.disc
	j.SetLabel("Audio CD")
	j.Step("Reading every track", 0.05)
	r := d.Check(func(done float64) bool {
		j.Progress(0.05 + done*0.9)
		return true
	})
	j.Add(wittymodule.DiagFinding{Severity: wittymodule.DiagOK, Title: "This is an audio CD image",
		What: fmt.Sprintf("%d audio tracks, %s long, in %s form. It belongs in a Music library.", len(d.AudioTracks()), hms(r.Facts.DurationSec), d.Format)})
	for _, f := range r.Findings {
		sev := wittymodule.DiagOK
		switch f.Severity {
		case wittycd.SeverityNote:
			sev = wittymodule.DiagInfo
		case wittycd.SeverityProblem:
			sev = wittymodule.DiagProblem
		}
		title := "The disc"
		if f.Track > 0 {
			title = fmt.Sprintf("Track %d", f.Track)
		}
		j.Add(wittymodule.DiagFinding{Severity: sev, Title: title, What: f.Text})
	}
	return nil, nil, nil
}

func hms(sec float64) string {
	s := int(sec + 0.5)
	return fmt.Sprintf("%d:%02d:%02d", s/3600, s/60%60, s%60)
}

func (h *helper) stream(id int) *wittycd.Stream {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.streams[id]
}

func (h *helper) dropFlac(id int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if f := h.flacs[id]; f != nil {
		_ = f.r.Close()
		delete(h.flacs, id)
	}
}

// coded names the errors the server acts on.
func coded(err error) error {
	code := ""
	switch {
	case errors.Is(err, wittycd.ErrNotCD):
		code = "not-cd"
	case errors.Is(err, wittycd.ErrNoAudio):
		code = "no-audio"
	case errors.Is(err, wittycd.ErrUnsupported):
		code = "unsupported"
	case errors.Is(err, fs.ErrPermission):
		code = "denied"
	}
	return &wittymodule.CallError{Code: code, Message: err.Error()}
}

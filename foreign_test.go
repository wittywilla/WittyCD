// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package wittycd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSheetOverAForeignFile(t *testing.T) {
	dir := t.TempDir()
	sheet := filepath.Join(dir, "album.cue")
	audio := filepath.Join(dir, "album.ape")
	if err := os.WriteFile(audio, []byte("MAC not really"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sheet, []byte("FILE \"album.ape\" WAVE\n  TRACK 01 AUDIO\n    INDEX 01 00:00:00\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	defer func() { ForeignAudio = nil }()

	ForeignAudio = nil
	if _, image := SheetFiles(sheet); image {
		t.Error("with nothing to read it, the sheet isn't an image")
	}
	if _, err := Open(sheet); !errors.Is(err, ErrUnsupported) {
		t.Errorf("with nothing to read it: %v", err)
	}

	asked := ""
	ForeignAudio = func(p string) (string, error) { asked = p; return "", errors.New("can't") }
	files, image := SheetFiles(sheet)
	if !image || len(files) != 1 || files[0] != audio {
		t.Errorf("with a reader it is one: %v %v", files, image)
	}
	if _, err := Open(sheet); !errors.Is(err, ErrUnsupported) || asked != audio {
		t.Errorf("a reader that fails: %v, asked %q", err, asked)
	}

	// Only FLAC comes back: nothing else is trusted to be CD audio.
	ForeignAudio = func(string) (string, error) { return audio, nil }
	if _, err := Open(sheet); !errors.Is(err, ErrUnsupported) {
		t.Errorf("an answer that isn't FLAC: %v", err)
	}
}

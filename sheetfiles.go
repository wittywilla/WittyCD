// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package wittycd

import (
	"path/filepath"
	"regexp"
	"strings"
)

var (
	cueFileLine = regexp.MustCompile(`(?i)^\s*FILE\s+"?(.+?)"?\s+[A-Z0-9]+\s*$`)
	tocFileLine = regexp.MustCompile(`(?i)^\s*(?:AUDIOFILE|DATAFILE|FILE)\s+"([^"]+)"`)
)

// SheetFiles lists the files a sheet plays, resolved beside it, without opening
// any of them, and says whether the sheet is really a disc image. It is cheap
// enough for a scan to ask about every sheet it finds.
//
// A sheet is not treated as an image when its audio is already a set of separately
// compressed files, one to a track: those are tagged songs, and the sheet is just
// a rip's notes. The same goes for a sheet naming compressed audio WittyCD can't
// read. A .ccd, .nrg or .chd holds its own audio, and is always an image.
func SheetFiles(path string) (files []string, image bool) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".cue":
	case ".toc":
	default:
		return nil, IsImageName(path)
	}
	raw, err := readSheet(path)
	if err != nil {
		return nil, false
	}
	line := cueFileLine
	if strings.EqualFold(filepath.Ext(path), ".toc") {
		line = tocFileLine
	}
	dir := filepath.Dir(path)
	seen := map[string]bool{}
	compressed, named := 0, 0
	for _, l := range strings.Split(sheetText(raw), "\n") {
		m := line.FindStringSubmatch(strings.TrimRight(l, "\r"))
		if m == nil {
			continue
		}
		named++
		p, err := resolveBeside(dir, m[1])
		if err != nil {
			if q := sameNameAudio(path); q != "" {
				p = q
			} else {
				continue
			}
		}
		if seen[p] {
			continue
		}
		seen[p] = true
		switch ext := strings.ToLower(filepath.Ext(p)); {
		case ext == ".flac":
			compressed++
		case foreignAudioExt(ext):
			if ForeignAudio == nil {
				return nil, false
			}
			compressed++
		}
		files = append(files, p)
	}
	if named == 0 || compressed > 1 {
		return nil, false
	}
	// A sheet whose files are all missing is still an image, and opening it says so.
	return files, true
}

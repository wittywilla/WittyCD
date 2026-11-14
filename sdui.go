// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package wittycd

import (
	"github.com/wittywilla/wittymodule/sdui"
	core "github.com/wittywilla/wittymodule/sdui/core"
)

// ModuleID is this module's ID, the same as module.json's.
const ModuleID = "wittycd"

// PageIdentify is "Find this album" for one disc (?item=<its id>): what MusicBrainz knows it
// could be, each to choose, for an administrator. It is the TV's way to the web player's
// identify dialog, as the Apple TV and Android TV apps have no web view.
const PageIdentify = ModuleID + "-identify"

// SourceMatches answers the identify page: the matches for a disc (item), each running
// CommandIdentify, and a row going back to the disc's own names.
const SourceMatches = ModuleID + ".matches"

// CommandIdentify uses a release's names and cover for a disc (item); params.release "" goes
// back to the disc's own names. WittyCD's own route, which checks again who is asking.
const CommandIdentify = ModuleID + ".identify"

// Commands are WittyCD's own commands, as sdui.json declares them.
func Commands() []core.CommandSpec {
	return []core.CommandSpec{
		{Op: CommandIdentify, Label: "Use these names", Method: "POST", Path: "/api/modules/" + ModuleID + "/cds/{item}/identify", Body: `{"releaseId":"{release}"}`,
			Description: "Audio CDs: an administrator uses a MusicBrainz release's names and cover for a disc (item); params.release \"\" goes back to the disc's own names.",
			Params:      []core.CommandParam{{Name: "release", Type: "string", Label: "MusicBrainz release"}}},
	}
}

// UI is WittyCD's part of every app (module API 5, WittyMediaServer docs/SDUI-MODULES.md),
// written to sdui.json beside module.json. An audio CD's page and its tracks are the server's
// music pages, and the player is the app's own screen (the web part's, in a browser); what is
// here is the theme scope and the page that finds which album a disc is, which its service
// answers.
func UI() *sdui.Module {
	m := sdui.New(ModuleID, "Audio CDs")
	m.AddSource(sdui.SourceSpec{Type: SourceMatches, Label: "What this disc could be", Icon: "search", Title: "Matches",
		Description: "For an administrator: the albums MusicBrainz has for a disc's table of contents, each choosing its names and cover, and a row going back to the disc's own names. Only the table of contents is sent.",
		Params:      []sdui.PropSpec{{Name: "item", Type: "ref", Label: "Disc", Required: true, Help: "The disc's ID, $item on its page."}},
		Yields:      []string{"view"}, CardStyle: "tile"})
	m.Commands = Commands()
	p := sdui.Page{ID: PageIdentify, Kind: "custom", Title: "Find this album", Icon: "disc", Scope: ModuleID, Sections: []sdui.Section{
		sdui.Sec("intro", "Text", sdui.Props("text", "Choose the album this disc is. Its tracks take that album's names and cover; nothing about the disc's files changes."), nil),
		sdui.Sec("matches", "List", sdui.Props("title", "Matches", "focusable", true, "empty", "message", "emptyText", "Nobody has told MusicBrainz about this disc yet, so it keeps the names it came with."),
			sdui.Src(SourceMatches, "item", "$item")),
		sdui.Notice("credit", "info", "Album data from MusicBrainz (CC0). Covers from the Cover Art Archive."),
	}}
	p.Description = "Find which album an audio CD is, for an administrator, on every app: opened with ?item=<the disc's id>."
	m.AddPage(p)
	return m
}

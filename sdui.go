// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package wittycd

import "github.com/wittywilla/wittymodule/sdui"

// ModuleID is this module's ID, the same as module.json's.
const ModuleID = "wittycd"

// UI is WittyCD's part of every app (module API 4, WittyMediaServer docs/SDUI-MODULES.md),
// written to sdui.json beside module.json. It is only the theme scope, on purpose:
//
//   - An audio CD's page and its tracks are the server's music pages, drawn from what this
//     module's helper reads, so a page or menu entry here would only show them twice.
//   - The helper reads no settings of its own, and answers no data source: a CD image is an
//     item of the server's music libraries already.
//
// The scope still names the module for the theme editor.
func UI() *sdui.Module {
	return sdui.New(ModuleID, "Audio CDs")
}

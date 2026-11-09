// SPDX-License-Identifier: AGPL-3.0-or-later
//
// WittyCD - Audio CD images played as discs: gapless, with their tracks, indexes and CD-TEXT, in a CD player.
// Copyright (C) 2026 WittyWilla and contributors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// Command wittycd inspects audio CD images with the WittyCD engine: tracks,
// indexes, CD-TEXT, disc IDs, and a read of the whole image to find damage. It
// reads nothing but the image and writes nothing, and its output holds no audio,
// so it is safe to attach to a bug report.
package main

import (
	"fmt"
	"os"

	"github.com/wittywilla/wittycd/cli"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "info", "check":
			os.Exit(cli.CD(os.Args[1:]))
		case "version", "--version", "-v":
			fmt.Println("wittycd", version)
			return
		}
	}
	fmt.Fprint(os.Stderr, `wittycd `+version+`: inspect audio CD images

  wittycd info  [-json] <image>   tracks, indexes, CD-TEXT, flags and disc ID
  wittycd check [-json] <image>   read the whole image and report what is wrong with it

<image> is a .cue, .toc, .ccd, .nrg, .chd, or a bare .iso, .img or .bin of CD audio.
`)
	os.Exit(2)
}

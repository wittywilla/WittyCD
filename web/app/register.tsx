// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors
//
// WittyCD's screens in WittyMediaServer's web app (module API 5; integration.json says what it
// claims). The app calls register(host) once: the CD player (/cd/:id, opened for a disc of the
// "cd" kind), and the engine the app's audio player runs a disc with (its audio-engine slot), so
// the music carries on with the bar along the bottom when someone leaves the player's page. Every
// other app plays a disc with its own player and finds its album with the SDUI page wittycd-identify.
import { lazy, Suspense } from 'react'
import { Navigate } from 'react-router-dom'
import { Spinner } from '@wms/app/components/common'
import { useCdEngine } from './cdEngine'

const CdPlayer = lazy(() => import('./CdPlayer'))

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function register(_host: unknown): any {
  return {
    routes: {
      '/cd/:id': () => <Suspense fallback={<Spinner />}><CdPlayer /></Suspense>,
    },
    players: {
      cd: ({ item }: { item: { id: number } }) => <Navigate replace to={`/cd/${item.id}?play=1`} />,
    },
    audioEngines: { cd: useCdEngine },
  }
}

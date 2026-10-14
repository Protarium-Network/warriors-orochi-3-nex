# warriors-orochi-3-nex

A NEX server for **Warriors Orochi 3 Hyper** on the Wii U (*Musou Orochi 2
Hyper* in Japan), so the online mode can work again. It's built on the
[Pretendo](https://github.com/PretendoNetwork) libraries (`nex-go`,
`nex-protocols-go`, `nex-protocols-common-go`) and written in Go, with
PostgreSQL behind it.

Started from our Ninja Gaiden 3 server, since both games ship the same NEX
3.4.13 and use ranking, DataStore and matchmaking.

## Where it stands

Tried on real consoles (one EUR, one USA):

- login, ticket and secure connection
- matchmaking, including a USA console joining a session hosted on an EUR one
- ranking reads (the leaderboard comes back with data)

Not confirmed yet: uploading scores from a finished match, and the DataStore
side (Customize data upload/download). The first run into Musou Battlefield
died with `106-0103` because `AcquireNexUniqueID` wasn't answered; that is
fixed but hasn't been re-tested on a console as I write this.

## Numbers you'll want

| | |
|---|---|
| access key | `d74bb27d` |
| NEX version | 3.4.13 (`ngs/3.4.x.3`) |
| `LegacyConnectionSignature` | `false` (set `PN_WO3_LEGACY_SIGNATURE=1` if a console sits on 106-0502) |
| `UseStructureHeader` | `false` |
| game server ID | `1010ea00` |

The ID is read straight out of the EUR executable and it's the Japanese
title-ID half, so it's one ID for every region, the way DKC: Tropical Freeze
does it. `config.go` also lists the USA (`10110200`) and EUR (`10112b00`)
title halves in case some build asks for those; whatever hands out your NEX
tokens should send all of them to the same auth port. A USA and an EUR console
both requested `1010ea00` in testing.

## Things that bit me

- **The game refuses sessions from another region.** Attribute 0 of a
  matchmake session carries the console's region in its high 16 bits
  (EUR is 3, USA is 2), and the same value sits at byte 46 of the application
  buffer. Searching matched exactly on it, so a USA console never found an EUR
  host, and even when it did the client dropped the session. The server now
  ignores that word when searching, and rewrites it to the requester's own
  region in the `BrowseMatchmakeSession` reply. See
  `internal/.../matchmake-extension/database/find_matchmake_session_by_search_criteria.go`
  and `browse_matchmake_session.go`.
- **`AcquireNexUniqueID` must return something.** The game calls it when it
  enters Musou Battlefield. Leaving `GenerateNEXUniqueID` unset answers
  NotImplemented and the console shows `106-0103`. It's a plain counter here.
- **`LegacyConnectionSignature` depends on the NEX version**, not on the
  console. Most 3.4.x titles of ours want `false`, Trine 2 wanted `true`.

## What's registered

Authentication and secure endpoints. Secure has Secure Connection, Utility,
NAT Traversal, Ranking, DataStore, MatchMaking, MatchMakingExt and
MatchmakeExtension. Every call is logged as
`[WO3 Secure] PID=... protocol=0x.. method=0x..`, which is the quickest way
to see what the game actually uses. `PROTOCOL_COVERAGE.md` has the list and
`RECON.md` how the constants were recovered.

`Ranking::GetStats` isn't implemented; the game asks for it right after a
ranking read and seems fine without.

## Running it

```bash
cp .env.example .env
cp settings.example.json settings.json   # PID + NEX password of your console
docker compose up --build                # UDP 29000 (auth) / 29001 (secure)
```

Set `PN_WO3_SECURE_HOST` in `.env` to the address your console should reach
the secure server on. Without Docker: `go build -o wo3-nex .`, then run it
with the `PN_WO3_*` variables from `compose.yaml` and a PostgreSQL URI in
`PN_WO3_POSTGRES_URI`.

With an account server in front, set `PN_WO3_LOCAL_MODE=0` plus
`PN_WO3_NEX_TOKEN_AES_KEY` (64 hex chars) and `PN_WO3_NEX_PASSWORD_SECRET`
(32+ bytes, hex) matching it. DataStore uploads need an S3-compatible bucket
(`PN_S3_*`; MinIO works).

`internal/nex-protocols-common-go-patch/` is a vendored fork of
`nex-protocols-common-go` with the DataStore/S3 and matchmaking changes this
stack needs.

## Finding things in the executable

`recon/` has the small Python scripts used to dump the RPX and find the access
key and game server ID. They don't include any game files.

## License

AGPL-3.0, see [LICENSE](LICENSE) and [NOTICE](NOTICE).

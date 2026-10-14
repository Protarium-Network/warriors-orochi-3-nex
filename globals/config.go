package globals

import (
	"os"
	"strings"
)

// NEX configuration for "Warriors Orochi 3 Hyper" (Wii U, Omega Force /
// Koei Tecmo, 2012; "Musou Orochi 2 Hyper" in Japan).
//
// AccessKey and game server ID come from the retail EUR executable
// (OROCHI2_WiiU_EURelease.rpx, see RECON.md) and match kinnay.github.io's
// public Wii U NEX database (id 269543936, key d74bb27d,
// branch origin/release/ngs/3.4.x.3, build 3_4_13_3_0).
//
// The EUR binary carries the JPN title-ID half (1010ea00) as its game server
// ID, so the ID is shared across regions (same pattern as DKC:TF). The USA and
// EUR title IDs are listed as well so the NASC / nex_token front-end maps every
// plausible ID to this server's auth port.
const (
	AccessKey = "d74bb27d"

	GameServerIDJPN = "1010ea00" // title 000500001010EA00 (WUP-P-AHBJ) - in the EUR binary
	GameServerIDUSA = "10110200" // title 0005000010110200 (WUP-P-AHBE)
	GameServerIDEUR = "10112b00" // title 0005000010112B00 (WUP-P-AHBP)
)

// GameServerIDs lists every region this server is meant to serve.
var GameServerIDs = []string{GameServerIDJPN, GameServerIDUSA, GameServerIDEUR}

// kinnay build tag "build:3_4_13_3_0" (origin/release/ngs/3.4.x.3), same NGS
// generation as Trine 2 / Sochi 2014 / DKC:TF.
//
// MatchMaking is pinned to 3.3.0 (pre-3.4 MatchmakeSession layout: no
// ProgressScore byte, as console-captured on Trine 2 and Sochi 2014). Ranking
// is bumped to 3.6.0 in nex/secure.go so RankingRankData.UpdateTime is
// serialized.
const (
	NEXMajor = 3
	NEXMinor = 4
	NEXPatch = 13

	MatchMakingMajor = 3
	MatchMakingMinor = 3
	MatchMakingPatch = 0
)

// LegacyConnectionSignature selects the PRUDPv1 CONNECT-ACK signature scheme.
// Unverified on a console for this title: DKC:TF (3.4.13) needed false, Trine 2
// needed true. Set
// PN_WO3_LEGACY_SIGNATURE=1 if the console reports 106-0502.
var LegacyConnectionSignature = false

func init() {
	if v := strings.TrimSpace(os.Getenv("PN_WO3_LEGACY_SIGNATURE")); v != "" {
		LegacyConnectionSignature = v == "1"
	}
}

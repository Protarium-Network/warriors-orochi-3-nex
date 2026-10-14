# Recon — Warriors Orochi 3 Hyper (Wii U)

Source: `OROCHI2_WiiU_EURelease.rpx` (EUR build, internal name "OROCHI2").
Reproduce with the scripts in `recon/` (see `recon/README.md`).

## Constants

| Field | Value | Evidence |
|---|---|---|
| Access key | `d74bb27d` | UTF-16BE literal at flat-dump offset `0x905b04`; matches kinnay `nexwiiu.json` ("WARRIORS OROCHI 3 Hyper"). |
| Game server ID | `0x1010ea00` (269543936) | kinnay database; equals the JPN title-ID half (`000500001010EA00`). The binary is the EUR build, so the ID is shared across regions. |
| NEX build | `3.4.13` (`ngs/3.4.x.3`) | kinnay: `build:3_4_13_3_0`. Library tags `NEX_3_0_1_5`, `NEX_DS/RK/MM/UT_3_0_1` in `.rodata`. |

### Regions (wiiubrew title DB)

| Region | Product code | Title ID | Game server ID |
|---|---|---|---|
| JPN | WUP-P-AHBJ | `000500001010EA00` | `1010ea00` (confirmed) |
| USA | WUP-P-AHBE | `0005000010110200` | `10110200` (assumed same as JPN; registered defensively) |
| EUR | WUP-P-AHBP | `0005000010112B00` | `10112b00` (assumed same as JPN; registered defensively) |

## What the game links

`nn::nex` clients: TicketGranting/Authentication, SecureConnection,
AccountManagement, Health, Monitoring, Notification,
NintendoNotificationEvent, RemoteLogDevice, NATTraversal (+Relay,
ReportInternal), MatchMakingProtocol(+Ext), MatchmakeExtension, Ranking,
Utility, DataStore.

Game-side names (RTTI/strings): `COnlineSession`, `COnlineMatchWaitWin`,
`CDuelRankList` / `CDuelRankingListWin` (Duel ranking),
`CCustomizeWinOnlineDownLoadRanking` / `DownLoadSave` / `UpLoadServer`
(Customize data via DataStore), `Ktol::Session::*` (session layer),
`_ktolNexSendDataDirect`, plus methods `UploadScore`, `UploadCommonData`,
`GetRanking`, `DeleteScore`, `AutoMatchmake*`, `BrowseMatchmakeSession`,
`CreateMatchmakeSession`, `JoinMatchmakeSession*`, `UpdateMatchmakeSession*`.

## Server consequences

Auth + Secure endpoints; secure registers SecureConnection, Utility, NAT
Traversal, MatchMaking, MatchMakingExt, MatchmakeExtension, Ranking,
DataStore. The exact RMC methods the retail code calls were not mapped
(stripped binary); the server enables the full common protocol set and logs
every call (`[WO3 Secure] PID=… protocol=… method=…`).

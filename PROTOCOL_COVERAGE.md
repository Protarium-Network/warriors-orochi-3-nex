# Protocol coverage

| Protocol | Coverage |
|---|---|
| Ticket Granting | Login / LoginEx / RequestTicket (stock) |
| Secure Connection | stock (insecure Register enabled for Wii U) |
| Utility | stock (AcquireNexUniqueID …) |
| Ranking | UploadScore, UploadCommonData, GetRanking (Range / Near / User), GetCommonData, GetCachedTopX; Postgres `wo3_*` tables |
| DataStore | PreparePostObject / CompletePostObject / PrepareGetObject / GetMeta / search; ratings stub; objects in `datastore.objects`, blobs in S3 |
| NAT Traversal | stock |
| MatchMaking / MatchMakingExt / MatchmakeExtension | stock common handlers; MatchMaking lib pinned 3.3.0; pre-3.4 `AutoMatchmakeWithSearchCriteria_Postpone` decoder and friends-only `AutoMatchmakeWithGatheringID_Postpone` handler ported from the Sochi 2014 server |

Not registered: Notification, Monitoring, Health, AccountManagement
(housekeeping clients the game does not need answered).

**Unverified on a console.** The first session's log (`[WO3 Secure] …`) will
say which methods the retail code really calls.

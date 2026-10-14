package nex

import (
	"fmt"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	nex "github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_datastore "github.com/PretendoNetwork/nex-protocols-common-go/v2/datastore"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	common_matchmaking "github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making"
	common_matchmaking_ext "github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making-ext"
	common_matchmake_extension "github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension"
	common_nat_traversal "github.com/PretendoNetwork/nex-protocols-common-go/v2/nat-traversal"
	common_ranking "github.com/PretendoNetwork/nex-protocols-common-go/v2/ranking"
	common_secure "github.com/PretendoNetwork/nex-protocols-common-go/v2/secure-connection"
	common_utility "github.com/PretendoNetwork/nex-protocols-common-go/v2/utility"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	matchmaking "github.com/PretendoNetwork/nex-protocols-go/v2/match-making"
	matchmaking_ext "github.com/PretendoNetwork/nex-protocols-go/v2/match-making-ext"
	matchmaking_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
	nat_traversal "github.com/PretendoNetwork/nex-protocols-go/v2/nat-traversal"
	ranking "github.com/PretendoNetwork/nex-protocols-go/v2/ranking"
	secure "github.com/PretendoNetwork/nex-protocols-go/v2/secure-connection"
	utility "github.com/PretendoNetwork/nex-protocols-go/v2/utility"
	"github.com/Protarium-Network/warriors-orochi-3-nex/database"
	"github.com/Protarium-Network/warriors-orochi-3-nex/globals"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var nexUniqueIDCounter atomic.Uint64

func init() { nexUniqueIDCounter.Store(uint64(time.Now().Unix()) << 16) }

var SecureServer *nex.PRUDPServer
var SecureEndpoint *nex.PRUDPEndPoint

func StartSecureServer() {
	SecureServer = nex.NewPRUDPServer()

	// See globals/config.go: console-selectable via PN_WO3_LEGACY_SIGNATURE.
	SecureServer.PRUDPV1Settings.LegacyConnectionSignature = globals.LegacyConnectionSignature

	SecureEndpoint = nex.NewPRUDPEndPoint(1)
	SecureEndpoint.IsSecureEndPoint = true
	SecureEndpoint.ServerAccount = globals.SecureServerAccount
	SecureEndpoint.AccountDetailsByPID = globals.AccountDetailsByPID
	SecureEndpoint.AccountDetailsByUsername = globals.AccountDetailsByUsername
	SecureServer.BindPRUDPEndPoint(SecureEndpoint)

	SecureServer.LibraryVersions.SetDefault(nex.NewLibraryVersion(int(globals.NEXMajor), int(globals.NEXMinor), int(globals.NEXPatch)))
	// WO3's matchmaking packets use the pre-3.4 layout: MatchmakeSession
	// has no ProgressScore byte and search criteria have no VacantParticipants
	// field. The 0x20 after ParticipationCount is the SessionKey buffer
	// length, not a progress score.
	SecureServer.LibraryVersions.MatchMaking = nex.NewLibraryVersion(int(globals.MatchMakingMajor), int(globals.MatchMakingMinor), int(globals.MatchMakingPatch))
	// Bump Ranking to 3.6.0 so RankingRankData.UpdateTime is serialized -
	// without it every entry after the first in a multi-row response is read
	// at the wrong offset client-side.
	SecureServer.LibraryVersions.Ranking = nex.NewLibraryVersion(3, 6, 0)
	SecureServer.AccessKey = globals.AccessKey
	// See authentication.go: this title's client does NOT write
	// structure-header version bytes. Leave ByteStreamSettings at the
	// library default (false) on both servers.

	SecureEndpoint.OnData(func(packet nex.PacketInterface) {
		request := packet.RMCMessage()
		if request == nil {
			return
		}
		pid := uint64(packet.Sender().PID())
		fmt.Printf("[WO3 Secure] PID=%d protocol=0x%02X method=0x%02X\n", pid, request.ProtocolID, request.MethodID)
	})

	SecureEndpoint.OnConnectionEnded(func(connection *nex.PRUDPConnection) {
		fmt.Printf("[WO3 Secure] PID=%d disconnected\n", uint64(connection.PID()))
	})

	// Captures of this title against the official servers show real online
	// matchmaking (AutoMatchmakeWithSearchCriteria_Postpone /
	// OpenParticipation / EndParticipation), so the manager is created here
	// once the secure endpoint exists.
	globals.MatchmakingManager = common_globals.NewMatchmakingManager(SecureEndpoint, database.Postgres)
	globals.MatchmakingManager.GetUserFriendPIDs = globals.GetUserFriendPIDs

	registerSecureServerProtocols()

	port, _ := strconv.Atoi(os.Getenv("PN_WO3_SECURE_PORT"))
	globals.Logger.Successf("[WO3] Secure server listening on UDP %d", port)
	SecureServer.Listen(port)
}

// registerSecureServerProtocols wires up the secure-connection handshake,
// utility, ranking and DataStore, plus online
// matchmaking (NAT Traversal + MatchMaking + MatchMakingExt +
// MatchmakeExtension) which captures confirm this title uses.
func registerSecureServerProtocols() {
	secureProtocol := secure.NewProtocol()
	SecureEndpoint.RegisterServiceProtocol(secureProtocol)
	secureCommon := common_secure.NewCommonProtocol(secureProtocol)
	secureCommon.EnableInsecureRegister()
	secureCommon.CreateReportDBRecord = database.CreateReportDBRecord

	utilityProtocol := utility.NewProtocol()
	SecureEndpoint.RegisterServiceProtocol(utilityProtocol)
	utilityCommon := common_utility.NewCommonProtocol(utilityProtocol)
	// The game calls AcquireNexUniqueID right after the secure register; a
	// missing generator answers NotImplemented and the console shows 106-0103.
	// ponytail: in-memory counter seeded from the clock, unique per process run.
	utilityCommon.GenerateNEXUniqueID = func() uint64 { return nexUniqueIDCounter.Add(1) }

	rankingProtocol := ranking.NewProtocol()
	SecureEndpoint.RegisterServiceProtocol(rankingProtocol)
	rankingCommon := common_ranking.NewCommonProtocol(rankingProtocol)
	rankingCommon.GetRankingsAndCountByCategoryAndRankingOrderParam = database.WO3GetRankingsAndCountByCategoryAndRankingOrderParam
	rankingCommon.GetRankingsByMode = database.WO3GetRankings
	rankingCommon.GetCommonData = database.WO3GetCommonData
	rankingCommon.UploadCommonData = database.WO3UploadCommonData
	rankingCommon.InsertRankingByPIDAndRankingScoreData = database.WO3InsertRankingByPIDAndRankingScoreData

	datastoreProtocol := datastore.NewProtocol()
	SecureEndpoint.RegisterServiceProtocol(datastoreProtocol)
	datastoreCommon := common_datastore.NewCommonProtocol(datastoreProtocol)
	datastoreCommon.GetObjectInfosByDataStoreSearchParam = database.GetObjectInfosByDataStoreSearchParam
	datastoreCommon.InitializeObjectByPreparePostParam = database.InitializeObjectByPreparePostParam
	datastoreCommon.InitializeObjectRatingWithSlot = database.InitializeObjectRatingWithSlot
	datastoreCommon.GetObjectInfoByDataID = database.GetObjectInfoByDataID
	datastoreCommon.UpdateObjectPeriodByDataIDWithPassword = database.UpdateObjectPeriodByDataIDWithPassword
	datastoreCommon.UpdateObjectMetaBinaryByDataIDWithPassword = database.UpdateObjectMetaBinaryByDataIDWithPassword
	datastoreCommon.UpdateObjectDataTypeByDataIDWithPassword = database.UpdateObjectDataTypeByDataIDWithPassword
	datastoreCommon.GetObjectInfoByDataIDWithPassword = database.GetObjectInfoByDataIDWithPassword
	datastoreCommon.GetObjectInfoByPersistenceTargetWithPassword = database.GetObjectInfoByPersistenceTargetWithPassword
	datastoreProtocol.GetRatings = database.WO3GetRatings
	// Required by DataStore::CompletePostObject (last step of the score
	// upload/attachment flow).
	datastoreCommon.GetObjectOwnerByDataID = database.GetObjectOwnerByDataID
	datastoreCommon.GetObjectSizeByDataID = database.GetObjectSizeByDataID
	datastoreCommon.UpdateObjectUploadCompletedByDataID = database.UpdateObjectUploadCompletedByDataID
	datastoreCommon.DeleteObjectByDataID = database.DeleteObjectByDataID

	// DataStore::PreparePostObject (score-upload attachment) needs an
	// S3-compatible presigned-URL backend. Point PN_S3_ENDPOINT at any
	// S3-compatible service (self-hosted MinIO works well).
	s3Endpoint := os.Getenv("PN_S3_ENDPOINT")
	if s3Endpoint != "" {
		minioClient, err := minio.New(s3Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(os.Getenv("PN_S3_ACCESS_KEY"), os.Getenv("PN_S3_SECRET_KEY"), ""),
			Secure: true,
		})
		if err != nil {
			globals.Logger.Errorf("[WO3] Failed to create MinIO client: %s", err.Error())
		} else {
			s3Bucket := os.Getenv("PN_S3_BUCKET")
			if s3Bucket == "" {
				s3Bucket = "wo3-datastore"
			}
			datastoreCommon.S3Bucket = s3Bucket
			datastoreCommon.SetDataKeyBase("wo3")
			datastoreCommon.SetMinIOClient(minioClient)
		}
	} else {
		globals.Logger.Warning("[WO3] PN_S3_ENDPOINT not set - DataStore::PreparePostObject will fail")
	}

	// NAT Traversal + matchmaking - real online play (confirmed by capture:
	// AutoMatchmakeWithSearchCriteria_Postpone / OpenParticipation /
	// EndParticipation, plus NAT Traversal::ReportNATProperties).
	natTraversalProtocol := nat_traversal.NewProtocol()
	SecureEndpoint.RegisterServiceProtocol(natTraversalProtocol)
	common_nat_traversal.NewCommonProtocol(natTraversalProtocol)

	matchMakingProtocol := matchmaking.NewProtocol()
	SecureEndpoint.RegisterServiceProtocol(matchMakingProtocol)
	commonMatchMakingProtocol := common_matchmaking.NewCommonProtocol(matchMakingProtocol)
	commonMatchMakingProtocol.SetManager(globals.MatchmakingManager)

	matchMakingExtProtocol := matchmaking_ext.NewProtocol()
	SecureEndpoint.RegisterServiceProtocol(matchMakingExtProtocol)
	commonMatchMakingExtProtocol := common_matchmaking_ext.NewCommonProtocol(matchMakingExtProtocol)
	commonMatchMakingExtProtocol.SetManager(globals.MatchmakingManager)

	matchmakeExtensionProtocol := matchmake_extension.NewProtocol()
	SecureEndpoint.RegisterServiceProtocol(matchmakeExtensionProtocol)
	commonMatchmakeExtensionProtocol := common_matchmake_extension.NewCommonProtocol(matchmakeExtensionProtocol)
	commonMatchmakeExtensionProtocol.SetManager(globals.MatchmakingManager)
	wo3AutoMatchPatch := newWO3AutoMatchProtocol(matchmakeExtensionProtocol.AutoMatchmakeWithSearchCriteriaPostpone)
	wo3AutoMatchPatch.SetEndpoint(SecureEndpoint)
	matchmakeExtensionProtocol.Patches = wo3AutoMatchPatch
	matchmakeExtensionProtocol.PatchedMethods = []uint32{matchmake_extension.MethodAutoMatchmakeWithSearchCriteriaPostpone}
	// Required by AutoMatchmakePostpone / AutoMatchmakeWithSearchCriteriaPostpone
	// ("find an opponent"): both hard-fail with Core::NotImplemented if left
	// nil even though there is nothing to clean up here. Pretendo's own
	// reference server sets the same pair of callbacks to no-ops.
	commonMatchmakeExtensionProtocol.CleanupMatchmakeSessionSearchCriterias = func(searchCriterias types.List[matchmaking_types.MatchmakeSessionSearchCriteria]) {}
	commonMatchmakeExtensionProtocol.CleanupSearchMatchmakeSession = func(matchmakeSession *matchmaking_types.MatchmakeSession) {}
	matchmakeExtensionProtocol.AutoMatchmakeWithGatheringIDPostpone = wo3AutoMatchmakeWithGatheringIDPostpone
}

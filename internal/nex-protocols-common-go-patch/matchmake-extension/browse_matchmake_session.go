package matchmake_extension

import (
	"fmt"
	"strconv"
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/database"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
)

func (commonProtocol *CommonProtocol) browseMatchmakeSession(err error, packet nex.PacketInterface, callID uint32, searchCriteria match_making_types.MatchmakeSessionSearchCriteria, resultRange types.ResultRange) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	commonProtocol.manager.Mutex.RLock()

	searchCriterias := []match_making_types.MatchmakeSessionSearchCriteria{searchCriteria}

	lstSearchCriteria := types.NewList[match_making_types.MatchmakeSessionSearchCriteria]()
	lstSearchCriteria = searchCriterias

	if commonProtocol.CleanupMatchmakeSessionSearchCriterias != nil {
		commonProtocol.CleanupMatchmakeSessionSearchCriterias(lstSearchCriteria)
	}

	sessions, nexError := database.FindMatchmakeSessionBySearchCriteria(commonProtocol.manager, connection, searchCriterias, resultRange, nil)
	if nexError != nil {
		commonProtocol.manager.Mutex.RUnlock()
		return nil, nexError
	}

	lstGathering := types.NewList[match_making_types.GatheringHolder]()

	// * WO3: the game refuses sessions whose region word (high 16 bits of
	// * attribute 0, mirrored at application buffer byte 46) differs from its
	// * own. Present every session as same-region to the requester.
	var region uint64
	var haveRegion bool
	if len(searchCriteria.Attribs) > 0 {
		if v, perr := strconv.ParseUint(string(searchCriteria.Attribs[0]), 10, 32); perr == nil {
			region, haveRegion = v>>16, true
		}
	}

	for _, session := range sessions {
		if haveRegion {
			if len(session.Attributes) > 0 {
				old := uint32(session.Attributes[0])
				session.Attributes[0] = types.NewUInt32(uint32(region)<<16 | old&0xFFFF)
			}
			if len(session.ApplicationBuffer) > 46 {
				session.ApplicationBuffer[46] = byte(region)
			}
			fmt.Printf("[WO3 MM] browse: presenting session %d to PID %d with region word %d\n", uint32(session.ID), uint32(connection.PID()), region)
		}

		// * Scrap session key and user password
		session.SessionKey = make([]byte, 0)
		session.UserPassword = ""

		matchmakeSessionDataHolder := match_making_types.NewGatheringHolder()
		matchmakeSessionDataHolder.Object = session.Copy().(match_making_types.GatheringInterface)

		lstGathering = append(lstGathering, matchmakeSessionDataHolder)
	}

	commonProtocol.manager.Mutex.RUnlock()

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	lstGathering.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodBrowseMatchmakeSession
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterBrowseMatchmakeSession != nil {
		go commonProtocol.OnAfterBrowseMatchmakeSession(packet, searchCriteria, resultRange)
	}

	return rmcResponse, nil
}

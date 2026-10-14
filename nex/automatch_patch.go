package nex

import (
	"fmt"

	nex "github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	protocolglobals "github.com/PretendoNetwork/nex-protocols-go/v2/globals"
	matchmakingtypes "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	matchmakeextension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
)

type wo3AutoMatchHandler func(
	err error,
	packet nex.PacketInterface,
	callID uint32,
	criteria types.List[matchmakingtypes.MatchmakeSessionSearchCriteria],
	gathering matchmakingtypes.GatheringHolder,
	message types.String,
) (*nex.RMCMessage, *nex.Error)

// wo3AutoMatchProtocol patches only method 0x0F. WO3 serializes this
// request with the pre-3.4 search-criteria/session layout.
type wo3AutoMatchProtocol struct {
	endpoint nex.EndpointInterface
	handler  wo3AutoMatchHandler
}

func newWO3AutoMatchProtocol(handler wo3AutoMatchHandler) *wo3AutoMatchProtocol {
	return &wo3AutoMatchProtocol{handler: handler}
}

func (protocol *wo3AutoMatchProtocol) Endpoint() nex.EndpointInterface {
	return protocol.endpoint
}

func (protocol *wo3AutoMatchProtocol) SetEndpoint(endpoint nex.EndpointInterface) {
	protocol.endpoint = endpoint
}

func wo3RequestLibraryVersions(endpoint nex.EndpointInterface) *nex.LibraryVersions {
	base := endpoint.LibraryVersions()
	versions := *base
	versions.MatchMaking = nex.NewLibraryVersion(3, 3, 0)
	return &versions
}

func decodeWO3AutoMatchRequest(parameters []byte, endpoint nex.EndpointInterface) (
	types.List[matchmakingtypes.MatchmakeSessionSearchCriteria],
	matchmakingtypes.GatheringHolder,
	types.String,
	error,
) {
	criteria := types.NewList[matchmakingtypes.MatchmakeSessionSearchCriteria]()
	gathering := matchmakingtypes.NewGatheringHolder()
	message := types.NewString("")
	stream := nex.NewByteStreamIn(parameters, wo3RequestLibraryVersions(endpoint), endpoint.ByteStreamSettings())

	if err := criteria.ExtractFrom(stream); err != nil {
		return criteria, gathering, message, fmt.Errorf("read search criteria: %w", err)
	}
	if err := gathering.ExtractFrom(stream); err != nil {
		return criteria, gathering, message, fmt.Errorf("read gathering: %w", err)
	}
	if err := message.ExtractFrom(stream); err != nil {
		return criteria, gathering, message, fmt.Errorf("read message: %w", err)
	}
	if stream.Remaining() != 0 {
		return criteria, gathering, message, fmt.Errorf("unexpected trailing request data: %d bytes", stream.Remaining())
	}

	// This field does not exist in WO3's request layout. The generic common
	// handler expects it and the behavior of the older implementation was one
	// vacant participant when omitted.
	for index := range criteria {
		criteria[index].VacantParticipants = types.NewUInt16(1)
	}

	return criteria, gathering, message, nil
}

func (protocol *wo3AutoMatchProtocol) HandlePacket(packet nex.PacketInterface) {
	request := packet.RMCMessage()
	if request == nil || !request.IsRequest || request.ProtocolID != matchmakeextension.ProtocolID || request.MethodID != matchmakeextension.MethodAutoMatchmakeWithSearchCriteriaPostpone {
		return
	}

	criteria, gathering, message, err := decodeWO3AutoMatchRequest(request.Parameters, packet.Sender().Endpoint())
	if err != nil {
		protocolglobals.Logger.Errorf("[WO3] AutoMatch request: %v", err)
		protocolglobals.RespondError(packet, matchmakeextension.ProtocolID, nex.NewError(nex.ResultCodes.Core.InvalidArgument, err.Error()))
		return
	}

	response, rmcError := protocol.handler(nil, packet, request.CallID, criteria, gathering, message)
	if rmcError != nil {
		protocolglobals.RespondError(packet, matchmakeextension.ProtocolID, rmcError)
		return
	}
	protocolglobals.Respond(packet, response)
}

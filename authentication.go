package nexproto

import (
	"errors"
	"fmt"

	nex "github.com/PretendoNetwork/nex-go"
)

const (
	AuthenticationProtocolID = 0xA

	AuthenticationMethodLogin          = 0x1
	AuthenticationMethodLoginEx        = 0x2
	AuthenticationMethodRequestTicket  = 0x3
	AuthenticationMethodGetPID         = 0x4
	AuthenticationMethodGetName        = 0x5
	AuthenticationMethodLoginWithParam = 0x6
)

type AuthenticationProtocol struct {
	server                *nex.Server
	LoginHandler          func(err error, client *nex.Client, callID uint32, username string)
	LoginExHandler        func(err error, client *nex.Client, callID uint32, username string, authenticationInfo *AuthenticationInfo)
	RequestTicketHandler  func(err error, client *nex.Client, callID uint32, userPID uint32, serverPID uint32)
	GetPIDHandler         func(err error, client *nex.Client, callID uint32, username string)
	GetNameHandler        func(err error, client *nex.Client, callID uint32, userPID uint32)
	LoginWithParamHandler func(err error, client *nex.Client, callID uint32)
}

type NintendoLoginData struct {
	Token string
}

type AuthenticationInfo struct {
	Token         string
	NGSVersion    uint32
	TokenType     uint8
	ServerVersion uint32

	hierarchy []nex.StructureInterface
	*nex.NullData
}

func (authenticationInfo *AuthenticationInfo) GetHierarchy() []nex.StructureInterface {
	return authenticationInfo.hierarchy
}

func (authenticationInfo *AuthenticationInfo) ExtractFromStream(stream *nex.StreamIn) error {
	var err error
	var token string

	token, err = stream.ReadString()

	if err != nil {
		return err
	}

	if len(stream.Bytes()[stream.ByteOffset():]) < 9 {
		return errors.New("[AuthenticationInfo::ExtractFromStream] Data size too small")
	}

	authenticationInfo.Token = token
	authenticationInfo.TokenType = stream.ReadUInt8()
	authenticationInfo.NGSVersion = stream.ReadUInt32LE()
	authenticationInfo.ServerVersion = stream.ReadUInt32LE()

	fmt.Printf("%+v\n", authenticationInfo)

	return nil
}

func NewAuthenticationInfo() *AuthenticationInfo {
	authenticationInfo := &AuthenticationInfo{}

	nullData := nex.NewNullData()

	authenticationInfo.NullData = nullData

	authenticationInfo.hierarchy = []nex.StructureInterface{
		nullData,
	}

	return authenticationInfo
}

func (authenticationProtocol *AuthenticationProtocol) Setup() {
	nexServer := authenticationProtocol.server

	nexServer.On("Data", func(packet nex.PacketInterface) {
		request := packet.GetRMCRequest()

		if AuthenticationProtocolID == request.GetProtocolID() {
			switch request.GetMethodID() {
			case AuthenticationMethodLogin:
				go authenticationProtocol.handleLogin(packet)
			case AuthenticationMethodLoginEx:
				go authenticationProtocol.handleLoginEx(packet)
			case AuthenticationMethodRequestTicket:
				go authenticationProtocol.handleRequestTicket(packet)
			case AuthenticationMethodGetPID:
				go authenticationProtocol.handleGetPID(packet)
			case AuthenticationMethodGetName:
				go authenticationProtocol.handleGetName(packet)
			case AuthenticationMethodLoginWithParam:
				go authenticationProtocol.handleLoginWithParam(packet)
			default:
				fmt.Printf("Unsupported Authentication method ID: %#v\n", request.GetMethodID())
			}
		}
	})
}

func (authenticationProtocol *AuthenticationProtocol) respondNotImplemented(packet nex.PacketInterface) {
	client := packet.GetSender()
	request := packet.GetRMCRequest()

	rmcResponse := nex.NewRMCResponse(AuthenticationProtocolID, request.GetCallID())
	rmcResponse.SetError(0x80010002)

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PacketInterface

	if packet.GetVersion() == 1 {
		responsePacket, _ = nex.NewPacketV1(client, nil)
	} else {
		responsePacket, _ = nex.NewPacketV0(client, nil)
	}

	responsePacket.SetVersion(packet.GetVersion())
	responsePacket.SetSource(packet.GetDestination())
	responsePacket.SetDestination(packet.GetSource())
	responsePacket.SetType(nex.DataPacket)
	responsePacket.SetPayload(rmcResponseBytes)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	authenticationProtocol.server.Send(responsePacket)
}

func (authenticationProtocol *AuthenticationProtocol) Login(handler func(err error, client *nex.Client, callID uint32, username string)) {
	authenticationProtocol.LoginHandler = handler
}

func (authenticationProtocol *AuthenticationProtocol) LoginEx(handler func(err error, client *nex.Client, callID uint32, username string, authenticationInfo *AuthenticationInfo)) {
	authenticationProtocol.LoginExHandler = handler
}

func (authenticationProtocol *AuthenticationProtocol) RequestTicket(handler func(err error, client *nex.Client, callID uint32, userPID uint32, serverPID uint32)) {
	authenticationProtocol.RequestTicketHandler = handler
}

func (authenticationProtocol *AuthenticationProtocol) GetPID(handler func(err error, client *nex.Client, callID uint32, username string)) {
	authenticationProtocol.GetPIDHandler = handler
}

func (authenticationProtocol *AuthenticationProtocol) GetName(handler func(err error, client *nex.Client, callID uint32, userPID uint32)) {
	authenticationProtocol.GetNameHandler = handler
}

func (authenticationProtocol *AuthenticationProtocol) LoginWithParam(handler func(err error, client *nex.Client, callID uint32)) {
	authenticationProtocol.LoginWithParamHandler = handler
}

func (authenticationProtocol *AuthenticationProtocol) handleLogin(packet nex.PacketInterface) {
	if authenticationProtocol.LoginHandler == nil {
		fmt.Println("[Warning] AuthenticationProtocol::Login not implemented")
		go authenticationProtocol.respondNotImplemented(packet)
		return
	}

	client := packet.GetSender()
	request := packet.GetRMCRequest()

	callID := request.GetCallID()
	parameters := request.GetParameters()

	parametersStream := nex.NewStreamIn(parameters, authenticationProtocol.server)

	username, err := parametersStream.ReadString()

	if err != nil {
		go authenticationProtocol.LoginHandler(err, client, callID, "")
		return
	}

	go authenticationProtocol.LoginHandler(nil, client, callID, username)
}

func (authenticationProtocol *AuthenticationProtocol) handleLoginEx(packet nex.PacketInterface) {
	if authenticationProtocol.LoginExHandler == nil {
		fmt.Println("[Warning] AuthenticationProtocol::LoginEx not implemented")
		go authenticationProtocol.respondNotImplemented(packet)
		return
	}

	client := packet.GetSender()
	request := packet.GetRMCRequest()

	callID := request.GetCallID()
	parameters := request.GetParameters()

	parametersStream := nex.NewStreamIn(parameters, authenticationProtocol.server)

	username, err := parametersStream.ReadString()

	if err != nil {
		go authenticationProtocol.LoginExHandler(err, client, callID, "", nil)
		return
	}

	dataHolderName, err := parametersStream.ReadString()

	if err != nil {
		go authenticationProtocol.LoginExHandler(err, client, callID, "", nil)
		return
	}

	if dataHolderName != "AuthenticationInfo" {
		err := errors.New("[AuthenticationProtocol::LoginEx] Data holder name does not match")
		go authenticationProtocol.LoginExHandler(err, client, callID, "", nil)
		return
	}

	_ = parametersStream.ReadUInt32LE() // length including this field

	dataHolderContent, err := parametersStream.ReadBuffer()

	if err != nil {
		go authenticationProtocol.LoginExHandler(err, client, callID, "", nil)
		return
	}

	dataHolderContentStream := nex.NewStreamIn(dataHolderContent, authenticationProtocol.server)

	authenticationInfo, err := dataHolderContentStream.ReadStructure(NewAuthenticationInfo())

	if err != nil {
		go authenticationProtocol.LoginExHandler(err, client, callID, "", nil)
		return
	}

	go authenticationProtocol.LoginExHandler(nil, client, callID, username, authenticationInfo.(*AuthenticationInfo))
}

func (authenticationProtocol *AuthenticationProtocol) handleRequestTicket(packet nex.PacketInterface) {
	if authenticationProtocol.RequestTicketHandler == nil {
		fmt.Println("[Warning] AuthenticationProtocol::RequestTicket not implemented")
		go authenticationProtocol.respondNotImplemented(packet)
		return
	}

	client := packet.GetSender()
	request := packet.GetRMCRequest()

	callID := request.GetCallID()
	parameters := request.GetParameters()

	if len(parameters) != 8 {
		err := errors.New("[AuthenticationProtocol::RequestTicket] Parameters length not 8")
		go authenticationProtocol.RequestTicketHandler(err, client, callID, 0, 0)
	}

	parametersStream := nex.NewStreamIn(parameters, authenticationProtocol.server)

	userPID := parametersStream.ReadUInt32LE()
	serverPID := parametersStream.ReadUInt32LE()

	go authenticationProtocol.RequestTicketHandler(nil, client, callID, userPID, serverPID)
}

func (authenticationProtocol *AuthenticationProtocol) handleGetPID(packet nex.PacketInterface) {
	if authenticationProtocol.GetPIDHandler == nil {
		fmt.Println("[Warning] AuthenticationProtocol::GetPID not implemented")
		go authenticationProtocol.respondNotImplemented(packet)
		return
	}

	client := packet.GetSender()
	request := packet.GetRMCRequest()

	callID := request.GetCallID()
	parameters := request.GetParameters()

	parametersStream := nex.NewStreamIn(parameters, authenticationProtocol.server)

	username, err := parametersStream.ReadString()

	if err != nil {
		go authenticationProtocol.GetPIDHandler(err, client, callID, "")
		return
	}

	go authenticationProtocol.GetPIDHandler(nil, client, callID, username)
}

func (authenticationProtocol *AuthenticationProtocol) handleGetName(packet nex.PacketInterface) {
	if authenticationProtocol.GetNameHandler == nil {
		fmt.Println("[Warning] AuthenticationProtocol::GetName not implemented")
		go authenticationProtocol.respondNotImplemented(packet)
		return
	}

	client := packet.GetSender()
	request := packet.GetRMCRequest()

	callID := request.GetCallID()
	parameters := request.GetParameters()

	parametersStream := nex.NewStreamIn(parameters, authenticationProtocol.server)

	if len(parameters) != 4 {
		err := errors.New("[AuthenticationProtocol::GetName] Parameters length not 4")
		go authenticationProtocol.RequestTicketHandler(err, client, callID, 0, 0)
	}

	userPID := parametersStream.ReadUInt32LE()

	go authenticationProtocol.GetNameHandler(nil, client, callID, userPID)
}

func (authenticationProtocol *AuthenticationProtocol) handleLoginWithParam(packet nex.PacketInterface) {
	if authenticationProtocol.LoginWithParamHandler == nil {
		fmt.Println("[Warning] AuthenticationProtocol::LoginWithParam not implemented")
		go authenticationProtocol.respondNotImplemented(packet)
		return
	}

	// Unsure what data is sent here, or how to trigger the console to send it
}

func NewAuthenticationProtocol(server *nex.Server) *AuthenticationProtocol {
	authenticationProtocol := &AuthenticationProtocol{server: server}

	authenticationProtocol.Setup()

	return authenticationProtocol
}

package xvm

import (
	"chainmaker.org/chainmaker-go/common/serialize"
	"chainmaker.org/chainmaker-go/wxvm/xvm/exec"
	"fmt"
)

const (
	contextIDKey = "ctxid"
	responseKey  = "callResponse"
)

type responseDesc struct {
	Body  []byte
	Error bool
}

type contextServiceResolver struct {
	contextService *ContextService
}

func NewContextServiceResolver(service *ContextService) exec.Resolver {
	return &contextServiceResolver{
		contextService: service,
	}
}

func (s *contextServiceResolver) ResolveGlobal(module, name string) (int64, bool) {
	return 0, false
}

func (s *contextServiceResolver) ResolveFunc(module, name string) (interface{}, bool) {
	fullname := module + "." + name
	switch fullname {
	case "env._call_method":
		return s.cCallMethod, true
	case "env._fetch_response":
		return s.cFetchResponse, true
	default:
		return nil, false
	}
}
func (s *contextServiceResolver) cFetchResponse(ctx exec.Context, userBuf, userLen uint32) uint32 {
	codec := exec.NewCodec(ctx)
	iresponse := ctx.GetUserData(responseKey)
	if iresponse == nil {
		exec.Throw(exec.NewTrap("call fetchResponse on nil value"))
	}
	response := iresponse.(responseDesc)
	userbuf := codec.Bytes(userBuf, userLen)
	if len(response.Body) != len(userbuf) {
		exec.Throw(exec.NewTrap(fmt.Sprintf("call fetchResponse with bad length, got %d, expect %d", len(userbuf), len(response.Body))))
	}
	copy(userbuf, response.Body)
	success := uint32(1)
	if response.Error {
		success = 0
	}
	ctx.SetUserData(responseKey, nil)
	return success
}

func (s *contextServiceResolver) cCallMethod(
	ctx exec.Context,
	methodAddr, methodLen uint32,
	requestAddr, requestLen uint32,
	responseAddr, responseLen uint32,
	successAddr uint32) uint32 {

	codec := exec.NewCodec(ctx)
	ctxId := ctx.GetUserData(contextIDKey).(int64)
	method := codec.String(methodAddr, methodLen)
	requestBufTmp := codec.Bytes(requestAddr, requestLen)
	responseBuf := codec.Bytes(responseAddr, responseLen)
	requestBuf := make([]byte, len(requestBufTmp))
	copy(requestBuf, requestBufTmp)

	context, ok := s.contextService.Context(ctxId)
	if !ok {
		s.contextService.logger.Errorw("encounter bad ctx id. failed to call method:", method)
		codec.SetUint32(successAddr, 1)
		return uint32(0)
	}

	reqItems := serialize.NewEasyCodecWithBytes(requestBuf).GetItems()
	s.contextService.ctxId = ctxId
	context.in = reqItems
	context.requestBody = requestBuf
	context.gasUsed = ctx.GasUsed()
	switch method {
	case "GetObject":
		s.contextService.GetState()
	case "PutObject":
		s.contextService.PutState()
	case "DeleteObject":
		s.contextService.DeleteState()
	case "NewIterator":
		s.contextService.NewIterator()
	case "GetCallArgs":
		s.contextService.GetCallArgs()
	case "SetOutput":
		s.contextService.SetOutput()
	case "ContractCall":
		s.contextService.CallContract()
	case "LogMsg":
		s.contextService.LogMessage()
	default:
		s.contextService.logger.Errorw("no such method ", method)
	}
	if context.err != nil {
		s.contextService.logger.Errorw("failed to call method:", method, context.err)
		codec.SetUint32(successAddr, 1)
		return uint32(0)
	}

	possibleResponseBuf := serialize.NewEasyCodecWithItems(context.resp).Marshal()

	// fast path
	if len(possibleResponseBuf) <= len(responseBuf) {
		copy(responseBuf, possibleResponseBuf)
		codec.SetUint32(successAddr, 1)
		return uint32(len(possibleResponseBuf))
	}

	// slow path
	var responseDesc responseDesc
	responseDesc.Body = possibleResponseBuf
	ctx.SetUserData(responseKey, responseDesc)
	return uint32(len(responseDesc.Body))
}

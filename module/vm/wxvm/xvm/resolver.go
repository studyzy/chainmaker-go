package xvm

import (
	"chainmaker.org/chainmaker-go/wxvm/xvm/exec"
	"fmt"
	"github.com/gogo/protobuf/proto"
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
	requestBuf := codec.Bytes(requestAddr, requestLen)
	responseBuf := codec.Bytes(responseAddr, responseLen)

	var respMessage proto.Message
	var err error

	switch method {
	case "GetObject":
		var req GetRequest
		if err = proto.Unmarshal(requestBuf, &req); err == nil {
			respMessage, err = s.contextService.GetObject(ctxId, &req)
		}
	case "PutObject":
		var req PutRequest
		if err = proto.Unmarshal(requestBuf, &req); err == nil {
			respMessage, err = s.contextService.PutObject(ctxId, &req)
		}
	case "DeleteObject":
		var req DeleteRequest
		if err = proto.Unmarshal(requestBuf, &req); err == nil {
			respMessage, err = s.contextService.DeleteObject(ctxId, &req)
		}
	case "NewIterator":
		var req IteratorRequest
		if err = proto.Unmarshal(requestBuf, &req); err == nil {
			respMessage, err = s.contextService.NewIterator(ctxId, &req)
		}
	case "GetCallArgs":
		var req GetCallArgsRequest
		if err = proto.Unmarshal(requestBuf, &req); err == nil {
			respMessage, err = s.contextService.GetCallArgs(ctxId, &req)
		}
	case "SetOutput":
		var req SetOutputRequest
		if err = proto.Unmarshal(requestBuf, &req); err == nil {
			respMessage, err = s.contextService.SetOutput(ctxId, &req)
		}
	case "ContractCall":
		var req ContractCallRequest
		if err = proto.Unmarshal(requestBuf, &req); err == nil {
			respMessage, err = s.contextService.ContractCall(ctxId, &req, ctx.GasUsed())
		}
	case "LogMsg":
		var req LogMsgRequest
		if err = proto.Unmarshal(requestBuf, &req); err == nil {
			respMessage, err = s.contextService.LogMsg(ctxId, &req)
		}
	default:
		s.contextService.logger.Errorw("no such method ", method)
	}
	if err != nil {
		s.contextService.logger.Errorw("failed to call method:", method, err)
		codec.SetUint32(successAddr, 1)
		return uint32(0)
	}

	possibleResponseBuf, err := proto.Marshal(respMessage)

	// fast path
	if err != nil {
		s.contextService.logger.Errorw("contract syscall error", "ctxid", ctxId, "method", method, "error", err)
		msg := err.Error()
		if len(msg) <= len(responseBuf) {
			copy(responseBuf, msg)
			codec.SetUint32(successAddr, 0)
			return uint32(len(msg))
		}
	} else {
		if len(possibleResponseBuf) <= len(responseBuf) {
			copy(responseBuf, possibleResponseBuf)
			codec.SetUint32(successAddr, 1)
			return uint32(len(possibleResponseBuf))
		}
	}

	// slow path
	var responseDesc responseDesc
	if err != nil {
		s.contextService.logger.Errorw("contract service call error", "ctxid", ctxId, "method", method, "error", err)
		responseDesc.Error = true
		responseDesc.Body = []byte(err.Error())
	} else {
		responseDesc.Body = possibleResponseBuf
	}
	ctx.SetUserData(responseKey, responseDesc)
	return uint32(len(responseDesc.Body))
}

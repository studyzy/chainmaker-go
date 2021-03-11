package xvm

import (
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"errors"
	"fmt"
	"sort"
	"sync"
)

const DefaultCap = 10

type ContextService struct {
	lock    sync.Mutex
	chainId string
	ctxId   int64
	ctxMap  map[int64]*Context
	logger  *logger.CMLogger
}

// NewContextService build a ContextService
func NewContextService(chainId string) *ContextService {
	return &ContextService{
		lock:    sync.Mutex{},
		chainId: chainId,
		ctxId:   0,
		ctxMap:  make(map[int64]*Context),
		logger:  logger.GetLoggerByChain(logger.MODULE_VM, chainId),
	}
}

func (c *ContextService) Context(id int64) (*Context, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	ctx, ok := c.ctxMap[id]
	return ctx, ok
}

func (c *ContextService) MakeContext(contractId *commonPb.ContractId, txSimContext protocol.TxSimContext,
	contractResult *commonPb.ContractResult, parameters map[string]string) *Context {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.ctxId++
	ctx := &Context{
		ID:             c.ctxId,
		Parameters:     parameters,
		TxSimContext:   txSimContext,
		ContractId:     contractId,
		ContractResult: contractResult,
	}
	c.ctxMap[ctx.ID] = ctx

	var args []*ArgPair
	for key, value := range parameters {
		args = append(args, &ArgPair{
			Key:   key,
			Value: []byte(value),
		})
	}
	sort.Slice(args, func(i, j int) bool {
		return args[i].Key < args[j].Key
	})
	ctx.callArgs = &CallArgs{
		Args: args,
	}
	return ctx
}

func (c *ContextService) DestroyContext(ctx *Context) {
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.ctxMap, ctx.ID)
}

// PutObject implements Syscall interface
func (c *ContextService) PutObject(ctxId int64, in *PutRequest) (*PutResponse, error) {
	context, ok := c.Context(ctxId)
	if !ok {
		return nil, fmt.Errorf("put object encounter bad ctx id:%d", ctxId)
	}
	context.TxSimContext.Put(context.ContractId.ContractName, in.Key, in.Value)

	return &PutResponse{}, nil
}

// GetObject implements Syscall interface
func (c *ContextService) GetObject(ctxId int64, in *GetRequest) (*GetResponse, error) {
	context, ok := c.Context(ctxId)
	if !ok {
		return nil, fmt.Errorf("get object encounter bad ctx id:%d", ctxId)
	}
	value, err := context.TxSimContext.Get(context.ContractId.ContractName, in.Key)

	if err != nil {
		return nil, err
	}
	return &GetResponse{
		Value: value,
	}, nil
}

// DeleteObject implements Syscall interface
func (c *ContextService) DeleteObject(ctxId int64, in *DeleteRequest) (*DeleteResponse, error) {
	context, ok := c.Context(ctxId)
	if !ok {
		return nil, fmt.Errorf("delete object encounter bad ctx id:%d", ctxId)
	}
	err := context.TxSimContext.Del(context.ContractId.ContractName, in.Key)
	if err != nil {
		return nil, err
	}
	return &DeleteResponse{}, nil
}

// NewIterator implements Syscall interface
func (c *ContextService) NewIterator(ctxId int64, in *IteratorRequest) (*IteratorResponse, error) {
	context, ok := c.Context(ctxId)
	if !ok {
		return nil, fmt.Errorf("new iterator encounter bad ctx id:%d", ctxId)
	}
	limit := in.Cap
	if limit <= 0 {
		limit = DefaultCap
	}
	iter, _ := context.TxSimContext.Select(context.ContractId.ContractName, in.Start, in.Limit)

	out := new(IteratorResponse)
	for iter.Next() && limit > 0 {
		out.Items = append(out.Items, &IteratorItem{
			Key:   append([]byte(""), iter.Key()...), //make a copy
			Value: append([]byte(""), iter.Value()...),
		})
		limit -= 1
	}
	if iter.Error() != nil {
		return nil, errors.New(fmt.Sprintf("Select error, %s", iter.Error()))
	}
	iter.Release()
	return out, nil
}

// GetCallArgs implements Syscall interface
func (c *ContextService) GetCallArgs(ctxId int64, in *GetCallArgsRequest) (*CallArgs, error) {
	context, ok := c.Context(ctxId)
	if !ok {
		return nil, fmt.Errorf("get call args encounter bad ctx id:%d", ctxId)
	}
	return context.callArgs, nil
}

// SetOutput implements Syscall interface
func (c *ContextService) SetOutput(ctxId int64, in *SetOutputRequest) (*SetOutputResponse, error) {
	context, ok := c.Context(ctxId)
	if !ok {
		return nil, fmt.Errorf("set output encounter bad ctx id:%d", ctxId)
	}
	resp := in.GetResponse()
	switch resp.Code {
	case 0:
		context.ContractResult.Code = commonPb.ContractResultCode_OK
	default:
		context.ContractResult.Code = commonPb.ContractResultCode_FAIL
	}
	context.ContractResult.Message = resp.Message
	context.ContractResult.Result = resp.Result
	return new(SetOutputResponse), nil
}

// ContractCall implements Syscall interface
func (c *ContextService) ContractCall(ctxId int64, in *ContractCallRequest, gasUsed uint64) (*ContractCallResponse, error) {
	context, ok := c.Context(ctxId)
	if !ok {
		return nil, fmt.Errorf("contract call encounter bad ctx id:%d", ctxId)
	}
	paramMap := make(map[string]string, 8)

	for _, arg := range in.Args {
		paramMap[arg.Key] = string(arg.Value)
	}

	contractResult, txStatusCode := context.TxSimContext.CallContract(&commonPb.ContractId{ContractName: in.Contract}, in.Method, nil, paramMap, gasUsed, commonPb.TxType_INVOKE_USER_CONTRACT)

	if txStatusCode != commonPb.TxStatusCode_SUCCESS {
		return &ContractCallResponse{
			Response: &Response{
				Code:    int32(contractResult.Code),
				Message: contractResult.Message,
				Result:  contractResult.Result,
			},
		}, fmt.Errorf(contractResult.Message)
	}
	return &ContractCallResponse{
		Response: nil,
	}, nil
}

// PostLog handle log entry from contract
func (c *ContextService) LogMsg(ctxId int64, in *LogMsgRequest) (*LogMsgResponse, error) {
	context, ok := c.Context(ctxId)
	if !ok {
		return nil, fmt.Errorf("bad ctx id:%d", ctxId)
	}
	c.logger.Debugf("wxvm log >>[%s] [%d] %s", context.TxSimContext.GetTx().Header.TxId, ctxId, in.Msg)
	return new(LogMsgResponse), nil
}

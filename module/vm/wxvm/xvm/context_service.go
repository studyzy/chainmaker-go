package xvm

import (
	"errors"
	"fmt"
	"sync"

	"chainmaker.org/chainmaker-go/common/serialize"
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
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
	ctx.callArgs = serialize.ParamsMapToEasyCodecItem(parameters)
	return ctx
}

func (c *ContextService) DestroyContext(ctx *Context) {
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.ctxMap, ctx.ID)
}

// PutObject implements Syscall interface
func (c *ContextService) PutObject(ctxId int64, in []*serialize.EasyCodecItem) ([]*serialize.EasyCodecItem, error) {
	context, ok := c.Context(ctxId)
	if !ok {
		return nil, fmt.Errorf("put object encounter bad ctx id:%d", ctxId)
	}
	key, ok := serialize.GetValueFromItems(in, "key", serialize.EasyKeyType_USER)
	if !ok {
		return nil, fmt.Errorf("put object request have no key:%d", ctxId)
	}
	value, ok := serialize.GetValueFromItems(in, "value", serialize.EasyKeyType_USER)
	if !ok {
		return nil, fmt.Errorf("put object request have no value:%d", ctxId)
	}
	context.TxSimContext.Put(context.ContractId.ContractName, []byte(key.(string)), []byte(value.(string)))
	items := make([]*serialize.EasyCodecItem, 0)
	return items, nil
}

// GetObject implements Syscall interface
func (c *ContextService) GetObject(ctxId int64, in []*serialize.EasyCodecItem) ([]*serialize.EasyCodecItem, error) {
	context, ok := c.Context(ctxId)
	if !ok {
		return nil, fmt.Errorf("get object encounter bad ctx id:%d", ctxId)
	}
	key, ok := serialize.GetValueFromItems(in, "key", serialize.EasyKeyType_USER)
	if !ok {
		return nil, fmt.Errorf("get object request have no key:%d", ctxId)
	}
	value, err := context.TxSimContext.Get(context.ContractId.ContractName, []byte(key.(string)))

	if err != nil {
		return nil, err
	}
	items := make([]*serialize.EasyCodecItem, 0)
	var valueItem serialize.EasyCodecItem
	valueItem.Key = "value"
	valueItem.KeyType = serialize.EasyKeyType_USER
	valueItem.ValueType = serialize.EasyValueType_BYTES
	valueItem.Value = value
	items = append(items, &valueItem)
	return items, nil
}

// DeleteObject implements Syscall interface
func (c *ContextService) DeleteObject(ctxId int64, in []*serialize.EasyCodecItem) ([]*serialize.EasyCodecItem, error) {
	context, ok := c.Context(ctxId)
	if !ok {
		return nil, fmt.Errorf("delete object encounter bad ctx id:%d", ctxId)
	}
	key, ok := serialize.GetValueFromItems(in, "key", serialize.EasyKeyType_USER)
	if !ok {
		return nil, fmt.Errorf("delete object request have no key:%d", ctxId)
	}
	err := context.TxSimContext.Del(context.ContractId.ContractName, []byte(key.(string)))
	if err != nil {
		return nil, err
	}
	items := make([]*serialize.EasyCodecItem, 0)
	return items, nil
}

// NewIterator implements Syscall interface
func (c *ContextService) NewIterator(ctxId int64, in []*serialize.EasyCodecItem) ([]*serialize.EasyCodecItem, error) {
	context, ok := c.Context(ctxId)
	if !ok {
		return nil, fmt.Errorf("new iterator encounter bad ctx id:%d", ctxId)
	}
	limit, ok := serialize.GetValueFromItems(in, "limit", serialize.EasyKeyType_SYSTEM)
	if !ok {
		return nil, fmt.Errorf("new iterator request have no limit:%d", ctxId)
	}
	start, ok := serialize.GetValueFromItems(in, "start", serialize.EasyKeyType_SYSTEM)
	if !ok {
		return nil, fmt.Errorf("new iterator request have no start:%d", ctxId)
	}
	cap, ok := serialize.GetValueFromItems(in, "cap", serialize.EasyKeyType_SYSTEM)
	if !ok {
		return nil, fmt.Errorf("new iterator request have no cap:%d", ctxId)
	}
	capLimit := cap.(int32)
	if capLimit <= 0 {
		capLimit = DefaultCap
	}
	iter, _ := context.TxSimContext.Select(context.ContractId.ContractName, []byte(start.(string)), []byte(limit.(string)))

	//out := new(IteratorResponse)
	out := make([]*serialize.EasyCodecItem, 0)
	for iter.Next() && capLimit > 0 {
		var item serialize.EasyCodecItem
		item.Key = string(iter.Key())
		item.KeyType = serialize.EasyKeyType_USER
		item.ValueType = serialize.EasyValueType_BYTES
		item.Value = iter.Value()
		out = append(out, &item)
		capLimit -= 1
	}
	if iter.Error() != nil {
		return nil, errors.New(fmt.Sprintf("Select error, %s", iter.Error()))
	}
	iter.Release()
	return out, nil
}

// GetCallArgs implements Syscall interface
func (c *ContextService) GetCallArgs(ctxId int64, in []*serialize.EasyCodecItem) ([]*serialize.EasyCodecItem, error) {
	context, ok := c.Context(ctxId)
	if !ok {
		return nil, fmt.Errorf("get call args encounter bad ctx id:%d", ctxId)
	}
	return context.callArgs, nil
}

// SetOutput implements Syscall interface
func (c *ContextService) SetOutput(ctxId int64, in []*serialize.EasyCodecItem) ([]*serialize.EasyCodecItem, error) {
	context, ok := c.Context(ctxId)
	if !ok {
		return nil, fmt.Errorf("set output  bad ctx id:%d", ctxId)
	}
	code, ok := serialize.GetValueFromItems(in, "code", serialize.EasyKeyType_USER)
	if !ok {
		return nil, fmt.Errorf("set out put respsonse have no code:%d", ctxId)
	}
	msg, _ := serialize.GetValueFromItems(in, "msg", serialize.EasyKeyType_USER)

	result, _ := serialize.GetValueFromItems(in, "result", serialize.EasyKeyType_USER)

	switch code.(int32) {
	case 0:
		context.ContractResult.Code = commonPb.ContractResultCode_OK
	default:
		context.ContractResult.Code = commonPb.ContractResultCode_FAIL
	}
	context.ContractResult.Message = msg.(string)
	context.ContractResult.Result = result.([]byte)
	items := make([]*serialize.EasyCodecItem, 0)
	return items, nil
}

// ContractCall implements Syscall interface
func (c *ContextService) ContractCall(ctxId int64, in []*serialize.EasyCodecItem, gasUsed uint64) ([]*serialize.EasyCodecItem, error) {
	context, ok := c.Context(ctxId)
	if !ok {
		return nil, fmt.Errorf("contract call encounter bad ctx id:%d", ctxId)
	}
	contract, ok := serialize.GetValueFromItems(in, "contract", serialize.EasyKeyType_USER)
	if !ok {
		return nil, fmt.Errorf("contract call request  have no contract name:%d", ctxId)
	}
	method, ok := serialize.GetValueFromItems(in, "method", serialize.EasyKeyType_USER)
	if !ok {
		return nil, fmt.Errorf("contract call request  have no method name:%d", ctxId)
	}
	args, ok := serialize.GetValueFromItems(in, "args", serialize.EasyKeyType_USER)
	if !ok {
		return nil, fmt.Errorf("contract call request  have no args:%d", ctxId)
	}
	argsItems := serialize.EasyUnmarshal(args.([]byte))
	paramMap := serialize.EasyCodecItemToParamsMap(argsItems)
	contractResult, txStatusCode := context.TxSimContext.CallContract(&commonPb.ContractId{ContractName: contract.(string)}, method.(string), nil, paramMap, gasUsed, commonPb.TxType_INVOKE_USER_CONTRACT)
	respItems := make([]*serialize.EasyCodecItem, 0)
	if txStatusCode != commonPb.TxStatusCode_SUCCESS {
		var codeItem serialize.EasyCodecItem
		codeItem.KeyType = serialize.EasyKeyType_USER
		codeItem.Key = "code"
		codeItem.ValueType = serialize.EasyValueType_INT32
		codeItem.Value = contractResult.Code
		respItems = append(respItems, &codeItem)
		var msgItem serialize.EasyCodecItem
		msgItem.KeyType = serialize.EasyKeyType_USER
		msgItem.Key = "msg"
		msgItem.ValueType = serialize.EasyValueType_STRING
		msgItem.Value = contractResult.Message
		respItems = append(respItems, &msgItem)
		var resultItem serialize.EasyCodecItem
		resultItem.KeyType = serialize.EasyKeyType_USER
		resultItem.Key = "result"
		resultItem.ValueType = serialize.EasyValueType_BYTES
		resultItem.Value = contractResult.Result
		respItems = append(respItems, &resultItem)
		return respItems, fmt.Errorf(contractResult.Message)
	}
	return respItems, nil
}

// PostLog handle log entry from contract
func (c *ContextService) LogMsg(ctxId int64, in []*serialize.EasyCodecItem) ([]*serialize.EasyCodecItem, error) {
	context, ok := c.Context(ctxId)
	if !ok {
		return nil, fmt.Errorf("bad ctx id:%d", ctxId)
	}
	msg, _ := serialize.GetValueFromItems(in, "msg", serialize.EasyKeyType_USER)
	c.logger.Debugf("wxvm log >>[%s] [%d] %s", context.TxSimContext.GetTx().Header.TxId, ctxId, msg.(string))
	msgItems := make([]*serialize.EasyCodecItem, 0)
	return msgItems, nil
}

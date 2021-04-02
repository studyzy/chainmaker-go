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
	lock        sync.Mutex
	chainId     string
	ctxId       int64
	ctxMap      map[int64]*Context
	logger      *logger.CMLogger
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

// PutState implements Syscall interface
func (c *ContextService) PutState() int32 {
	context,_ := c.Context(c.ctxId)

	key, ok1 := serialize.GetValueFromItems(context.in, "key", serialize.EasyKeyType_USER)
	value, ok2 := serialize.GetValueFromItems(context.in, "value", serialize.EasyKeyType_USER)
	if !ok1 || !ok2 {
		context.err = fmt.Errorf("put state param[key | value] is required:%d", c.ctxId)
		return protocol.ContractSdkSignalResultFail
	}

	context.TxSimContext.Put(context.ContractId.ContractName, []byte(key.(string)), []byte(value.(string)))

	items := make([]*serialize.EasyCodecItem, 0)

	context.resp = items
	return protocol.ContractSdkSignalResultSuccess
}

// GetState implements Syscall interface
func (c *ContextService) GetState() int32 {
	context,_ := c.Context(c.ctxId)
	key, ok := serialize.GetValueFromItems(context.in, "key", serialize.EasyKeyType_USER)
	if !ok {
		context.err = fmt.Errorf("get object param[key] is required:%d", c.ctxId)
		return protocol.ContractSdkSignalResultFail
	}

	value, err := context.TxSimContext.Get(context.ContractId.ContractName, []byte(key.(string)))
	if err != nil {
		context.err = err
		return protocol.ContractSdkSignalResultFail
	}

	items := make([]*serialize.EasyCodecItem, 0)
	var valueItem serialize.EasyCodecItem
	valueItem.Key = "value"
	valueItem.KeyType = serialize.EasyKeyType_USER
	valueItem.ValueType = serialize.EasyValueType_BYTES
	valueItem.Value = value
	items = append(items, &valueItem)

	context.resp = items
	return protocol.ContractSdkSignalResultSuccess
}

// DeleteState implements Syscall interface
func (c *ContextService) DeleteState() int32 {
	context,_ := c.Context(c.ctxId)
	key, ok := serialize.GetValueFromItems(context.in, "key", serialize.EasyKeyType_USER)
	if !ok {
		context.err = fmt.Errorf("delete state request have no key:%d", c.ctxId)
		return protocol.ContractSdkSignalResultFail
	}
	err := context.TxSimContext.Del(context.ContractId.ContractName, []byte(key.(string)))
	if err != nil {
		context.err = err
		return protocol.ContractSdkSignalResultFail
	}
	items := make([]*serialize.EasyCodecItem, 0)

	context.resp = items
	return protocol.ContractSdkSignalResultSuccess
}

// NewIterator implements Syscall interface
func (c *ContextService) NewIterator() int32 {
	context,_ := c.Context(c.ctxId)

	limit, ok1 := serialize.GetValueFromItems(context.in, "limit", serialize.EasyKeyType_SYSTEM)
	start, ok2 := serialize.GetValueFromItems(context.in, "start", serialize.EasyKeyType_SYSTEM)
	cap, ok3 := serialize.GetValueFromItems(context.in, "cap", serialize.EasyKeyType_SYSTEM)
	if !ok1 || !ok2 || !ok3 {
		context.err = fmt.Errorf("new iterator param[limit | start | cap] is required:%d", c.ctxId)
		return protocol.ContractSdkSignalResultFail
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
		context.err = errors.New(fmt.Sprintf("new iterator select error, %s", iter.Error()))
		return protocol.ContractSdkSignalResultFail
	}
	iter.Release()

	context.resp = out
	return protocol.ContractSdkSignalResultSuccess
}

// GetCallArgs implements Syscall interface
func (c *ContextService) GetCallArgs() int32 {
	context,_ := c.Context(c.ctxId)
	context.resp = context.callArgs
	return protocol.ContractSdkSignalResultSuccess
}

// SetOutput implements Syscall interface
func (c *ContextService) SetOutput() int32 {
	context,_ := c.Context(c.ctxId)
	code, ok := serialize.GetValueFromItems(context.in, "code", serialize.EasyKeyType_USER)
	if !ok {
		context.err = fmt.Errorf("set out put param[code] is required:%d", c.ctxId)
		return protocol.ContractSdkSignalResultFail
	}
	msg, ok := serialize.GetValueFromItems(context.in, "msg", serialize.EasyKeyType_USER)
	if ok {
		context.ContractResult.Message += msg.(string)
	}
	result, ok := serialize.GetValueFromItems(context.in, "result", serialize.EasyKeyType_USER)
	if ok {
		context.ContractResult.Result = []byte(result.(string))
	}
	if context.ContractResult.Code == commonPb.ContractResultCode_FAIL {
		items := make([]*serialize.EasyCodecItem, 0)
		context.resp = items
		return protocol.ContractSdkSignalResultSuccess
	}
	switch code.(int32) {
	case 0:
		context.ContractResult.Code = commonPb.ContractResultCode_OK
	default:
		context.ContractResult.Code = commonPb.ContractResultCode_FAIL
	}
	items := make([]*serialize.EasyCodecItem, 0)

	context.resp = items
	return protocol.ContractSdkSignalResultSuccess
}

// CallContract implements Syscall interface
func (c *ContextService) CallContract() int32 {
	context,_ := c.Context(c.ctxId)
	contract, ok1 := serialize.GetValueFromItems(context.in, "contract", serialize.EasyKeyType_USER)
	method, ok2 := serialize.GetValueFromItems(context.in, "method", serialize.EasyKeyType_USER)
	args, ok3 := serialize.GetValueFromItems(context.in, "args", serialize.EasyKeyType_USER)
	if !ok1 || !ok2 || !ok3 {
		context.err = fmt.Errorf("call contract param[contract | method | args] is required:%d", c.ctxId)
		return protocol.ContractSdkSignalResultFail
	}
	argsItems := serialize.EasyUnmarshal(args.([]byte))
	paramMap := serialize.EasyCodecItemToParamsMap(argsItems)
	contractResult, txStatusCode := context.TxSimContext.CallContract(&commonPb.ContractId{ContractName: contract.(string)}, method.(string), nil, paramMap, context.gasUsed, commonPb.TxType_INVOKE_USER_CONTRACT)
	respItems := make([]*serialize.EasyCodecItem, 0)
	var codeItem serialize.EasyCodecItem
	codeItem.KeyType = serialize.EasyKeyType_USER
	codeItem.Key = "code"
	codeItem.ValueType = serialize.EasyValueType_INT32
	codeItem.Value = int32(contractResult.Code)
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

	context.resp = respItems

	if txStatusCode != commonPb.TxStatusCode_SUCCESS {
		context.err = fmt.Errorf(contractResult.Message)
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// LogMessage handle log entry from contract
func (c *ContextService) LogMessage() int32 {
	context,_ := c.Context(c.ctxId)
	msg, ok := serialize.GetValueFromItems(context.in, "msg", serialize.EasyKeyType_USER)
	if !ok {
		context.err = fmt.Errorf("log message param[msg] is required:%d", c.ctxId)
		return protocol.ContractSdkSignalResultFail
	}
	c.logger.Debugf("wxvm log >>[%s] [%d] %s", context.TxSimContext.GetTx().Header.TxId, c.ctxId, msg.(string))
	msgItems := make([]*serialize.EasyCodecItem, 0)
	context.resp = msgItems
	return protocol.ContractSdkSignalResultSuccess
}
func (c *ContextService) SuccessResult() int32 {
	return c.SetOutput()
}
func (c *ContextService) ErrorResult() int32 {
	return c.SetOutput()
}

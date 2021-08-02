package xvm

import (
	"fmt"
	"sync"

	"chainmaker.org/chainmaker-go/utils"

	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker/common/serialize"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/protocol"
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

func (c *ContextService) MakeContext(contract *commonPb.Contract, txSimContext protocol.TxSimContext,
	contractResult *commonPb.ContractResult, parameters map[string][]byte) *Context {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.ctxId++
	ctx := &Context{
		ID:             c.ctxId,
		Parameters:     parameters,
		TxSimContext:   txSimContext,
		ContractId:     contract,
		ContractResult: contractResult,
	}
	c.ctxMap[ctx.ID] = ctx
	ec := serialize.NewEasyCodecWithMap(parameters)
	ctx.callArgs = ec.GetItems()
	return ctx
}

func (c *ContextService) DestroyContext(ctx *Context) {
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.ctxMap, ctx.ID)
}

// PutState implements Syscall interface
func (c *ContextService) PutState(ctxId int64) int32 {
	context, _ := c.Context(ctxId)
	ec := serialize.NewEasyCodecWithItems(context.in)
	key, err1 := ec.GetString("key")
	value, err2 := ec.GetString("value")

	if err1 != nil || err2 != nil {
		context.err = fmt.Errorf("put state param[key | value] is required:%d", c.ctxId)
		return protocol.ContractSdkSignalResultFail
	}

	if err := context.TxSimContext.Put(context.ContractId.Name, []byte(key), []byte(value)); err != nil {
		context.err = fmt.Errorf("put state param[key | value] failed, err: %v", err)
	}

	items := make([]*serialize.EasyCodecItem, 0)

	context.resp = items
	return protocol.ContractSdkSignalResultSuccess
}

// GetState implements Syscall interface
func (c *ContextService) GetState(ctxId int64) int32 {
	context, _ := c.Context(ctxId)

	ec := serialize.NewEasyCodecWithItems(context.in)
	key, err := ec.GetString("key")
	if err != nil {
		context.err = fmt.Errorf("get object param[key] is required:%d", c.ctxId)
		return protocol.ContractSdkSignalResultFail
	}

	value, err := context.TxSimContext.Get(context.ContractId.Name, []byte(key))
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
func (c *ContextService) EmitEvent(ctxId int64) int32 {
	context, _ := c.Context(ctxId)
	ec := serialize.NewEasyCodecWithItems(context.in)
	topic, err := ec.GetString("topic")
	if err != nil {
		context.err = fmt.Errorf("emit event encounter bad ctx id:%d", c.ctxId)
		return protocol.ContractSdkSignalResultFail
	}

	if err := protocol.CheckTopicStr(topic); err != nil {
		context.err = err
		return protocol.ContractSdkSignalResultFail
	}
	in := ec.GetItems()
	var eventData []string
	for i := 1; i < len(in); i++ {
		data, ok := in[i].Value.(string)
		if !ok {
			c.logger.Debugf("convert value failed, value=%v", in[i].Value)
		}

		eventData = append(eventData, data)
		c.logger.Debugf("method EmitEvent eventData :%v", data)
	}
	if err := protocol.CheckEventData(eventData); err != nil {
		context.err = err
		return protocol.ContractSdkSignalResultFail
	}
	contractEvent := &commonPb.ContractEvent{
		ContractName:    context.ContractId.Name,
		ContractVersion: context.ContractId.Version,
		Topic:           topic,
		TxId:            context.TxSimContext.GetTx().Payload.TxId,
		EventData:       eventData,
	}
	ddl := utils.GenerateSaveContractEventDdl(contractEvent, "chainId", 1, 1)
	count := utils.GetSqlStatementCount(ddl)
	if count != 1 {
		context.err = fmt.Errorf("contract event parameter error,exist sql injection")
		return protocol.ContractSdkSignalResultFail
	}
	context.ContractEvent = append(context.ContractEvent, contractEvent)
	items := make([]*serialize.EasyCodecItem, 0)

	context.resp = items
	return protocol.ContractSdkSignalResultSuccess
}

// DeleteState implements Syscall interface
func (c *ContextService) DeleteState(ctxId int64) int32 {
	context, _ := c.Context(ctxId)
	ec := serialize.NewEasyCodecWithItems(context.in)
	key, err := ec.GetString("key")
	if err != nil {
		context.err = fmt.Errorf("delete state request have no key:%d", c.ctxId)
		return protocol.ContractSdkSignalResultFail
	}
	err = context.TxSimContext.Del(context.ContractId.Name, []byte(key))
	if err != nil {
		context.err = err
		return protocol.ContractSdkSignalResultFail
	}
	items := make([]*serialize.EasyCodecItem, 0)

	context.resp = items
	return protocol.ContractSdkSignalResultSuccess
}

// NewIterator implements Syscall interface
func (c *ContextService) NewIterator(ctxId int64) int32 {
	context, _ := c.Context(ctxId)
	ec := serialize.NewEasyCodecWithItems(context.in)
	limit, err1 := ec.GetString("limit")
	start, err2 := ec.GetString("start")
	cap, err3 := ec.GetInt32("cap")

	if err1 != nil || err2 != nil || err3 != nil {
		context.err = fmt.Errorf("new iterator param[limit | start | cap] is required:%d", c.ctxId)
		return protocol.ContractSdkSignalResultFail
	}

	capLimit := cap
	if capLimit <= 0 {
		capLimit = DefaultCap
	}
	iter, _ := context.TxSimContext.Select(context.ContractId.Name, []byte(start), []byte(limit))

	//out := new(IteratorResponse)
	out := make([]*serialize.EasyCodecItem, 0)
	for iter.Next() && capLimit > 0 {
		kv, err := iter.Value()
		if err != nil {
			context.err = fmt.Errorf("new iterator select error, %s", err)
			return protocol.ContractSdkSignalResultFail
		}
		var item serialize.EasyCodecItem
		item.Key = string(kv.Key)
		item.KeyType = serialize.EasyKeyType_USER
		item.ValueType = serialize.EasyValueType_BYTES
		item.Value = kv.Value
		out = append(out, &item)
		capLimit--
	}

	iter.Release()

	context.resp = out
	return protocol.ContractSdkSignalResultSuccess
}

// GetCallArgs implements Syscall interface
func (c *ContextService) GetCallArgs(ctxId int64) int32 {
	context, _ := c.Context(ctxId)
	context.resp = context.callArgs
	return protocol.ContractSdkSignalResultSuccess
}

// SetOutput implements Syscall interface
func (c *ContextService) SetOutput(ctxId int64) int32 {
	context, _ := c.Context(ctxId)
	ec := serialize.NewEasyCodecWithItems(context.in)
	code, err := ec.GetInt32("code")
	if err != nil {
		context.err = fmt.Errorf("set out put param[code] is required:%d", c.ctxId)
		return protocol.ContractSdkSignalResultFail
	}

	msg, err := ec.GetString("msg")
	if err == nil {
		context.ContractResult.Message += msg
	}
	result, err := ec.GetString("result")
	if err == nil {
		context.ContractResult.Result = []byte(result)
	}
	if context.ContractResult.Code == 1 {
		items := make([]*serialize.EasyCodecItem, 0)
		context.resp = items
		return protocol.ContractSdkSignalResultSuccess
	}
	switch code {
	case 0:
		context.ContractResult.Code = 0
	default:
		context.ContractResult.Code = 1
	}
	items := make([]*serialize.EasyCodecItem, 0)

	context.resp = items
	return protocol.ContractSdkSignalResultSuccess
}

// CallContract implements Syscall interface
func (c *ContextService) CallContract(ctxId int64) int32 {
	context, _ := c.Context(ctxId)
	ec := serialize.NewEasyCodecWithItems(context.in)

	contract, err1 := ec.GetString("contract")
	method, err2 := ec.GetString("method")
	args, err3 := ec.GetBytes("args")
	if err1 != nil || err2 != nil || err3 != nil {
		context.err = fmt.Errorf("call contract param[contract | method | args] is required:%d", c.ctxId)
		return protocol.ContractSdkSignalResultFail
	}

	ecArg := serialize.NewEasyCodecWithBytes(args)
	paramMap := ecArg.ToMap()
	contractResult, txStatusCode := context.TxSimContext.CallContract(&commonPb.Contract{Name: contract}, method, nil,
		paramMap, context.gasUsed, commonPb.TxType_INVOKE_CONTRACT)

	ecParam := serialize.NewEasyCodec()
	ecParam.AddInt32("code", int32(contractResult.Code))
	ecParam.AddString("msg", contractResult.Message)
	ecParam.AddBytes("result", contractResult.Result)
	context.resp = ecParam.GetItems()

	if txStatusCode != commonPb.TxStatusCode_SUCCESS {
		context.err = fmt.Errorf(contractResult.Message)
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// LogMessage handle log entry from contract
func (c *ContextService) LogMessage(ctxId int64) int32 {
	context, _ := c.Context(ctxId)
	ec := serialize.NewEasyCodecWithItems(context.in)
	msg, err := ec.GetString("msg")
	if err != nil {
		context.err = fmt.Errorf("log message param[msg] is required:%d", c.ctxId)
		return protocol.ContractSdkSignalResultFail
	}
	c.logger.Debugf("wxvm log>> [%s] %s\n", context.TxSimContext.GetTx().Payload.TxId, msg)
	msgItems := make([]*serialize.EasyCodecItem, 0)
	context.resp = msgItems
	return protocol.ContractSdkSignalResultSuccess
}
func (c *ContextService) SuccessResult(ctxId int64) int32 {
	return c.SetOutput(ctxId)
}
func (c *ContextService) ErrorResult(ctxId int64) int32 {
	return c.SetOutput(ctxId)
}

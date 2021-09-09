package common

//
//import (
//	"chainmaker.org/chainmaker/logger/v2"
//	"chainmaker.org/chainmaker/protocol/v2/mock"
//	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
//	"chainmaker.org/chainmaker/pb-go/v2/config"
//	"encoding/hex"
//	"fmt"
//	"github.com/golang/mock/gomock"
//	"testing"
//)
//
//func TestValidateTx(t *testing.T) {
//	verifyTx, block := txPrepare(t)
//	hashes, _, _, _ := verifyTx.verifierTxs(block)
//
//	for _, hash := range hashes {
//		fmt.Println("test hash: ", hex.EncodeToString(hash))
//	}
//}
//
//func txPrepare(t *testing.T) (*VerifierTx, *commonpb.Block) {
//	block := newBlock()
//	contractId := &commonpb.Contract{
//		ContractName:    "ContractName",
//		ContractVersion: "1",
//		RuntimeType:     commonpb.RuntimeType_WASMER,
//	}
//
//	parameters := make(map[string]string, 8)
//	tx0 := newTx("a0000000000000000000000000000000", contractId, parameters)
//	txs := make([]*commonpb.Transaction, 0)
//	txs = append(txs, tx0)
//	block.Txs = txs
//
//	var txRWSetMap = make(map[string]*commonpb.TxRWSet, 3)
//	txRWSetMap[tx0.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx0.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractId.Name,
//			Key:          []byte("K1"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractId.Name,
//			Key:          []byte("K2"),
//			Value:        []byte("V"),
//		}},
//	}
//
//	rwHash, _ := hex.DecodeString("d02f421ed76e0e26e9def824a8b84c7c223d484762d6d060a8b71e1649d1abbf")
//	result := &commonpb.Result{
//		Code: commonpb.TxStatusCode_SUCCESS,
//		ContractResult: &commonpb.ContractResult{
//			Code:    0,
//			Result:  nil,
//			Message: "",
//			GasUsed: 0,
//		},
//		RwSetHash: rwHash,
//	}
//	tx0.Result = result
//	txResultMap := make(map[string]*commonpb.Result, 1)
//	txResultMap[tx0.Payload.TxId] = result
//
//	log := logger.GetLoggerByChain(logger.MODULE_CORE, "chain1")
//
//	ctl := gomock.NewController(t)
//	store := mock.NewMockBlockchainStore(ctl)
//	txPool := mock.NewMockTxPool(ctl)
//	ac := mock.NewMockAccessControlProvider(ctl)
//	chainConf := mock.NewMockChainConf(ctl)
//
//	store.EXPECT().TxExists(tx0).AnyTimes().Return(false, nil)
//
//	txsMap := make(map[string]*commonpb.Transaction)
//
//	txsMap[tx0.Payload.TxId] = tx0
//
//	txPool.EXPECT().GetTxsByTxIds([]string{tx0.Payload.TxId}).Return(txsMap, nil)
//	config := &config.ChainConfig{
//		ChainId: "chain1",
//		Crypto: &config.CryptoConfig{
//			Hash: "SHA256",
//		},
//	}
//	chainConf.EXPECT().ChainConfig().AnyTimes().Return(config)
//
//	principal := mock.NewMockPrincipal(ctl)
//	ac.EXPECT().LookUpResourceNameByTxType(tx0.Header.TxType).AnyTimes().Return("123", nil)
//	ac.EXPECT().CreatePrincipal("123", nil, nil).AnyTimes().Return(principal, nil)
//	ac.EXPECT().VerifyPrincipal(principal).AnyTimes().Return(true, nil)
//	verifyTxConf := &VerifierTxConfig{
//		Block:       block,
//		TxRWSetMap:  txRWSetMap,
//		TxResultMap: txResultMap,
//		Store:       store,
//		TxPool:      txPool,
//		Ac:          ac,
//		ChainConf:   chainConf,
//		Log:         log,
//	}
//	return NewVerifierTx(verifyTxConf), block
//}

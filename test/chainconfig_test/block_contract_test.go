/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package native_test

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"chainmaker.org/chainmaker/pb-go/v2/syscontract"

	native "chainmaker.org/chainmaker-go/test/chainconfig_test"
	apiPb "chainmaker.org/chainmaker/pb-go/v2/api"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var client apiPb.RpcNodeClient

func init() {
	conn, err := native.InitGRPCConnect(isTls)
	if err != nil {
		panic(err)
	}
	client = apiPb.NewRpcNodeClient(conn)
}

//tests for the following methods in block contract

//GetBlockByHeight
//GetBlockWithTxRWSetsByHeights
//GetBlockByHash
//GetBlockWithTxRWSetsByHash
//GetBlockTxId
//GetTxByTxId
//GetLastConfigBlock
//GetLastBlock
//GetFullBlockByHeight

// capitalize 't' to make it a unit test
func TestGetBlock(t *testing.T) {
	testHeight := -1
	height := int64(testHeight)
	blockByHeight, blockHashStringByHeight, txId := testGetBlockByHeight(t, client, height)
	blockByBlockHash := testGetBlockByHash(t, client, blockHashStringByHeight)

	if len(blockByBlockHash.Txs) == 0 {
		require.Equal(t, txId, "")
	}
	// checking the block returned from queryByHeight and queryByBlockHash are equal
	require.Equal(t, blockByHeight, blockByBlockHash)

	blockByTxId := testGetBlockByTxId(t, client, txId)
	if txId == "" {
		require.Nil(t, blockByTxId, nil)
	} else {
		require.Equal(t, blockByBlockHash, blockByTxId)
	}

	blockByHeightWithRWSets := testGetBlockWithTxRWSetsByHeights(t, client, height)
	require.Equal(t, blockByHeight, blockByHeightWithRWSets)

	blockByBlockHashWithRWSets := testGetBlockWithTxRWSetsByHash(t, client, blockHashStringByHeight)
	require.Equal(t, blockByHeight, blockByBlockHashWithRWSets)

	lastBlock := testGetLastBlock(t, client)
	lastBlockByHeight, _, _ := testGetBlockByHeight(t, client, int64(lastBlock.Header.BlockHeight))
	require.Equal(t, lastBlock, lastBlockByHeight)

	fullBlock := testGetFullBlockByHeight(t, client, height)
	block, _, _ := testGetBlockByHeight(t, client, height)
	require.Equal(t, fullBlock, block)

	lastConfigBlock := testGetLastConfigBlock(t, client)
	fmt.Printf("the last configured block has height of [%d]\n\n\n", lastConfigBlock.Header.BlockHeight)

	tx := testGetTxByTxId(t, client, txId)
	if tx != nil {
		block = testGetBlockByTxId(t, client, txId)
		require.Equal(t, block, blockByHeight)
	}

	fmt.Println("ALL TESTS PASSED!")
}

//returns Block hash and the txId of its first transaction
func testGetBlockByHeight(t *testing.T, client apiPb.RpcNodeClient, height int64) (*commonPb.Block, string, string) {
	fmt.Println("============ get block by height============")
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "blockHeight",
			Value: []byte(strconv.FormatInt(height, 10)),
		},
		{
			Key:   "withRWSet",
			Value: []byte("false"),
		},
	}

	sk, member := native.GetUserSK(1)
	resp, err := native.QueryRequest(sk, member, &client, &native.InvokeContractMsg{TxType: commonPb.TxType_QUERY_CONTRACT,
		ChainId: CHAIN1, ContractName: syscontract.SystemContract_CHAIN_QUERY.String(), MethodName: syscontract.ChainQueryFunction_GET_BLOCK_BY_HEIGHT.String(), Pairs: pairs})

	handleQueryReqError(err)
	//fmt.Printf("response: %v\n", resp)

	blockInfo := &commonPb.BlockInfo{}
	if err = proto.Unmarshal(resp.ContractResult.Result, blockInfo); err != nil {
		panic(err)
	}
	require.NotNil(t, blockInfo)

	//fmt.Println(blockInfo)

	var tx *commonPb.Transaction
	blockHash := blockInfo.Block.Header.BlockHash
	fmt.Println("blockHash", string(blockHash), hex.EncodeToString(blockHash))

	if len(blockInfo.GetBlock().Txs) > 0 {
		tx = blockInfo.GetBlock().Txs[0]
		if blockInfo.GetBlock().Header.BlockHeight > 0 {
			fmt.Printf("recv block [%d] => with (%d txs) from organization: %s\n", blockInfo.Block.Header.BlockHeight, len(blockInfo.Block.Txs), tx.Sender.Signer.OrgId)
		} else {
			fmt.Printf("recv block [%d] => with (%d txs)", blockInfo.Block.Header.BlockHeight, len(blockInfo.Block.Txs))
		}
		fmt.Println()
		fmt.Println()
		return blockInfo.Block, hex.EncodeToString(blockHash), tx.Payload.TxId
	} else {
		fmt.Printf("recv block [%d] => with (%d txs)\n", blockInfo.Block.Header.BlockHeight, len(blockInfo.Block.Txs))
	}

	fmt.Println()
	fmt.Println()

	return blockInfo.Block, hex.EncodeToString(blockHash), ""
}

func testGetBlockByHash(t *testing.T, client apiPb.RpcNodeClient, hash string) *commonPb.Block {
	fmt.Println("============ get block by hash ============")
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "blockHash",
			Value: []byte(hash),
		},
		{
			Key:   "withRWSet",
			Value: []byte("false"),
		},
	}

	sk, member := native.GetUserSK(1)
	resp, err := native.QueryRequest(sk, member, &client, &native.InvokeContractMsg{TxType: commonPb.TxType_QUERY_CONTRACT,
		ChainId: CHAIN1, ContractName: syscontract.SystemContract_CHAIN_QUERY.String(), MethodName: syscontract.ChainQueryFunction_GET_BLOCK_BY_HASH.String(), Pairs: pairs})

	handleQueryReqError(err)
	//fmt.Printf("response: %v\n", resp)

	blockInfo := &commonPb.BlockInfo{}
	if err = proto.Unmarshal(resp.ContractResult.Result, blockInfo); err != nil {
		panic(err)
	}
	require.NotNil(t, blockInfo)

	blockHash := blockInfo.Block.Header.BlockHash
	fmt.Println("blockHash", string(blockHash), hex.EncodeToString(blockHash))

	fmt.Println()
	fmt.Println()

	return blockInfo.Block
}

func testGetBlockByTxId(t *testing.T, client apiPb.RpcNodeClient, txId string) *commonPb.Block {
	fmt.Println("============================================================")
	fmt.Println("============================================================")
	fmt.Println("========get block by txId ", txId, "===============")
	fmt.Println("============================================================")
	fmt.Println("============================================================")
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "txId",
			Value: []byte(txId),
		},
		{
			Key:   "withRWSet",
			Value: []byte("false"),
		},
	}

	sk, member := native.GetUserSK(1)
	resp, err := native.QueryRequest(sk, member, &client, &native.InvokeContractMsg{TxType: commonPb.TxType_QUERY_CONTRACT,
		ChainId: CHAIN1, ContractName: syscontract.SystemContract_CHAIN_QUERY.String(), MethodName: syscontract.ChainQueryFunction_GET_BLOCK_BY_TX_ID.String(), Pairs: pairs})

	handleQueryReqError(err)
	//fmt.Printf("response: %v\n", resp)

	blockInfo := &commonPb.BlockInfo{}
	if resp == nil {
		return nil
	}
	if err = proto.Unmarshal(resp.ContractResult.Result, blockInfo); err != nil {
		panic(err)
	}
	require.NotNil(t, blockInfo)

	if blockInfo.Block == nil {
		return nil
	}

	blockHash := blockInfo.Block.Header.BlockHash
	fmt.Println("blockHash", string(blockHash), hex.EncodeToString(blockHash))

	fmt.Println()
	fmt.Println()
	return blockInfo.Block
}

func testGetBlockWithTxRWSetsByHeights(t *testing.T, client apiPb.RpcNodeClient, height int64) *commonPb.Block {
	fmt.Println("============================================================")
	fmt.Println("============================================================")
	fmt.Println("========get block with txRWsets by height ", height, "===============")
	fmt.Println("============================================================")
	fmt.Println("============================================================")
	fmt.Printf("\n============ get block with txRWsets by height [%d] ============\n", height)
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "blockHeight",
			Value: []byte(strconv.FormatInt(height, 10)),
		},
	}

	sk, member := native.GetUserSK(1)
	resp, err := native.QueryRequest(sk, member, &client, &native.InvokeContractMsg{TxType: commonPb.TxType_QUERY_CONTRACT,
		ChainId: CHAIN1, ContractName: syscontract.SystemContract_CHAIN_QUERY.String(), MethodName: syscontract.ChainQueryFunction_GET_BLOCK_WITH_TXRWSETS_BY_HEIGHT.String(), Pairs: pairs})

	handleQueryReqError(err)
	//fmt.Printf("response: %v\n", resp)

	blockInfo := &commonPb.BlockInfo{}
	if err = proto.Unmarshal(resp.ContractResult.Result, blockInfo); err != nil {
		panic(err)
	}
	require.NotNil(t, blockInfo)

	if blockInfo.Block == nil {
		return nil
	}

	blockHash := blockInfo.Block.Header.BlockHash
	fmt.Println("blockHash", string(blockHash), hex.EncodeToString(blockHash))

	fmt.Println()
	fmt.Println()
	return blockInfo.Block
}

func testGetBlockWithTxRWSetsByHash(t *testing.T, client apiPb.RpcNodeClient, hash string) *commonPb.Block {
	fmt.Println("============================================================")
	fmt.Println("============================================================")
	fmt.Println("========get block with txRWsets by hash ", hash, "===============")
	fmt.Println("============================================================")
	fmt.Println("============================================================")
	fmt.Printf("\n============ get block with txRWsets by hash [%s] ============\n", hash)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "blockHash",
			Value: []byte(hash),
		},
	}

	sk, member := native.GetUserSK(1)
	resp, err := native.QueryRequest(sk, member, &client, &native.InvokeContractMsg{TxType: commonPb.TxType_QUERY_CONTRACT,
		ChainId: CHAIN1, ContractName: syscontract.SystemContract_CHAIN_QUERY.String(), MethodName: syscontract.ChainQueryFunction_GET_BLOCK_WITH_TXRWSETS_BY_HASH.String(), Pairs: pairs})

	handleQueryReqError(err)
	//fmt.Printf("response: %v\n", resp)

	blockInfo := &commonPb.BlockInfo{}
	if err = proto.Unmarshal(resp.ContractResult.Result, blockInfo); err != nil {
		panic(err)
	}
	require.NotNil(t, blockInfo)

	if blockInfo.Block == nil {
		return nil
	}

	blockHash := blockInfo.Block.Header.BlockHash
	fmt.Println("blockHash", string(blockHash), hex.EncodeToString(blockHash))

	fmt.Println()
	fmt.Println()
	return blockInfo.Block
}

func testGetLastBlock(t *testing.T, client apiPb.RpcNodeClient) *commonPb.Block {
	fmt.Println("============================================================")
	fmt.Println("============================================================")
	fmt.Println("=======================get last block=======================")
	fmt.Println("============================================================")
	fmt.Println("============================================================")

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "withRWSet",
			Value: []byte("true"),
		},
	}

	sk, member := native.GetUserSK(1)
	resp, err := native.QueryRequest(sk, member, &client, &native.InvokeContractMsg{TxType: commonPb.TxType_QUERY_CONTRACT,
		ChainId: CHAIN1, ContractName: syscontract.SystemContract_CHAIN_QUERY.String(), MethodName: syscontract.ChainQueryFunction_GET_LAST_BLOCK.String(), Pairs: pairs})

	handleQueryReqError(err)
	//fmt.Printf("response: %v\n", resp)

	blockInfo := &commonPb.BlockInfo{}
	if err = proto.Unmarshal(resp.ContractResult.Result, blockInfo); err != nil {
		panic(err)
	}
	require.NotNil(t, blockInfo)

	if blockInfo.Block == nil {
		return nil
	}

	blockHash := blockInfo.Block.Header.BlockHash
	fmt.Println("blockHash", string(blockHash), hex.EncodeToString(blockHash))

	fmt.Println()
	fmt.Println()
	return blockInfo.Block
}

func testGetFullBlockByHeight(t *testing.T, client apiPb.RpcNodeClient, height int64) *commonPb.Block {
	fmt.Println("============================================================")
	fmt.Println("============================================================")
	fmt.Println("========get full block by height ", height, "===============")
	fmt.Println("============================================================")
	fmt.Println("============================================================")
	fmt.Printf("\n============ get full block by height [%d] ============\n", height)
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "blockHeight",
			Value: []byte(strconv.FormatInt(height, 10)),
		},
	}

	sk, member := native.GetUserSK(1)
	resp, err := native.QueryRequest(sk, member, &client, &native.InvokeContractMsg{TxType: commonPb.TxType_QUERY_CONTRACT,
		ChainId: CHAIN1, ContractName: syscontract.SystemContract_CHAIN_QUERY.String(), MethodName: syscontract.ChainQueryFunction_GET_FULL_BLOCK_BY_HEIGHT.String(), Pairs: pairs})

	handleQueryReqError(err)
	//fmt.Printf("response: %v\n", resp)

	blockInfo := &commonPb.BlockInfo{}
	if err = proto.Unmarshal(resp.ContractResult.Result, blockInfo); err != nil {
		panic(err)
	}
	require.NotNil(t, blockInfo)

	if blockInfo.Block == nil {
		return nil
	}

	blockHash := blockInfo.Block.Header.BlockHash
	fmt.Println("blockHash", string(blockHash), hex.EncodeToString(blockHash))

	fmt.Println()
	fmt.Println()
	return blockInfo.Block
}

func testGetLastConfigBlock(t *testing.T, client apiPb.RpcNodeClient) *commonPb.Block {
	fmt.Println("============================================================")
	fmt.Println("============================================================")
	fmt.Println("====================get last config block===================")
	fmt.Println("============================================================")
	fmt.Println("============================================================")
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "withRWSet",
			Value: []byte("true"),
		},
	}

	sk, member := native.GetUserSK(1)
	resp, err := native.QueryRequest(sk, member, &client, &native.InvokeContractMsg{TxType: commonPb.TxType_QUERY_CONTRACT,
		ChainId: CHAIN1, ContractName: syscontract.SystemContract_CHAIN_QUERY.String(), MethodName: syscontract.ChainQueryFunction_GET_LAST_CONFIG_BLOCK.String(), Pairs: pairs})

	handleQueryReqError(err)
	//fmt.Printf("response: %v\n", resp)

	blockInfo := &commonPb.BlockInfo{}
	if err = proto.Unmarshal(resp.ContractResult.Result, blockInfo); err != nil {
		panic(err)
	}
	require.NotNil(t, blockInfo)

	if blockInfo.Block == nil {
		return nil
	}

	blockHash := blockInfo.Block.Header.BlockHash
	fmt.Println("blockHash", string(blockHash), hex.EncodeToString(blockHash))

	fmt.Println()
	fmt.Println()
	return blockInfo.Block
}

func testGetTxByTxId(t *testing.T, client apiPb.RpcNodeClient, txId string) *commonPb.Transaction {
	fmt.Println("========================================================================================================")
	fmt.Println("========================================================================================================")
	fmt.Println("========get tx by txId ", txId, "===============")
	fmt.Println("========================================================================================================")
	fmt.Println("========================================================================================================")

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "txId",
			Value: []byte(txId),
		},
	}

	sk, member := native.GetUserSK(1)
	resp, err := native.QueryRequest(sk, member, &client, &native.InvokeContractMsg{TxType: commonPb.TxType_QUERY_CONTRACT,
		ChainId: CHAIN1, ContractName: syscontract.SystemContract_CHAIN_QUERY.String(), MethodName: syscontract.ChainQueryFunction_GET_TX_BY_TX_ID.String(), Pairs: pairs})

	handleQueryReqError(err)
	//fmt.Printf("response: %v\n", resp)

	result := &commonPb.TransactionInfo{}
	if err := proto.Unmarshal(resp.ContractResult.Result, result); err != nil {
		panic(err)
	}

	require.NotNil(t, result)
	return result.Transaction
}

func handleQueryReqError(err error) {
	if err != nil {
		statusErr, ok := status.FromError(err)
		if ok {
			if statusErr.Code() == codes.DeadlineExceeded {
				fmt.Println("WARN: client.call err: deadline")

			}
		}

		fmt.Printf("ERROR: client.call err: %v\n", err)
		return
	}
}

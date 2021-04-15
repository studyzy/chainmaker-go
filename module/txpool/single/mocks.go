/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package single

import (
	"crypto/x509/pkix"

	bcx509 "chainmaker.org/chainmaker-go/common/crypto/x509"
	"chainmaker.org/chainmaker-go/common/msgbus"
	acPb "chainmaker.org/chainmaker-go/pb/protogo/accesscontrol"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	configPb "chainmaker.org/chainmaker-go/pb/protogo/config"
	netPb "chainmaker.org/chainmaker-go/pb/protogo/net"
	storePb "chainmaker.org/chainmaker-go/pb/protogo/store"
	"chainmaker.org/chainmaker-go/protocol"
)

var errStr = "implement me"

type mockBlockChainStore struct {
	txs map[string]*commonPb.Transaction
}

func newMockBlockChainStore() *mockBlockChainStore {
	return &mockBlockChainStore{txs: make(map[string]*commonPb.Transaction)}
}

func (m *mockBlockChainStore) QuerySingle(contractName, sql string, values ...interface{}) (protocol.SqlRow, error) {
	panic(errStr)
}

func (m *mockBlockChainStore) QueryMulti(contractName, sql string, values ...interface{}) (protocol.SqlRows, error) {
	panic(errStr)
}

func (m *mockBlockChainStore) ExecDdlSql(contractName, sql string) error {
	panic(errStr)
}

func (m *mockBlockChainStore) BeginDbTransaction(txName string) (protocol.SqlDBTransaction, error) {
	panic(errStr)
}

func (m *mockBlockChainStore) GetDbTransaction(txName string) (protocol.SqlDBTransaction, error) {
	panic(errStr)
}

func (m *mockBlockChainStore) CommitDbTransaction(txName string) error {
	panic(errStr)
}

func (m *mockBlockChainStore) RollbackDbTransaction(txName string) error {
	panic(errStr)
}

func (m *mockBlockChainStore) InitGenesis(genesisBlock *storePb.BlockWithRWSet) error {
	panic(errStr)
}

func (m *mockBlockChainStore) PutBlock(block *commonPb.Block, txRWSets []*commonPb.TxRWSet) error {
	panic(errStr)
}

func (m *mockBlockChainStore) GetBlockByHash(blockHash []byte) (*commonPb.Block, error) {
	panic(errStr)
}

func (m *mockBlockChainStore) BlockExists(blockHash []byte) (bool, error) {
	panic(errStr)
}

func (m *mockBlockChainStore) GetBlock(height int64) (*commonPb.Block, error) {
	panic(errStr)
}

func (m *mockBlockChainStore) GetLastConfigBlock() (*commonPb.Block, error) {
	panic(errStr)
}

func (m *mockBlockChainStore) GetBlockByTx(txId string) (*commonPb.Block, error) {
	panic(errStr)
}

func (m *mockBlockChainStore) GetBlockWithRWSets(height int64) (*storePb.BlockWithRWSet, error) {
	panic(errStr)
}

func (m *mockBlockChainStore) GetTx(txId string) (*commonPb.Transaction, error) {
	tx := m.txs[txId]
	return tx, nil
}

func (m *mockBlockChainStore) TxExists(txId string) (bool, error) {
	_, exist := m.txs[txId]
	return exist, nil
}

func (m *mockBlockChainStore) GetTxConfirmedTime(txId string) (int64, error) {
	panic(errStr)
}

func (m *mockBlockChainStore) GetLastBlock() (*commonPb.Block, error) {
	panic(errStr)
}

func (m *mockBlockChainStore) ReadObject(contractName string, key []byte) ([]byte, error) {
	panic(errStr)
}

func (m *mockBlockChainStore) SelectObject(contractName string, startKey []byte, limit []byte) protocol.Iterator {
	panic(errStr)
}

func (m *mockBlockChainStore) GetTxRWSet(txId string) (*commonPb.TxRWSet, error) {
	panic(errStr)
}

func (m *mockBlockChainStore) GetTxRWSetsByHeight(height int64) ([]*commonPb.TxRWSet, error) {
	panic(errStr)
}

func (m *mockBlockChainStore) GetDBHandle(dbName string) protocol.DBHandle {
	panic(errStr)
}

func (m *mockBlockChainStore) Close() error {
	panic(errStr)
}

type mockMessageBus struct {
}

func newMockMessageBus() *mockMessageBus {
	return &mockMessageBus{}
}

func (m *mockMessageBus) Register(topic msgbus.Topic, sub msgbus.Subscriber) {
	// no impl
}

func (m *mockMessageBus) Publish(topic msgbus.Topic, payload interface{}) {
	// no impl
}

func (m *mockMessageBus) PublishSafe(topic msgbus.Topic, payload interface{}) {
	panic(errStr)
}

func (m *mockMessageBus) Close() {
	panic(errStr)
}

type mockAccessControlProvider struct {
}

func newMockAccessControlProvider() *mockAccessControlProvider {
	return &mockAccessControlProvider{}
}

func (m *mockAccessControlProvider) DeserializeMember(serializedMember []byte) (protocol.Member, error) {
	panic(errStr)
}

func (m *mockAccessControlProvider) GetHashAlg() string {
	panic(errStr)
}

func (m *mockAccessControlProvider) ValidateResourcePolicy(resourcePolicy *configPb.ResourcePolicy) bool {
	panic(errStr)
}

func (m *mockAccessControlProvider) LookUpResourceNameByTxType(txType commonPb.TxType) (string, error) {
	panic(errStr)
}

func (m *mockAccessControlProvider) CreatePrincipal(resourceName string, endorsements []*commonPb.EndorsementEntry, message []byte) (protocol.Principal, error) {
	panic(errStr)
}

func (m *mockAccessControlProvider) CreatePrincipalForTargetOrg(resourceName string, endorsements []*commonPb.EndorsementEntry, message []byte, targetOrgId string) (protocol.Principal, error) {
	panic(errStr)
}

func (m *mockAccessControlProvider) GetValidEndorsements(principal protocol.Principal) ([]*commonPb.EndorsementEntry, error) {
	panic(errStr)
}

func (m *mockAccessControlProvider) VerifyPrincipal(principal protocol.Principal) (bool, error) {
	panic(errStr)
}

func (m *mockAccessControlProvider) ValidateCRL(crl []byte) ([]*pkix.CertificateList, error) {
	panic(errStr)
}

func (m *mockAccessControlProvider) IsCertRevoked(certChain []*bcx509.Certificate) bool {
	panic(errStr)
}

func (m *mockAccessControlProvider) GetLocalOrgId() string {
	panic(errStr)
}

func (m *mockAccessControlProvider) GetLocalSigningMember() protocol.SigningMember {
	panic(errStr)
}

func (m *mockAccessControlProvider) NewMemberFromCertPem(orgId, certPEM string) (protocol.Member, error) {
	panic(errStr)
}

func (m *mockAccessControlProvider) NewMemberFromProto(serializedMember *acPb.SerializedMember) (protocol.Member, error) {
	panic(errStr)
}

func (m *mockAccessControlProvider) NewSigningMemberFromCertFile(orgId, prvKeyFile, password, certFile string) (protocol.SigningMember, error) {
	panic(errStr)
}

func (m *mockAccessControlProvider) NewSigningMember(member protocol.Member, privateKeyPem, password string) (protocol.SigningMember, error) {
	panic(errStr)
}

type mockNet struct {
}

func newMockNet() *mockNet {
	return &mockNet{}
}

func (m *mockNet) BroadcastMsg(msg []byte, msgType netPb.NetMsg_MsgType) error {
	panic(errStr)
}

func (m *mockNet) Subscribe(msgType netPb.NetMsg_MsgType, handler protocol.MsgHandler) error {
	panic(errStr)
}

func (m *mockNet) CancelSubscribe(msgType netPb.NetMsg_MsgType) error {
	panic(errStr)
}

func (m *mockNet) ConsensusBroadcastMsg(msg []byte, msgType netPb.NetMsg_MsgType) error {
	panic(errStr)
}

func (m *mockNet) ConsensusSubscribe(msgType netPb.NetMsg_MsgType, handler protocol.MsgHandler) error {
	panic(errStr)
}

func (m *mockNet) CancelConsensusSubscribe(msgType netPb.NetMsg_MsgType) error {
	panic(errStr)
}

func (m *mockNet) SendMsg(msg []byte, msgType netPb.NetMsg_MsgType, to ...string) error {
	panic(errStr)
}

func (m *mockNet) ReceiveMsg(msgType netPb.NetMsg_MsgType, handler protocol.MsgHandler) error {
	panic(errStr)
}

func (m *mockNet) Start() error {
	panic(errStr)
}

func (m *mockNet) Stop() error {
	panic(errStr)
}

func (m *mockNet) GetNodeUidByCertId(certId string) (string, error) {
	panic(errStr)
}

func (m *mockNet) GetChainNodesInfoProvider() protocol.ChainNodesInfoProvider {
	panic(errStr)
}

type mockChainConf struct {
}

func newMockChainConf() *mockChainConf {
	return &mockChainConf{}
}

func (m *mockChainConf) Init() error {
	panic(errStr)
}

func (m *mockChainConf) ChainConfig() *configPb.ChainConfig {
	return &configPb.ChainConfig{}
}

func (m *mockChainConf) GetChainConfigFromFuture(blockHeight int64) (*configPb.ChainConfig, error) {
	panic(errStr)
}

func (m *mockChainConf) GetChainConfigAt(blockHeight int64) (*configPb.ChainConfig, error) {
	panic(errStr)
}

func (m *mockChainConf) GetConsensusNodeIdList() ([]string, error) {
	panic(errStr)
}

func (m *mockChainConf) CompleteBlock(block *commonPb.Block) error {
	panic(errStr)
}

func (m *mockChainConf) AddWatch(w protocol.Watcher) {
	panic(errStr)
}

func (m *mockChainConf) AddVmWatch(w protocol.VmWatcher) {
	panic(errStr)
}

/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chainedbft

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"chainmaker.org/chainmaker-go/accesscontrol"
	"chainmaker.org/chainmaker-go/consensus/chainedbft/consensus_mock"
	"chainmaker.org/chainmaker-go/consensus/chainedbft/liveness"
	"chainmaker.org/chainmaker-go/consensus/chainedbft/utils"
	"chainmaker.org/chainmaker/chainconf/v2"
	"chainmaker.org/chainmaker/common/v2/msgbus"
	"chainmaker.org/chainmaker/localconf/v2"
	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/consensus/chainedbft"
	systemPb "chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/protocol/v2/mock"
	"chainmaker.org/chainmaker/protocol/v2/test"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

const LoadConfigErrorFmt = "load config error:%v"

var configPath = "../../../config/"

var chainedBftNode []*ConsensusChainedBftImpl
var nodeConfigEnv = []string{"wx-org1", "wx-org2", "wx-org3", "wx-org4"}
var coreNode []*consensus_mock.MockCoreEngine
var nodeLocalConf []*localconf.CMConfig
var nodeChainConf []*chainconf.ChainConf

var nodeLists = []string{
	"QmcQHCuAXaFkbcsPUj7e37hXXfZ9DdN7bozseo5oX4qiC4",
	"QmeyNRs2DwWjcHTpcVHoUSaDAAif4VQZ2wQDQAUNDP33gH",
	"QmXf6mnQDBR9aHauRmViKzSuZgpumkn7x6rNxw1oqqRr45",
	"QmRRWXJpAVdhFsFtd9ah5F4LDQWFFBDVKpECAF8hssqj6H",
}

func initChainMakerConfig(path string) (*localconf.CMConfig, error) {

	cmviper := viper.New()
	lconfig := &localconf.CMConfig{}
	cmviper.SetConfigFile(path + "/" + "chainmaker.yml")
	if err := cmviper.MergeInConfig(); err != nil {
		println("init cfg error:", err)
		return nil, err
	}

	if err := cmviper.Unmarshal(lconfig); err != nil {
		return nil, err
	}

	return lconfig, nil
}

//TestLoadConfig test load config
//func TestLoadConfig(t *testing.T) {
//	localpath := configPath + nodeConfigEnv[0]
//
//	//init config
//	lf, err := initChainMakerConfig(localpath)
//	if err != nil {
//		panic(fmt.Errorf(LoadConfigErrorFmt, err))
//	}
//	nodeLocalConf = append(nodeLocalConf, lf)
//	assert.Equal(t, "wx-org1.chainmaker.org", lf.NodeConfig.OrgId)
//}

func initChainConf(filePath string, t *testing.T) (*chainconf.ChainConf, error) {
	pbcf, err := chainconf.Genesis(filePath)
	if err != nil {
		panic(fmt.Errorf("genesis error %v", err))
	}
	pbcfbyte, _ := proto.Marshal(pbcf)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := mock.NewMockBlockchainStore(ctrl)
	store.EXPECT().ReadObject(systemPb.SystemContract_CHAIN_CONFIG.String(),
		[]byte(systemPb.SystemContract_CHAIN_CONFIG.String())).Return(pbcfbyte, nil).AnyTimes()
	nodecf, _ := chainconf.NewChainConf(
		chainconf.WithBlockchainStore(store),
	)
	nodecf.Init()
	return nodecf, nil
}

//TestLoadChainConfig test load chainconfig
//func TestLoadChainConfig(t *testing.T) {
//	localpath := configPath + nodeConfigEnv[0]
//
//	//init config
//	cf, err := initChainConf(localpath+"/chainconfig/bc1.yml", t)
//	if err != nil {
//		panic(fmt.Errorf(LoadConfigErrorFmt, err))
//	}
//	assert.Equal(t, 4, int(cf.ChainConfig().Consensus.Type))
//}

func createMsgbusTotal() map[string]msgbus.MessageBus {
	buses := make(map[string]msgbus.MessageBus)
	for i := 0; i < len(nodeChainConf); i++ {
		mb := msgbus.NewMessageBus()
		buses[nodeLists[i]] = mb
	}
	return buses
}

func createNodeConf(t *testing.T, index int) {
	localpath := configPath + nodeConfigEnv[index]
	//init config
	lf, err := initChainMakerConfig(localpath)
	if err != nil {
		panic(fmt.Errorf(LoadConfigErrorFmt, err))
	}
	nodeLocalConf = append(nodeLocalConf, lf)

	//init chainconf
	cf, err := initChainConf(localpath+"/chainconfig/bc1.yml", t)
	if err != nil {
		panic(fmt.Errorf(LoadConfigErrorFmt, err))
	}
	nodeChainConf = append(nodeChainConf, cf)
}

func createCertNodesTotal() map[string]string {
	nodecert := make(map[string]string)
	for i := 0; i < len(nodeLocalConf); i++ {
		localpath := configPath + nodeConfigEnv[i]
		lf := nodeLocalConf[i]
		certPEM, _ := ioutil.ReadFile(filepath.Join(localpath, lf.NetConfig.TLSConfig.CertFile))
		skFile := lf.NodeConfig.PrivKeyFile
		confDir := filepath.Dir(localconf.ConfigFilepath)
		if !filepath.IsAbs(skFile) {
			skFile = filepath.Join(confDir, skFile)
		}
		certFile := lf.NodeConfig.CertFile
		if !filepath.IsAbs(certFile) {
			certFile = filepath.Join(confDir, certFile)
		}
		acLog := &test.GoLogger{}
		ac, _ := accesscontrol.NewAccessControlWithChainConfig(nodeChainConf[i], lf.NodeConfig.OrgId, nil, acLog)
		pbMember := &pbac.Member{
			OrgId:      lf.NodeConfig.OrgId,
			MemberType: pbac.MemberType_CERT_HASH,
			MemberInfo: certPEM,
		}

		member, _ := ac.NewMember(pbMember)
		// member, _ := accesscontrol.MockAccessControl().NewMember(lf.NodeConfig.OrgId, string(certPEM))
		nodecert[member.GetMemberId()] = nodeLists[i]
	}
	return nodecert
}

func InitGenesis(chainid string) *commonPb.Block {
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:     chainid,
			BlockHeight: 0,
			Signature:   []byte(""),
			BlockHash:   []byte(""),
		},
		Dag: &commonPb.DAG{},
		Txs: []*commonPb.Transaction{
			{
				Payload: &commonPb.Payload{
					ChainId: chainid,
				},
			},
		},
	}

	blockHash := []byte(fmt.Sprintf("%s-%d-%s", chainid, 0, time.Now()))
	block.Header.BlockHash = blockHash[:]
	return block
}

func createNode(t *testing.T, index int, chainid string,
	msgBus map[string]msgbus.MessageBus, certnodes map[string]string,
	isCreateBlock bool, genesis *commonPb.Block) {
	// singer organization
	//localpath := configPath + nodeConfigEnv[index]
	//
	//lf := nodeLocalConf[index]
	//cf := nodeChainConf[index]
	//nodeConfig := lf.NodeConfig
	//skFile := lf.NodeConfig.PrivKeyFile
	//confDir := filepath.Dir(localconf.ConfigFilepath)
	//if !filepath.IsAbs(skFile) {
	//	skFile = filepath.Join(confDir, skFile)
	//}
	//certFile := lf.NodeConfig.CertFile
	//if !filepath.IsAbs(certFile) {
	//	certFile = filepath.Join(confDir, certFile)
	//}
	//ac, err := accesscontrol.NewAccessControlWithChainConfig(skFile, lf.NodeConfig.PrivKeyPassword, certFile, cf, nodeConfig.OrgId, nil)
	//if err != nil {
	//	panic(fmt.Errorf("init org err%v", err))
	//}
	//signer, err := ac.NewSigningMemberFromCertFile(cf.ChainConfig().AuthType,
	//	filepath.Join(localpath, nodeConfig.PrivKeyFile), nodeConfig.PrivKeyPassword,
	//	filepath.Join(localpath, nodeConfig.CertFile))
	//if err != nil {
	//	panic(fmt.Errorf("init signer err%v", err))
	//}
	//
	//ledger := consensus_mock.NewLedger(chainid, genesis)
	//store := consensus_mock.NewMockMockBlockchainStore(genesis, cf)
	//
	//ce := consensus_mock.NewMockCoreEngine(t, nodeLists[index], chainid, msgBus[nodeLists[index]], ledger, store, isCreateBlock)
	//net := consensus_mock.NewMockProtocolNetService(certnodes)

	//node, _ := New(chainid, nodeLists[index], signer, ac, ce.Ledger,
	//	ce.Proposer, ce.Verifer, ce.Committer, net, store, msgBus[nodeLists[index]], cf, nil)
	//coreNode = append(coreNode, ce)
	//chainedBftNode = append(chainedBftNode, node)
	//consensus_mock.NewMockNet(nodeLists[index], msgBus)
}

func initNode(t *testing.T, isCreateBlock bool) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chainid := "TestConsensusChainedBftImpl"

	nodeLocalConf = make([]*localconf.CMConfig, 0)
	nodeChainConf = make([]*chainconf.ChainConf, 0)
	coreNode = make([]*consensus_mock.MockCoreEngine, 0)
	chainedBftNode = make([]*ConsensusChainedBftImpl, 0)

	for i := 0; i < len(nodeLists); i++ {
		createNodeConf(t, i)
	}

	buses := createMsgbusTotal()
	certnodes := createCertNodesTotal()
	genesis := InitGenesis(chainid)
	for i := 0; i < len(nodeLists); i++ {
		createNode(t, i, chainid, buses, certnodes, isCreateBlock, genesis)
	}

}

func startNode(t *testing.T) {
	for _, core := range coreNode {
		go core.Loop()
	}
	time.Sleep(1 * time.Second)
	t.Log("start chainedBft...")
	for _, chainedBft := range chainedBftNode {
		chainedBft.Start()
	}
}

//func TestConsensusChainedBftImpl_FourNode(t *testing.T) {
//	initNode(t, true)
//	startNode(t)
//
//	var wg sync.WaitGroup
//	wg.Add(len(coreNode))
//	for _, ce := range coreNode {
//		go func(ce *consensus_mock.MockCoreEngine) {
//			defer wg.Done()
//			timer := time.NewTimer(20 * time.Second)
//
//			for {
//				select {
//				case <-timer.C:
//					t.Logf("ce %v got to timeout, ce height %v", ce.GetID(), ce.GetHeight()-1)
//					return
//				}
//			}
//		}(ce)
//	}
//
//	wg.Wait()
//}

func initOneNode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chainid := "TestConsensusChainedBftImplOneNode"

	nodeLocalConf = make([]*localconf.CMConfig, 0)
	nodeChainConf = make([]*chainconf.ChainConf, 0)
	coreNode = make([]*consensus_mock.MockCoreEngine, 0)
	chainedBftNode = make([]*ConsensusChainedBftImpl, 0)

	createNodeConf(t, 0)
	buses := createMsgbusTotal()
	certnodes := createCertNodesTotal()
	genesis := InitGenesis(chainid)

	createNode(t, 0, chainid, buses, certnodes, false, genesis)
}

func cfgChainedBftNode(t *testing.T) {
	t.Log("cfg chainedBft...")
	for _, cbi := range chainedBftNode {
		cbi.logger.Infof("service selfIndexInEpoch %v started consensus.chainedbft", cbi.selfIndexInEpoch)
		var err error
		cbi.chainStore, err = openChainStore(cbi.ledgerCache, cbi.blockCommitter, cbi.store, cbi, cbi.logger)
		if err != nil {
			cbi.logger.Errorf("failed to new consensus service, err %v", err)
		}
		//cbi.smr.safetyRules = safetyrules.NewSafetyRules(cbi.logger, cbi.chainStore.blockPool)
		cbi.commitHeight = cbi.chainStore.getCommitHeight()
		cbi.createEpoch(cbi.commitHeight)
		epoch := cbi.nextEpoch
		cbi.smr.initCommittee(epoch.useValidators)
		cbi.msgPool = epoch.msgPool
		cbi.selfIndexInEpoch = epoch.index
		cbi.smr.paceMaker = liveness.NewPacemaker(cbi.logger, cbi.selfIndexInEpoch, 0, epoch.epochId, cbi.timerService)
		cbi.smr.forwardNewHeightIfNeed()
		cbi.nextEpoch = nil

		cbi.msgbus.Register(msgbus.ProposedBlock, cbi)
		cbi.msgbus.Register(msgbus.RecvConsensusMsg, cbi)
		cbi.msgbus.Register(msgbus.BlockInfo, cbi)
	}
}

//func TestSignAndVerifyMsg(t *testing.T) {
//	initOneNode(t)
//	cfgChainedBftNode(t)
//	proposalBlock := coreNode[0].Proposer.CreateBlock(1, nil)
//	//proposal
//	payload := chainedBftNode[0].constructProposal(proposalBlock, chainedBftNode[0].smr.getHeight(),
//		chainedBftNode[0].smr.getCurrentLevel(), 0)
//	assert.NotNil(t, payload)
//
//	msg := &chainedbft.ConsensusMsg{
//		Payload:   payload,
//		SignEntry: nil,
//	}
//
//	err := utils.SignConsensusMsg(msg, chainedBftNode[0].chainConf.ChainConfig().Crypto.Hash, chainedBftNode[0].singer)
//	assert.Nil(t, err)
//	assert.NotNil(t, msg.SignEntry)
//	assert.NotNil(t, msg.SignEntry.Signer)
//	assert.NotNil(t, msg.SignEntry.Signature)
//
//	err = utils.VerifyConsensusMsgSign(msg, chainedBftNode[0].accessControlProvider)
//	assert.Nil(t, err)
//}

//testInsertProposal tests InsertProposal function
func testInsertProposal(height uint64, round int64,
	msg *chainedbft.ConsensusMsg, t *testing.T) {
	for i := 0; i < len(chainedBftNode); i++ {
		inserted, _ := chainedBftNode[i].msgPool.InsertProposal(uint64(height),
			uint64(round), msg)
		assert.Equal(t, true, inserted)
	}
}

//testGetProposal tests GetProposal function
func testGetProposal(height uint64, round int64, t *testing.T) {
	for i := 0; i < len(chainedBftNode); i++ {
		msg := chainedBftNode[i].msgPool.GetProposal(uint64(height), uint64(round))
		assert.NotNil(t, msg)
		msg = chainedBftNode[i].msgPool.GetProposal(uint64(height+1), uint64(round))
		assert.Nil(t, msg)
	}
}

//signMsg signs payload with given key, and returns wrapped consensus message
func signMsg(payload *chainedbft.ConsensusPayload, singer protocol.SigningMember, chainconf protocol.ChainConf) *chainedbft.ConsensusMsg {
	consensusMessage := &chainedbft.ConsensusMsg{
		Payload:   payload,
		SignEntry: nil,
	}
	if singer == nil {
		panic(fmt.Errorf("signer nil error"))
	}
	err := utils.SignConsensusMsg(consensusMessage, chainconf.ChainConfig().Crypto.Hash, singer)
	if err != nil {
		panic(fmt.Errorf("sign consensus msg error %v", err))
	}
	return consensusMessage
}

//testEndorseBlock tests endorse proposal block
//func testEndorseBlock(height uint64, level int64,
//	block *commonPb.Block, t *testing.T) {
//	endorsePayload0 := chainedBftNode[0].constructVote(uint64(height), uint64(level), 0, block)
//	assert.NotNil(t, endorsePayload0)
//	endorseMsg0 := signMsg(endorsePayload0, chainedBftNode[0].singer, chainedBftNode[0].chainConf)
//	chainedBftNode[0].msgPool.InsertVote(uint64(height), uint64(level), endorseMsg0)
//
//	for i := 1; i < len(chainedBftNode); i++ {
//		endorsePayload := chainedBftNode[i].constructVote(uint64(height), uint64(level), 0, block)
//		assert.NotNil(t, endorsePayload)
//		endorseMsg := signMsg(endorsePayload, chainedBftNode[i].singer, chainedBftNode[i].chainConf)
//		chainedBftNode[0].msgPool.InsertVote(uint64(height), uint64(level), endorseMsg)
//	}
//
//	ok := chainedBftNode[0].msgPool.CheckAnyVotes(uint64(height), uint64(level))
//	assert.Equal(t, true, ok)
//
//	blockID, emptyBlock, ok := chainedBftNode[0].msgPool.CheckVotesDone(uint64(height), uint64(level))
//	assert.Equal(t, true, ok)
//	assert.NotNil(t, blockID)
//	assert.Equal(t, false, emptyBlock)
//	assert.Equal(t, blockID, block.Header.BlockHash)
//}

//func testEndorseNil(height uint64, level uint64, block *commonPb.Block, t *testing.T) {
//	endorsePayload0 := chainedBftNode[0].constructVote(height, level, 0, block)
//	assert.NotNil(t, endorsePayload0)
//	endorseMsg0 := signMsg(endorsePayload0, chainedBftNode[0].singer, chainedBftNode[0].chainConf)
//	chainedBftNode[0].msgPool.InsertVote(height, level, endorseMsg0)
//
//	for i := 1; i < len(chainedBftNode); i++ {
//		endorsePayload := chainedBftNode[i].constructVote(height, level, 0, nil)
//		assert.NotNil(t, endorsePayload)
//		endorseMsg := signMsg(endorsePayload, chainedBftNode[i].singer, chainedBftNode[i].chainConf)
//		chainedBftNode[0].msgPool.InsertVote(height, level, endorseMsg)
//	}
//
//	ok := chainedBftNode[0].msgPool.CheckAnyVotes(height, level)
//	assert.Equal(t, true, ok)
//
//	blockID, emptyBlock, ok := chainedBftNode[0].msgPool.CheckVotesDone(height,
//		level)
//	assert.Equal(t, true, ok)
//	assert.Nil(t, blockID)
//	assert.Equal(t, true, emptyBlock)
//}

//func TestMsgPool(t *testing.T) {
//	initNode(t, false)
//	cfgChainedBftNode(t)
//
//	proposalBlock := coreNode[0].Proposer.CreateBlock(1, nil)
//	//proposal
//	payload := chainedBftNode[0].constructProposal(proposalBlock, chainedBftNode[0].smr.getHeight(),
//		chainedBftNode[0].smr.getCurrentLevel(), 0)
//	assert.NotNil(t, payload)
//
//	msg := signMsg(payload, chainedBftNode[0].singer, chainedBftNode[0].chainConf)
//
//	proposal := payload.GetProposalMsg()
//	assert.NotNil(t, proposal)
//
//	testInsertProposal(int64(proposal.ProposalData.Height), int64(proposal.ProposalData.Level), msg, t)
//	testGetProposal(int64(proposal.ProposalData.Height), int64(proposal.ProposalData.Level), t)
//	//endorse valid block
//	testEndorseBlock(int64(proposal.ProposalData.Height), int64(proposal.ProposalData.Level),
//		proposal.ProposalData.Block, t)
//
//	for i := 0; i < len(chainedBftNode); i++ {
//		chainedBftNode[i].msgPool.Cleanup()
//	}
//
//	// Vote nil block
//	payload = chainedBftNode[0].constructProposal(proposalBlock, chainedBftNode[0].smr.getHeight(),
//		chainedBftNode[0].smr.getCurrentLevel(), 0)
//	assert.NotNil(t, payload)
//
//	msg = signMsg(payload, chainedBftNode[0].singer, chainedBftNode[0].chainConf)
//
//	proposal = payload.GetProposalMsg()
//	assert.NotNil(t, proposal)
//	for i := 0; i < len(chainedBftNode); i++ {
//		inserted, _ := chainedBftNode[i].msgPool.InsertProposal(proposal.ProposalData.Height, proposal.ProposalData.Level, msg)
//		assert.Equal(t, true, inserted)
//	}
//
//	for i := 0; i < len(chainedBftNode); i++ {
//		msg := chainedBftNode[i].msgPool.GetProposal(proposal.ProposalData.Height, proposal.ProposalData.Level)
//		assert.NotNil(t, msg)
//		msg = chainedBftNode[i].msgPool.GetProposal(proposal.ProposalData.Height+1, proposal.ProposalData.Level)
//		assert.Nil(t, msg)
//	}
//
//	testEndorseNil(proposal.ProposalData.GetHeight(), proposal.ProposalData.GetLevel(), proposalBlock, t)
//
//	for i := 0; i < len(chainedBftNode); i++ {
//		chainedBftNode[i].msgPool.Cleanup()
//	}
//}

//func TestConsState(t *testing.T) {
//	initOneNode(t)
//	cfgChainedBftNode(t)
//
//	var err error
//	chainedBftNode[0].chainStore, err = openChainStore(chainedBftNode[0].ledgerCache,
//		chainedBftNode[0].blockCommitter, chainedBftNode[0].store, chainedBftNode[0], chainedBftNode[0].logger)
//	if err != nil {
//		panic(err)
//	}
//	chainedBftNode[0].smr.forwardNewHeightIfNeed()
//	assert.Equal(t, chainedbft.ConsStateType_NEW_HEIGHT, chainedBftNode[0].smr.state)
//
//	chainedBftNode[0].processNewHeight(chainedBftNode[0].smr.getHeight(), chainedBftNode[0].smr.getCurrentLevel())
//	assert.Equal(t, chainedbft.ConsStateType_NEW_LEVEL, chainedBftNode[0].smr.state)
//
//	chainedBftNode[0].processNewLevel(chainedBftNode[0].smr.getHeight(), chainedBftNode[0].smr.getCurrentLevel()+1)
//	assert.Equal(t, chainedbft.ConsStateType_PROPOSE, chainedBftNode[0].smr.state)
//
//}

//TestProposalTimeout tests proposal timeout into endorse state
//func TestProposalTimeout(t *testing.T) {
//	initOneNode(t)
//	startNode(t)
//
//	cs := chainedBftNode[0]
//	duration := timeservice.GetEventTimeout(timeservice.PROPOSAL_BLOCK_TIMEOUT, int32(cs.smr.getCurrentLevel()))
//	duration = duration +
//		timeservice.GetEventTimeout(timeservice.VOTE_BLOCK_TIMEOUT, int32(cs.smr.getCurrentLevel()))
//	time.Sleep(duration)
//
//	assert.Equal(t, chainedbft.ConsStateType_PROPOSE, cs.smr.state)
//
//	cs.Stop()
//}

//func TestValidProposal(t *testing.T) {
//	initNode(t, false)
//	cfgChainedBftNode(t)
//
//	err := chainedBftNode[0].Start()
//	assert.Nil(t, err)
//
//	chainedBftNode[1].chainStore, err = openChainStore(chainedBftNode[1].ledgerCache,
//		chainedBftNode[1].blockCommitter, chainedBftNode[1].store, chainedBftNode[1], chainedBftNode[1].logger)
//	if err != nil {
//		panic(err)
//	}
//	chainedBftNode[1].smr.forwardNewHeightIfNeed()
//
//	//init proposedblock
//	proposalBlock := coreNode[1].Proposer.CreateBlock(1, nil)
//	payload := chainedBftNode[1].constructProposal(proposalBlock, chainedBftNode[1].smr.getHeight(),
//		chainedBftNode[1].smr.getCurrentLevel(), 0)
//	assert.NotNil(t, payload)
//
//	chainedBftNode[1].signAndSendToPeer(payload, chainedBftNode[0].selfIndexInEpoch)
//
//	time.Sleep(3 * time.Second)
//
//	assert.Equal(t, chainedbft.ConsStateType_PROPOSE, chainedBftNode[0].smr.state)
//
//	chainedBftNode[0].Stop()
//}

//func TestInvalidValidator(t *testing.T) {
//	initOneNode(t)
//	cfgChainedBftNode(t)
//
//	var err error
//	chainedBftNode[0].chainStore, err = openChainStore(chainedBftNode[0].ledgerCache,
//		chainedBftNode[0].blockCommitter, chainedBftNode[0].store, chainedBftNode[0], chainedBftNode[0].logger)
//	if err != nil {
//		panic(err)
//	}
//	chainedBftNode[0].smr.forwardNewHeightIfNeed()
//
//	origin := chainedBftNode[0].selfIndexInEpoch
//	chainedBftNode[0].selfIndexInEpoch = math.MaxInt32
//
//	chainedBftNode[0].processNewHeight(chainedBftNode[0].smr.getHeight(), chainedBftNode[0].smr.getCurrentLevel())
//	assert.Equal(t, chainedbft.ConsStateType_NEW_HEIGHT, chainedBftNode[0].smr.state)
//	chainedBftNode[0].selfIndexInEpoch = origin
//}

//func TestProcessNewHeight(t *testing.T) {
//	initOneNode(t)
//	cfgChainedBftNode(t)
//
//	var err error
//	chainedBftNode[0].chainStore, err = openChainStore(chainedBftNode[0].ledgerCache,
//		chainedBftNode[0].blockCommitter, chainedBftNode[0].store, chainedBftNode[0], chainedBftNode[0].logger)
//	if err != nil {
//		panic(err)
//	}
//	chainedBftNode[0].smr.forwardNewHeightIfNeed()
//
//	assert.Equal(t, chainedbft.ConsStateType_NEW_HEIGHT, chainedBftNode[0].smr.state)
//
//	//mismatch height
//	chainedBftNode[0].processNewHeight(chainedBftNode[0].smr.getHeight()+1,
//		chainedBftNode[0].smr.getCurrentLevel())
//	assert.Equal(t, chainedbft.ConsStateType_NEW_HEIGHT, chainedBftNode[0].smr.state)
//}

//TestProcessNewRound tests processNewRound function
//func TestProcessNewLevel(t *testing.T) {
//	initOneNode(t)
//	cfgChainedBftNode(t)
//
//	var err error
//	chainedBftNode[0].chainStore, err = openChainStore(chainedBftNode[0].ledgerCache,
//		chainedBftNode[0].blockCommitter, chainedBftNode[0].store, chainedBftNode[0], chainedBftNode[0].logger)
//	if err != nil {
//		panic(err)
//	}
//	chainedBftNode[0].smr.forwardNewHeightIfNeed()
//
//	cs := chainedBftNode[0]
//	assert.Equal(t, chainedbft.ConsStateType_NEW_HEIGHT, cs.smr.state)
//	cs.smr.state = chainedbft.ConsStateType_NEW_LEVEL
//
//	//mismatch height
//	cs.processNewLevel(cs.smr.getHeight()+1, cs.smr.getCurrentLevel())
//	assert.Equal(t, chainedbft.ConsStateType_NEW_LEVEL, cs.smr.state)
//}

//func TestInitTimeOutConfig(t *testing.T) {
//	config := &configpb.ChainConfig{
//		Consensus: &configpb.ConsensusConfig{
//			Type:      consensuspb.ConsensusType_HOTSTUFF,
//			ExtConfig: nil,
//		},
//	}
//	impl := &ConsensusChainedBftImpl{}
//
//	// 1. no content in config
//	impl.initTimeOutConfig(config)
//	require.EqualValues(t, timeservice.RoundTimeout, 6000*time.Millisecond)
//	require.EqualValues(t, timeservice.RoundTimeoutInterval, 500*time.Millisecond)
//	require.EqualValues(t, timeservice.ProposerTimeout, 2000*time.Millisecond)
//	require.EqualValues(t, timeservice.ProposerTimeoutInterval, 500*time.Millisecond)
//
//	// 2. update chainConfig
//	config.Consensus.ExtConfig = append(config.Consensus.ExtConfig, &commonPb.KeyValuePair{Key: timeservice.RoundTimeoutMill, Value: "10"})
//	config.Consensus.ExtConfig = append(config.Consensus.ExtConfig, &commonPb.KeyValuePair{Key: timeservice.RoundTimeoutIntervalMill, Value: "100"})
//	config.Consensus.ExtConfig = append(config.Consensus.ExtConfig, &commonPb.KeyValuePair{Key: timeservice.ProposerTimeoutMill, Value: "1000"})
//	config.Consensus.ExtConfig = append(config.Consensus.ExtConfig, &commonPb.KeyValuePair{Key: timeservice.ProposerTimeoutIntervalMill, Value: "10000"})
//	impl.initTimeOutConfig(config)
//	require.EqualValues(t, timeservice.RoundTimeout, 10*time.Millisecond)
//	require.EqualValues(t, timeservice.RoundTimeoutInterval, 100*time.Millisecond)
//	require.EqualValues(t, timeservice.ProposerTimeout, 1000*time.Millisecond)
//	require.EqualValues(t, timeservice.ProposerTimeoutInterval, 10000*time.Millisecond)
//
//	// 3. invalid config
//	config.Consensus.ExtConfig = append(config.Consensus.ExtConfig, &commonPb.KeyValuePair{Key: timeservice.RoundTimeoutMill, Value: "-1"})
//	config.Consensus.ExtConfig = append(config.Consensus.ExtConfig, &commonPb.KeyValuePair{Key: timeservice.RoundTimeoutIntervalMill, Value: fmt.Sprintf("%d", math.MaxInt64/time.Millisecond+1)})
//	config.Consensus.ExtConfig = append(config.Consensus.ExtConfig, &commonPb.KeyValuePair{Key: timeservice.ProposerTimeoutMill, Value: "0"})
//	config.Consensus.ExtConfig = append(config.Consensus.ExtConfig, &commonPb.KeyValuePair{Key: timeservice.ProposerTimeoutIntervalMill, Value: fmt.Sprintf("%d", math.MaxInt64/time.Millisecond)})
//	impl.initTimeOutConfig(config)
//	require.EqualValues(t, timeservice.RoundTimeout, 10*time.Millisecond)
//	require.EqualValues(t, timeservice.RoundTimeoutInterval, 100*time.Millisecond)
//	require.EqualValues(t, timeservice.ProposerTimeout, 1000*time.Millisecond)
//	require.EqualValues(t, int64(timeservice.ProposerTimeoutInterval), math.MaxInt64/time.Millisecond*time.Millisecond)
//}

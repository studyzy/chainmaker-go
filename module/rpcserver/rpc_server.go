/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

/*
	GRPC Server framwork
*/
package rpcserver

import (
	"chainmaker.org/chainmaker-go/blockchain"
	"chainmaker.org/chainmaker/common/ca"
	"chainmaker.org/chainmaker/common/crypto"
	"chainmaker.org/chainmaker/common/crypto/hash"
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/monitor"
	apiPb "chainmaker.org/chainmaker-go/pb/protogo/api"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"net"
	"sort"
	"strings"
	"time"
)

// RPCServer struct define
type RPCServer struct {
	grpcServer                 *grpc.Server
	chainMakerServer           *blockchain.ChainMakerServer
	log                        *logger.CMLogger
	ctx                        context.Context
	cancel                     context.CancelFunc
	curChainConfTrustRootsHash string
	isShutdown                 bool
}

// prom monitor define
var (
	mRecv     *prometheus.CounterVec
	mRecvTime *prometheus.HistogramVec
)

const (
	// rpc ratelimit config
	rateLimitDefaultTokenPerSecond  = 10000
	rateLimitDefaultTokenBucketSize = 10000

	// subscriber ratelimit config
	subscriberRateLimitDefaultTokenPerSecond  = 1000
	subscriberRateLimitDefaultTokenBucketSize = 1000
)

// TLS Mode
const (
	TLS_MODE_DISABLE = "disable"
	TLS_MODE_ONEWAY  = "oneway"
	TLS_MODE_TWOWAY  = "twoway"
)

// NewRPCServer - new RPCServer object
func NewRPCServer(chainMakerServer *blockchain.ChainMakerServer) (*RPCServer, error) {

	server, err := newGrpc(chainMakerServer)
	if err != nil {
		return nil, fmt.Errorf("new grpc server failed, %s", err.Error())
	}

	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		mRecv = monitor.NewCounterVec(monitor.SUBSYSTEM_GRPC, "grpc_msg_received_total",
			"Total number of RPC messages received on the server.",
			"grpc_service", "grpc_method")
		mRecvTime = monitor.NewHistogramVec(monitor.SUBSYSTEM_GRPC, "grpc_msg_received_time",
			"The time of RPC messages received on the server.",
			[]float64{0.005, 0.01, 0.015, 0.05, 0.1, 1, 10},
			"grpc_service", "grpc_method")
	}

	var log = logger.GetLogger(logger.MODULE_RPC)

	return &RPCServer{
		grpcServer:       server,
		chainMakerServer: chainMakerServer,
		log:              log,
	}, nil
}

// Start - start RPCServer
func (s *RPCServer) Start() error {
	var (
		err error
	)

	s.ctx, s.cancel = context.WithCancel(context.Background())

	s.isShutdown = false

	if s.curChainConfTrustRootsHash == "" {
		s.curChainConfTrustRootsHash, err = s.getCurChainConfTrustRootsHash()
		if err != nil {
			return err
		}

		s.tryReloadChainConfTrustRootsChange()

		s.log.Debugf("[START] current chain config trust roots hash: %s", s.curChainConfTrustRootsHash)
	}

	if err = s.RegisterHandler(); err != nil {
		return fmt.Errorf("register handler failed, %s", err.Error())
	}

	endPoint := fmt.Sprintf(":%d", localconf.ChainMakerConfig.RpcConfig.Port)
	conn, err := net.Listen("tcp", endPoint)
	if err != nil {
		return fmt.Errorf("TCP listen failed, %s", err.Error())
	}

	go func() {
		err = s.grpcServer.Serve(conn)
		if err != nil {
			s.log.Errorf("grpc Serve failed, %s", err.Error())
		}
	}()

	s.log.Infof("gRPC server listen on %s", endPoint)

	return nil
}

// RegisterHandler - register apiservice handler to rpcserver
func (s *RPCServer) RegisterHandler() error {
	apiService := NewApiService(s.chainMakerServer, s.ctx)
	apiPb.RegisterRpcNodeServer(s.grpcServer, apiService)
	return nil
}

// Stop - stop RPCServer
func (s *RPCServer) Stop() {
	s.isShutdown = true
	s.cancel()
	s.grpcServer.GracefulStop()
	s.log.Info("RPCServer is stopped!")
}

// Restart - Restart RPCServer
func (s *RPCServer) Restart(reason string) error {
	var (
		err error
	)

	s.log.Info("RPCServer is beginning to restart")

	s.cancel()
	s.grpcServer.GracefulStop()

	s.grpcServer, err = newGrpc(s.chainMakerServer)
	if err != nil {
		errMsg := fmt.Sprintf("RPCServer restart for reason [%s], new rpc server failed, %s", reason, err.Error())
		s.log.Errorf(errMsg)
		return errors.New(errMsg)
	}

	if err := s.Start(); err != nil {
		errMsg := fmt.Sprintf("RPCServer restart for reason [%s] failed, %s", reason, err.Error())
		s.log.Errorf(errMsg)
		return errors.New(errMsg)
	}

	s.log.Infof("RPCServer is restarted, reason: %s", reason)
	return nil
}

func (s *RPCServer) getCurChainConfTrustRootsHash() (string, error) {
	chainConfs, err := s.chainMakerServer.GetAllChainConf()
	if err != nil {
		return "", fmt.Errorf("get all chain conf failed, %s", err)
	}

	var caCerts []string
	for _, chainConf := range chainConfs {
		for _, trustRoot := range chainConf.ChainConfig().TrustRoots {
			caCerts = append(caCerts, trustRoot.Root)
		}
	}

	sort.Strings(caCerts)

	caCertsStr := strings.Join(caCerts, ";")

	certsHash, err := hash.Get(crypto.HASH_TYPE_SM3, []byte(caCertsStr))
	if err != nil {
		return "", fmt.Errorf("get trust root certs hash failed, %s", err)
	}

	return hex.EncodeToString(certsHash), nil
}

func (s *RPCServer) tryReloadChainConfTrustRootsChange() {
	go func() {
		s.log.Debugf("check chainconf trust roots change goroutine start...")
		for {
			if s.isShutdown {
				break
			}

			s.sleep()
			s.log.Debug("begin to check chain config trust roots cert...")

			if err := s.checkAndRestart(); err != nil {
				s.log.Errorf("check and restart node failed, %s", err.Error())
				continue
			}
		}
	}()
}

func (s *RPCServer) sleep() {
	checkChainConfTrustRootsChangeInterval := localconf.ChainMakerConfig.RpcConfig.CheckChainConfTrustRootsChangeInterval
	if checkChainConfTrustRootsChangeInterval < 10 {
		checkChainConfTrustRootsChangeInterval = 10
	}
	time.Sleep(time.Duration(checkChainConfTrustRootsChangeInterval) * time.Second)
}

func (s *RPCServer) checkAndRestart() error {

	rootsHash, err := s.getCurChainConfTrustRootsHash()
	if err != nil {
		return err
	}

	if s.curChainConfTrustRootsHash != rootsHash {
		s.log.Debugf("different chain config trust roots cert hash: [old:%s]/[new:%s]", s.curChainConfTrustRootsHash, rootsHash)

		if err := s.Restart("TrustRoots certs change, reload it"); err != nil {
			return err
		}

		s.curChainConfTrustRootsHash = rootsHash
	} else {
		s.log.Debugf("same chain config trust roots cert hash: %s", rootsHash)
	}

	return nil
}

// newGrpc - new GRPC object
func newGrpc(chainMakerServer *blockchain.ChainMakerServer) (*grpc.Server, error) {
	var opts []grpc.ServerOption
	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		opts = []grpc.ServerOption{
			grpc_middleware.WithUnaryServerChain(
				RecoveryInterceptor,
				LoggingInterceptor,
				MonitorInterceptor,
				RateLimitInterceptor(),
			),
		}
	} else {
		opts = []grpc.ServerOption{
			grpc_middleware.WithUnaryServerChain(
				RecoveryInterceptor,
				LoggingInterceptor,
				RateLimitInterceptor(),
			),
		}
	}

	if localconf.ChainMakerConfig.RpcConfig.TLSConfig.Mode != TLS_MODE_DISABLE {

		chainConfs, err := chainMakerServer.GetAllChainConf()
		if err != nil {
			return nil, fmt.Errorf("get all chain conf failed, %s", err)
		}

		var caCerts []string
		for _, chainConf := range chainConfs {
			for _, trustRoot := range chainConf.ChainConfig().TrustRoots {
				caCerts = append(caCerts, trustRoot.Root)
			}
		}

		tlsRPCServer := ca.CAServer{
			CaCerts:  caCerts,
			CertFile: localconf.ChainMakerConfig.RpcConfig.TLSConfig.CertFile,
			KeyFile:  localconf.ChainMakerConfig.RpcConfig.TLSConfig.PrivKeyFile,
		}

		checkClientAuth := false
		if localconf.ChainMakerConfig.RpcConfig.TLSConfig.Mode == TLS_MODE_TWOWAY {
			checkClientAuth = true
			log.Infof("need check client auth")
		}

		c, err := tlsRPCServer.GetCredentialsByCA(checkClientAuth)
		if err != nil {
			log.Errorf("new gRPC failed, GetTLSCredentialsByCA err: %v", err)
			return nil, err
		}

		opts = append(opts, grpc.Creds(*c))
	}

	server := grpc.NewServer(opts...)

	return server, nil
}

func splitMethodName(fullMethodName string) (string, string) {
	fullMethodName = strings.TrimPrefix(fullMethodName, "/") // remove leading slash
	if i := strings.Index(fullMethodName, "/"); i >= 0 {
		return fullMethodName[:i], fullMethodName[i+1:]
	}
	return "unknown", "unknown"
}

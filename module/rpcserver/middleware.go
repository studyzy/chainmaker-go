/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpcserver

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"chainmaker.org/chainmaker/localconf/v2"
	"chainmaker.org/chainmaker/logger/v2"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

var log = logger.GetLogger(logger.MODULE_RPC)

const (
	//UNKNOWN unknown string
	UNKNOWN = "unknown"
)

const (
	rateLimitTypeGlobal = 0
)

func GetClientAddr(ctx context.Context) string {
	pr, ok := peer.FromContext(ctx)
	if !ok {
		log.Errorf("getClientAddr FromContext failed")
		return UNKNOWN
	}

	if pr.Addr == net.Addr(nil) {
		log.Errorf("getClientAddr failed, peer.Addr is nil")
		return UNKNOWN
	}

	return pr.Addr.String()
}

func getClientIp(ctx context.Context) string {
	addr := GetClientAddr(ctx)
	return strings.Split(addr, ":")[0]
}

// LoggingInterceptor - set logging interceptor
func LoggingInterceptor(ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

	addr := GetClientAddr(ctx)

	log.Debugf("[%s] call gRPC method: %s", addr, info.FullMethod)
	log.DebugDynamic(func() string {
		str := fmt.Sprintf("req detail: %+v", req)
		if len(str) > 1024 {
			str = str[:1024] + " ......"
		}
		return str
	})
	resp, err := handler(ctx, req)
	log.Debugf("[%s] call gRPC method: %s, resp detail: %+v", addr, info.FullMethod, resp)
	return resp, err
}

// RecoveryInterceptor - set recovery interceptor
func RecoveryInterceptor(ctx context.Context, req interface{},
	_ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {

	defer func() {
		if e := recover(); e != nil {
			stack := debug.Stack()
			os.Stderr.Write(stack)
			log.Errorf("panic stack: %s", string(stack))
			err = status.Errorf(codes.Internal, "Panic err: %v", e)
		}
	}()

	return handler(ctx, req)
}

// MonitorInterceptor - set monitor interceptor
func MonitorInterceptor(ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

	service, method := splitMethodName(info.FullMethod)
	mRecv.WithLabelValues(service, method).Inc()

	start := time.Now()
	resp, err := handler(ctx, req)
	elapsed := time.Since(start)

	mRecvTime.WithLabelValues(service, method).Observe(elapsed.Seconds())

	return resp, err
}

func getRateLimitBucket(bucketMap *sync.Map, tokenBucketSize, tokenPerSecond int, peerIpAddr string) *rate.Limiter {
	rateLimitType := localconf.ChainMakerConfig.RpcConfig.RateLimitConfig.Type
	var (
		bucket interface{}
		ok     bool
	)

	if rateLimitType == rateLimitTypeGlobal {
		if bucket, ok = bucketMap.Load(rateLimitTypeGlobal); ok {
			log.Debug("get rateLimit bucket from global")
			return bucket.(*rate.Limiter)
		}
	} else {
		if bucket, ok = bucketMap.Load(peerIpAddr); ok {
			log.Debugf("get rateLimit bucket from peerIpAddr [%s]", peerIpAddr)
			return bucket.(*rate.Limiter)
		}
	}

	if tokenBucketSize >= 0 && tokenPerSecond >= 0 {
		if tokenBucketSize == 0 {
			tokenBucketSize = rateLimitDefaultTokenBucketSize
		}

		if tokenPerSecond == 0 {
			tokenPerSecond = rateLimitDefaultTokenPerSecond
		}

		bucket = rate.NewLimiter(rate.Limit(tokenPerSecond), tokenBucketSize)
	} else {
		return nil
	}

	if rateLimitType == rateLimitTypeGlobal {
		if bucket, ok = bucketMap.LoadOrStore(rateLimitTypeGlobal, bucket); !ok {
			log.Debug("create rateLimit bucket from global")
		}
	} else {
		if bucket, ok = bucketMap.LoadOrStore(peerIpAddr, bucket); !ok {
			log.Debugf("create rateLimit bucket from peerIpAddr [%s]", peerIpAddr)
		}
	}

	return bucket.(*rate.Limiter)
}

// RateLimitInterceptor - set ratelimit interceptor
func RateLimitInterceptor() grpc.UnaryServerInterceptor {

	tokenBucketSize := localconf.ChainMakerConfig.RpcConfig.RateLimitConfig.TokenBucketSize
	tokenPerSecond := localconf.ChainMakerConfig.RpcConfig.RateLimitConfig.TokenPerSecond
	enabled := localconf.ChainMakerConfig.RpcConfig.RateLimitConfig.Enabled

	bucketMap := sync.Map{}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (
		interface{}, error) {

		if enabled {
			ipAddr := getClientIp(ctx)
			bucket := getRateLimitBucket(&bucketMap, tokenBucketSize, tokenPerSecond, ipAddr)
			if bucket != nil && !bucket.Allow() {
				errMsg := fmt.Sprintf("%s is rejected by ratelimit, try later pls", info.FullMethod)
				log.Warn(errMsg)
				return nil, status.Error(codes.ResourceExhausted, errMsg)
			}
		}

		return handler(ctx, req)
	}
}

// BlackListInterceptor - set ip blacklist interceptor
func BlackListInterceptor() grpc.UnaryServerInterceptor {

	blackIps := localconf.ChainMakerConfig.RpcConfig.BlackList.Addresses

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (
		interface{}, error) {

		ipAddr := getClientIp(ctx)
		for _, blackIp := range blackIps {
			if ipAddr == blackIp {
				errMsg := fmt.Sprintf("%s is rejected by black list [%s]", info.FullMethod, ipAddr)
				log.Warn(errMsg)
				return nil, status.Error(codes.ResourceExhausted, errMsg)
			}
		}

		return handler(ctx, req)
	}
}

// BlackListStreamInterceptor - set ip blacklist interceptor
func BlackListStreamInterceptor() grpc.StreamServerInterceptor {

	blackIps := localconf.ChainMakerConfig.RpcConfig.BlackList.Addresses

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {

		ipAddr := getClientIp(ss.Context())
		for _, blackIp := range blackIps {
			if ipAddr == blackIp {
				errMsg := fmt.Sprintf("%s is rejected by black list [%s]", info.FullMethod, ipAddr)
				log.Warn(errMsg)
				return status.Error(codes.ResourceExhausted, errMsg)
			}
		}

		return handler(srv, ss)
	}
}

func splitMethodName(fullMethodName string) (string, string) {
	fullMethodName = strings.TrimPrefix(fullMethodName, "/") // remove leading slash
	if i := strings.Index(fullMethodName, "/"); i >= 0 {
		return fullMethodName[:i], fullMethodName[i+1:]
	}
	return UNKNOWN, UNKNOWN
}

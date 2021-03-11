/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpcserver

import (
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/logger"
	"context"
	"fmt"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"os"
	"runtime/debug"
	"time"
)

var log = logger.GetLogger(logger.MODULE_RPC)

// LoggingInterceptor - set logging interceptor
func LoggingInterceptor(ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

	log.Debugf("call gRPC method: %s", info.FullMethod)
	log.Debugf("req detail: %+v", req)
	resp, err := handler(ctx, req)
	log.Debugf("call gRPC method: %s, resp detail: %+v", info.FullMethod, resp)
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

// RateLimitInterceptor - set ratelimit interceptor
func RateLimitInterceptor() grpc.UnaryServerInterceptor {

	tokenBucketSize := localconf.ChainMakerConfig.RpcConfig.RateLimitConfig.TokenBucketSize
	tokenPerSecond := localconf.ChainMakerConfig.RpcConfig.RateLimitConfig.TokenPerSecond

	var bucket *rate.Limiter
	if tokenBucketSize >= 0 && tokenPerSecond >= 0 {
		if tokenBucketSize == 0 {
			tokenBucketSize = rateLimitDefaultTokenBucketSize
		}

		if tokenPerSecond == 0 {
			tokenPerSecond = rateLimitDefaultTokenPerSecond
		}

		bucket = rate.NewLimiter(rate.Limit(tokenPerSecond), tokenBucketSize)
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if tokenBucketSize >= 0 && tokenPerSecond >= 0 && !bucket.Allow() {
			errMsg := fmt.Sprintf("%s is rejected by ratelimit, try later pls", info.FullMethod)
			log.Warn(errMsg)
			return nil, status.Error(codes.ResourceExhausted, errMsg)
		}

		return handler(ctx, req)
	}
}

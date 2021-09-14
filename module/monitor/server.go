/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package monitor

import (
	"fmt"
	"net"
	"net/http"

	"chainmaker.org/chainmaker/localconf/v2"
	"chainmaker.org/chainmaker/logger/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MonitorServer struct {
	httpServer *http.Server
	log        *logger.CMLogger
}

func NewMonitorServer() *MonitorServer {
	var log = logger.GetLogger(logger.MODULE_MONITOR)

	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		return &MonitorServer{
			httpServer: &http.Server{
				Handler: mux,
			},
			log: log,
		}
	} else {
		return &MonitorServer{
			log: log,
		}
	}
}

func (s *MonitorServer) Start() error {
	if s.httpServer != nil {
		endPoint := fmt.Sprintf(":%d", localconf.ChainMakerConfig.MonitorConfig.Port)
		conn, err := net.Listen("tcp", endPoint)
		if err != nil {
			return fmt.Errorf("TCP listen failed, %s", err.Error())
		}

		go func() {
			err = s.httpServer.Serve(conn)
			if err != nil {
				s.log.Errorf("http Serve failed, %s", err.Error())
			}
		}()

		s.log.Infof("Monitor http server listen on %s", endPoint)
	}

	return nil
}

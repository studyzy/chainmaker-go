/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package rawsqlprovider

type SqlDbConfig struct {
	//mysql, sqlite, postgres, sqlserver
	SqlDbType       string `mapstructure:"sqldb_type"`
	Dsn             string `mapstructure:"dsn"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	ConnMaxLifeTime int    `mapstructure:"conn_max_lifetime"` //second
	SqlLogMode      string `mapstructure:"sqllog_mode"`       //Silent,Error,Warn,Info
	SqlVerifier     string `mapstructure:"sql_verifier"`      //simple,safe
	DbPrefix        string `mapstructure:"db_prefix"`
}

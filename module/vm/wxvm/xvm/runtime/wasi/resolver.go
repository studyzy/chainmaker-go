/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package wasi

import "chainmaker.org/chainmaker-go/wxvm/xvm/exec"

var resolver = exec.MapResolver(map[string]interface{}{
	"env.___wasi_fd_prestat_get": func(ctx exec.Context, x, y uint32) uint32 {
		return 8
	},
	"env.___wasi_fd_fdstat_get": func(ctx exec.Context, x, y uint32) uint32 {
		return 8
	},
	"env.___wasi_fd_prestat_dir_name": func(ctx exec.Context, x, y, z uint32) uint32 {
		return 8
	},
	"env.___wasi_fd_close": func(ctx exec.Context, x uint32) uint32 {
		return 8
	},
	"env.___wasi_fd_seek": func(ctx exec.Context, x, y, z, w uint32) uint32 {
		return 8
	},
	"env.___wasi_fd_write": func(ctx exec.Context, x, y, z, w uint32) uint32 {
		return 8
	},
	"env.___wasi_environ_sizes_get": func(ctx exec.Context, x, y uint32) uint32 {
		return 0
	},
	"env.___wasi_environ_get": func(ctx exec.Context, x, y uint32) uint32 {
		return 0
	},
	"env.___wasi_args_sizes_get": func(ctx exec.Context, x, y uint32) uint32 {
		return 0
	},
	"env.___wasi_args_get": func(ctx exec.Context, x, y uint32) uint32 {
		return 0
	},
	"env.___wasi_proc_exit": func(ctx exec.Context, x uint32) uint32 {
		exec.Throw(exec.NewTrap("exit"))
		return 0
	},
})

// NewResolver return exec.Resolver which resolves symbols needed by wasi environment
func NewResolver() exec.Resolver {
	return resolver
}

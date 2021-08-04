/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

#include "wxvm.h"

extern wasm_rt_func_handle_t wxvm_resolve_func(void* env, char* module, char* name);
extern int64_t wxvm_resolve_global(void* env, char* module, char* name);
extern uint32_t wxvm_call_func(void* env, wasm_rt_func_handle_t handle, wxvm_context_t* ctx, uint32_t* params, uint32_t param_len);

wxvm_resolver_t make_resolver_t(void* env) {
	wxvm_resolver_t r;
	r.env = env;
	r.resolve_func = wxvm_resolve_func;
	r.resolve_global = wxvm_resolve_global;
	r.call_func = wxvm_call_func;
	return r;
}

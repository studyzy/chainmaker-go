/*
 * Copyright 2018 WebAssembly Community Group participants
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#ifndef wxvm_H_
#define wxvm_H_

#include "wasm-rt.h"

#ifdef __cplusplus
extern "C" {
#endif

struct wxvm_context_t;
struct wxvm_code_t;
struct wxvm_resolver_t;

// Override this variable to define trap function
extern void (*wasm_rt_trap)(wasm_rt_trap_t code);

#define TRAP_NO_MEMORY "run out of memory"
// wxvm_raise is used to raise internal trap error
extern void wxvm_raise(char* msg);

typedef struct wxvm_resolver_t {
  void* env;
  void* (*resolve_func)(void* env, char* module, char* name);
  int64_t (*resolve_global)(void* env, char* module, char* name);
  uint32_t (*call_func)(void* env, wasm_rt_func_handle_t hfunc, struct wxvm_context_t* ctx,
                        uint32_t* params, uint32_t param_len);
} wxvm_resolver_t;

struct FuncType;
typedef struct wxvm_code_t {
  void* dlhandle;
  struct FuncType* func_types;
  uint32_t func_type_count;
  wxvm_resolver_t resolver;
  void* (*new_handle_func)(void*);
  void (*init_func_types)(void*);
  void (*init_import_funcs)(void*);
} wxvm_code_t;

wxvm_code_t* wxvm_new_code(char* module_path, wxvm_resolver_t resolver);
int wxvm_init_code(wxvm_code_t* code);
void wxvm_release_code(wxvm_code_t* code);

typedef struct wxvm_context_t {
  wxvm_code_t* code;
  void* module_handle;
  wasm_rt_memory_t* mem;
  wasm_rt_table_t* table;
} wxvm_context_t;

int wxvm_init_context(wxvm_context_t* ctx, wxvm_code_t* code, uint64_t gas_limit);
void wxvm_release_context(wxvm_context_t* ctx);
uint32_t wxvm_call(wxvm_context_t* ctx, char* name, int64_t* params, int64_t param_len, int64_t* ret);
uint32_t wxvm_mem_static_top(wxvm_context_t* ctx);
void wxvm_reset_gas_used(wxvm_context_t* ctx);
void wxvm_set_gas_used(wxvm_context_t* ctx, uint64_t used);
uint64_t wxvm_gas_used(wxvm_context_t* ctx);

#ifdef __cplusplus
}
#endif

#endif // wxvm_H_

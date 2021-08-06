/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

#include "wxvm.h"
extern void go_wxvm_trap();

void init_go_trap() {
  wasm_rt_trap = go_wxvm_trap;
}
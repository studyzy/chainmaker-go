#include "wxvm.h"
extern void go_wxvm_trap();

void init_go_trap() {
  wasm_rt_trap = go_wxvm_trap;
}
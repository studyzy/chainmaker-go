#Bug Fix
对原有开源项目在vm_func.go NativeFunctionBlock增加：
HasElse                bool
以解决在判定条件分支时，可能跳转到错误的指令的问题
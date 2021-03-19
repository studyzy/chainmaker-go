go build
// test performance of gasm
./profile perf --total-call-times=500009 --vm-type=gasm --wasm-file-path=/Users/tianlehan/project/chainmaker-go/module/vm/test/profile/counter-go.wasm
// test performance of wasmer
./profile perf --total-call-times=500009 --vm-goroutines-num=100 --vm-type=wasmer --wasm-file-path=/Users/tianlehan/project/chainmaker-go/module/vm/test/profile/counter-rust.wasm

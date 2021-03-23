## Build profile
go build

## test performance of gasm
./profile perf --total-call-times=500000 --vm-goroutines-num=10000 --vm-type=gasm --wasm-file-path=/Users/tianlehan/project/chainmaker-go/module/vm/test/profile/counter-go.wasm --report-file-path=/usr/share/nginx/html/vm-performance-report.html

## test performance of wasmer
./profile perf --total-call-times=500000 --vm-goroutines-num=50000 --vm-type=wasmer --wasm-file-path=/Users/tianlehan/project/chainmaker-go/module/vm/test/profile/counter-rust.wasm --report-file-path=/usr/share/nginx/html/vm-performance-report.html

## performance
This profile program will generate a report.md file, please see the performance information in the file.  
This profile program has import the pprof, you can use pprof to optimize it.  
These commands like below:  
CPU Status: go tool pprof http://localhost:6060/debug/pprof/profile  
Memory Status: go tool pprof http://localhost:6060/debug/pprof/heap  
After you got the command line, you can use top or web command to see which used the cpu or memory.
# Build A Benchmark

(1) go build test/bench/*.go

# run bench

## usage

- num: how many client instance to be used
- span: sleep interval for each client between completing last tx and sending next
- method: 'grpc'
- url : like 'localhost:17898'
- cert_root_path : cert root dir, like '../config'
- orgid : specify an organization as a party to the transaction, like 'wx-org1'
- orglist : organization list, which is used for multi-party signature when creating contract, like 'wx-org1,wx-org2,wx-org3,wx-org4'
- update_height : block height is updated once every [update_height] transactions are sent
- wasm_path : wasm path, like '../wasm/counter-go.wasm', this tool require use counter-go.wasm contract, canot use other contracts

## Test by GRPC
```
./bench  -url=localhost:17898 -num=10 -method=grpc -cert_root_path=../config -orgid=wx-org1 -orglist=wx-org1,wx-org2,wx-org3,wx-org4 -update_height=100
```

## Output
```
Height:986,         //block height
Send:533983,        //total send tx count  
Succ:532971,        //total successd tx count
Fail:13, Timeout:0, Duplicated:12,   //fail tx statistics
TPS:1072.285848      //real time TPS,(last 1 minute)
```

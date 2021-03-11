#!/bin/bash
for (( i=0; i<=10000000; i++ ))
do
  ./cmc client contract user invoke --contract-name=asset_new22_1 --method=transfer --sdk-conf-path=../sdk/testdata/sdk_config.yml --org-id=wx-org1.chainmaker.org --client-crt-file-paths=../sdk/testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt --client-key-file-paths=../sdk/testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key --params="{\"amount\":\"1\",\"to\":\"c5d7d472124c988175beacef2b482206910c94845777eb3689af33e240c67129\"}" --sync-result=true
  sleep 2
done

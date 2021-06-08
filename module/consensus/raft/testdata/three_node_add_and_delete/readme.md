# start node1-node3
```sh
docker-compose -f docker-compose.yaml up -d node1 node2 node3
```

# add org4 TrustRoot and add org4
```sh
cd ../../../../../test/send_proposal_request_tool
./send_proposal_request_tool createContract --contract-name contract1 -w ../wasm/go-fact-1.0.0.wasm -p 7988 --org-id wx-org1.chainmaker.org --org-ids wx-org1.chainmaker.org,wx-org2.chainmaker.org,wx-org3.chainmaker.org --admin-sign-crts ../../config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.crt,../../config/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.crt,../../config/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.crt --admin-sign-keys ../../config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.key,../../config/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.key,../../config/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.key

./send_proposal_request_tool invoke  -p 7988 --org-id wx-org1.chainmaker.org --org-ids wx-org1.chainmaker.org,wx-org2.chainmaker.org,wx-org3.chainmaker.org -m save -a "[{\"key\":\"key\",\"value\":\"counter1\"}]"

./send_proposal_request_tool trustRootAdd -p 7988 --org-id wx-org1.chainmaker.org --org-ids wx-org1.chainmaker.org,wx-org2.chainmaker.org,wx-org3.chainmaker.org --trust_root_org_id wx-org4.chainmaker.org --trust_root_crt "-----BEGIN CERTIFICATE-----
MIICrzCCAlWgAwIBAgIDCQ5iMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnNC5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmc0LmNoYWlubWFrZXIub3JnMB4XDTIwMTIwODA2NTM0M1oXDTMw
MTIwNjA2NTM0M1owgYoxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmc0LmNoYWlubWFrZXIub3Jn
MRIwEAYDVQQLEwlyb290LWNlcnQxIjAgBgNVBAMTGWNhLnd4LW9yZzQuY2hhaW5t
YWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAQrXFaZb6AtbXSJMZMY
kxfbnpQHUNdUY1pSOKXteLKnkX6YQMBOkxPtu38Cg2rdnEnQ9JARZJl6QaSiYG9a
I6yco4GnMIGkMA4GA1UdDwEB/wQEAwIBpjAPBgNVHSUECDAGBgRVHSUAMA8GA1Ud
EwEB/wQFMAMBAf8wKQYDVR0OBCIEILnKrWgqcf7PtgtMJFynmIqdFt0oVKmKzxD+
NfPshcHEMEUGA1UdEQQ+MDyCDmNoYWlubWFrZXIub3Jngglsb2NhbGhvc3SCGWNh
Lnd4LW9yZzQuY2hhaW5tYWtlci5vcmeHBH8AAAEwCgYIKoZIzj0EAwIDSAAwRQIh
AMFrB9S3VRwkPpW1UEF2n/8M/6hkBAAgL6x06/5njPpBAiBybPh2fHOiMpaoYK1Y
vq3HKmR3TKv8Gl1ioQ24Xez6Dw==
-----END CERTIFICATE-----
" --admin-sign-keys ../../config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.key,../../config/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.key,../../config/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.key --admin-sign-crts ../../config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.crt,../../config/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.crt,../../config/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.crt  --seq=1

./send_proposal_request_tool chainConfigNodeOrgAdd -p 7988 --org-id wx-org1.chainmaker.org --org-ids wx-org1.chainmaker.org,wx-org2.chainmaker.org,wx-org3.chainmaker.org --nodeOrg_org_id wx-org4.chainmaker.org  --nodeOrg_addresses "QmRRWXJpAVdhFsFtd9ah5F4LDQWFFBDVKpECAF8hssqj6H"  --admin-sign-keys ../../config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.key,../../config/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.key,../../config/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.key --admin-sign-crts ../../config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.crt,../../config/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.crt,../../config/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.crt  --seq=2

./send_proposal_request_tool trustRootAdd -p 7988 --org-id wx-org1.chainmaker.org --org-ids wx-org1.chainmaker.org,wx-org2.chainmaker.org,wx-org3.chainmaker.org --trust_root_org_id wx-org5.chainmaker.org --trust_root_crt "-----BEGIN CERTIFICATE-----
MIICrzCCAlWgAwIBAgIDCoJWMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnNS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmc1LmNoYWlubWFrZXIub3JnMB4XDTIwMTIwODA2NTM0M1oXDTMw
MTIwNjA2NTM0M1owgYoxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmc1LmNoYWlubWFrZXIub3Jn
MRIwEAYDVQQLEwlyb290LWNlcnQxIjAgBgNVBAMTGWNhLnd4LW9yZzUuY2hhaW5t
YWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAQvKJbsKIIfwZDBl7Fd
QFzub5HVLMYHbg9Vocg7FRiuOvggk9nR7kvRm8RD+AY64OpThhE5fCmYJLUhKr0Q
YyhFo4GnMIGkMA4GA1UdDwEB/wQEAwIBpjAPBgNVHSUECDAGBgRVHSUAMA8GA1Ud
EwEB/wQFMAMBAf8wKQYDVR0OBCIEIEUAhxhcWZS15xG8t6OkdHY5bgbJhDdawNvk
X+ev1BPWMEUGA1UdEQQ+MDyCDmNoYWlubWFrZXIub3Jngglsb2NhbGhvc3SCGWNh
Lnd4LW9yZzUuY2hhaW5tYWtlci5vcmeHBH8AAAEwCgYIKoZIzj0EAwIDSAAwRQIg
Joe9KHupPPSSQF7M+u0hmT/3TjHH1P9WkBItt0bFy1kCIQCCaRznhe1jnZ8kD8XS
7F36kC80dzJI7t6qhubcmUbt5A==
-----END CERTIFICATE-----" --admin-sign-keys ../../config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.key,../../config/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.key,../../config/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.key --admin-sign-crts ../../config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.crt,../../config/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.crt,../../config/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.crt  --seq=3

./send_proposal_request_tool chainConfigNodeOrgAdd -p 7988 --org-id wx-org1.chainmaker.org --org-ids wx-org1.chainmaker.org,wx-org2.chainmaker.org,wx-org3.chainmaker.org --nodeOrg_org_id wx-org5.chainmaker.org  --nodeOrg_addresses "QmVSCXfPweL1GRSNt8gjcw1YQ2VcCirAtTdLKGkgGKsHqi"  --admin-sign-keys ../../config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.key,../../config/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.key,../../config/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.key --admin-sign-crts ../../config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.crt,../../config/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.crt,../../config/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.crt  --seq=4

docker-compose -f docker-compose.yaml up -d node4 node5

./send_proposal_request_tool chainConfigNodeAddrDelete -p 7988 --org-id wx-org1.chainmaker.org --org-ids wx-org1.chainmaker.org,wx-org2.chainmaker.org,wx-org3.chainmaker.org --node_addr_org_id wx-org2.chainmaker.org --node_old_address "QmeyNRs2DwWjcHTpcVHoUSaDAAif4VQZ2wQDQAUNDP33gH"  --admin-sign-keys ../../config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.key,../../config/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.key,../../config/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.key --admin-sign-crts ../../config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.crt,../../config/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.crt,../../config/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.crt  --seq=5

./send_proposal_request_tool chainConfigNodeAddrDelete -p 7988 --org-id wx-org1.chainmaker.org --org-ids wx-org1.chainmaker.org,wx-org2.chainmaker.org,wx-org3.chainmaker.org --node_addr_org_id wx-org3.chainmaker.org --node_old_address "QmXf6mnQDBR9aHauRmViKzSuZgpumkn7x6rNxw1oqqRr45"  --admin-sign-keys ../../config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.key,../../config/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.key,../../config/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.key --admin-sign-crts ../../config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.crt,../../config/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.crt,../../config/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.crt  --seq=6

docker-compose -f docker-compose.yaml down node2 node3

for (( i=0; i<10000; i++)) ; do ./send_proposal_request_tool invoke --contract-name contract1 -p 7988 --org-id wx-org1.chainmaker.org --org-ids wx-org1.chainmaker.org,wx-org2.chainmaker.org,wx-org3.chainmaker.org,wx-org4.chainmaker.org -m save -a "[{\"key\":\"key\",\"value\":\"counter1\"}]"; done
```

# stop network
```sh
docker-compose -f docker-compose.yaml down
```

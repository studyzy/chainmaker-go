# getBlockByHeight
./send_proposal_request_tool getBlockByHeight -H 0 -C chain1 -p 7988 --org-id wx-org1.chainmaker.org

# createContract
./send_proposal_request_tool createContract -w ../wasm/go-fact-1.0.0.wasm -p 7988 --org-id wx-org1.chainmaker.org --org-ids wx-org1.chainmaker.org,wx-org2.chainmaker.org,wx-org3.chainmaker.org,wx-org4.chainmaker.org

# invoke 
./send_proposal_request_tool invoke  -p 7988 --org-id wx-org1.chainmaker.org --org-ids wx-org1.chainmaker.org,wx-org2.chainmaker.org,wx-org3.chainmaker.org,wx-org4.chainmaker.org -m save -a "[{\"key\":\"key\",\"value\":\"counter1\"}]"
for (( i=0; i<10000; i++)) ; do ./send_proposal_request_tool invoke  -p 7988 --org-id wx-org1.chainmaker.org --org-ids wx-org1.chainmaker.org,wx-org2.chainmaker.org,wx-org3.chainmaker.org,wx-org4.chainmaker.org -m save -a "[{\"key\":\"key\",\"value\":\"counter1\"}]"; done
for (( i=0; i<10000; i++)) ; do ./send_proposal_request_tool invoke  -p 7988 --org-id wx-org1.chainmaker.org --org-ids wx-org1.chainmaker.org,wx-org2.chainmaker.org,wx-org3.chainmaker.org,wx-org4.chainmaker.org -m save -a "[{\"key\":\"key\",\"value\":\"counter1\"}]"; sleep $(($RANDOM%100)); done

# trustRootAdd
./send_proposal_request_tool trustRootAdd -p 7988 --org-id wx-org1.chainmaker.org --org-ids wx-org1.chainmaker.org,wx-org2.chainmaker.org,wx-org3.chainmaker.org,wx-org4.chainmaker.org --trust_root_org_id wx-org5.chainmaker.org --trust_root_crt ../../config/wx-org5/certs/ca/wx-org5.chainmaker.org/ca.crt

# chainConfigNodeOrgAdd
./send_proposal_request_tool chainConfigNodeOrgAdd -p 7988 --org-id wx-org1.chainmaker.org --org-ids wx-org1.chainmaker.org,wx-org2.chainmaker.org,wx-org3.chainmaker.org,wx-org4.chainmaker.org --nodeOrg_org_id wx-org5.chainmaker.org  --nodeOrg_addresses "/ip4/127.0.0.1/tcp/11305/p2p/QmVSCXfPweL1GRSNt8gjcw1YQ2VcCirAtTdLKGkgGKsHqi"

# chainConfigNodeAddrAdd
./send_proposal_request_tool chainConfigNodeAddrAdd -p 7988 --org-id wx-org1.chainmaker.org --org-ids wx-org1.chainmaker.org,wx-org2.chainmaker.org,wx-org3.chainmaker.org,wx-org4.chainmaker.org --node_addr_org_id wx-org4.chainmaker.org  --node_addresses "/ip4/127.0.0.1/tcp/11305/p2p/QmVSCXfPweL1GRSNt8gjcw1YQ2VcCirAtTdLKGkgGKsHqi"

# parallel
./send_proposal_request_tool parallel invoke  -H localhost:7988,localhost:7989,localhost:7990,localhost:7991 --org-id wx-org1.chainmaker.org --org-ids wx-org1.chainmaker.org,wx-org2.chainmaker.org,wx-org3.chainmaker.org,wx-org4.chainmaker.org -m save -a "[{\"key\":\"key\",\"value\":\"counter1\"}]"  -u ../../config/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key,../../config/crypto-config/wx-org2.chainmaker.org/user/client1/client1.tls.key,../../config/crypto-config/wx-org3.chainmaker.org/user/client1/client1.tls.key,../../config/crypto-config/wx-org4.chainmaker.org/user/client1/client1.tls.key -K ../../config/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt,../../config/crypto-config/wx-org2.chainmaker.org/user/client1/client1.tls.crt,../../config/crypto-config/wx-org3.chainmaker.org/user/client1/client1.tls.crt,../../config/crypto-config/wx-org4.chainmaker.org/user/client1/client1.tls.crt -P ../../config/crypto-config/wx-org1.chainmaker.org/ca,../../config/crypto-config/wx-org2.chainmaker.org/ca,../../config/crypto-config/wx-org3.chainmaker.org/ca,../../config/crypto-config/wx-org4.chainmaker.org/ca --admin-sign-crts ../../config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.crt,../../config/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.crt,../../config/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.crt,../../config/crypto-config/wx-org4.chainmaker.org/user/admin1/admin1.sign.crt --admin-sign-keys ../../config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.key,../../config/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.key,../../config/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.key,../../config/crypto-config/wx-org4.chainmaker.org/user/admin1/admin1.sign.key -I wx-org1.chainmaker.org,wx-org2.chainmaker.org,wx-org3.chainmaker.org,wx-org4.chainmaker.org --loopNum 100000 --threadNum 8 --climbTime=5  --timeout=172800

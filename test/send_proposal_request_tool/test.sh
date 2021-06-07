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


./send_proposal_request_tool chainConfigNodeOrgAdd -p 7988 --org-id wx-org1.chainmaker.org --org-ids wx-org1.chainmaker.org,wx-org2.chainmaker.org,wx-org3.chainmaker.org --nodeOrg_org_id wx-org4.chainmaker.org  --nodeOrg_addresses "/ip4/192.168.2.5/tcp/6666/p2p/QmRRWXJpAVdhFsFtd9ah5F4LDQWFFBDVKpECAF8hssqj6H"  --admin-sign-keys ../../config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.key,../../config/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.key,../../config/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.key --admin-sign-crts ../../config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.crt,../../config/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.crt,../../config/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.crt


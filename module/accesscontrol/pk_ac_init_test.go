/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/protocol/v2"

	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"chainmaker.org/chainmaker/common/v2/helper"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"github.com/stretchr/testify/require"
)

const (
	TestPK1 = `-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCwu3sB7+5LJMtqsP2Y3sn5eW8b
r/dv1d9WXa6p0UEsUoVY4bgDZ471X9e0htnZUWcuvI5B0wHkoaJiKhUxSk5AJ8OY
5IvFI0OqQS07IMqIj3/u3iERVluuawA5IUjPFmCiubJ/Pb/JZCpFDQbZb209h7Vs
OkD1v94WlNJN8sm7qQIDAQAB
-----END PUBLIC KEY-----`
	TestSK1 = `-----BEGIN RSA PRIVATE KEY-----
MIICWwIBAAKBgQCwu3sB7+5LJMtqsP2Y3sn5eW8br/dv1d9WXa6p0UEsUoVY4bgD
Z471X9e0htnZUWcuvI5B0wHkoaJiKhUxSk5AJ8OY5IvFI0OqQS07IMqIj3/u3iER
VluuawA5IUjPFmCiubJ/Pb/JZCpFDQbZb209h7VsOkD1v94WlNJN8sm7qQIDAQAB
AoGAXKZcjR5wSTqH3W3d9KdPMRcFNXmheSKhC9DfAS2vQgIc4AStCDPhESfmmEBd
snznX+v/k+h/xJEr5NR0+bsfm8hbNFBGDysEZt9tzD5YItafs/ePl+PDsAZVv8d/
q3CDwisgZ3oJzPIw+CqRSm2WeL3umQy1RdQXdJ2RsdnRD+UCQQDrB3U0Wefk+Vid
unYpXd+tAXN2cB7GxLsKYAGNz+1AyFLDnbG/K35tihdy0kaa1L/4K9h39xfDnCeK
/c3AfhSHAkEAwIBn4qLuuSQaRwqTGii0IV7Y8Asj/yyUTN/dOo3SB2804nFyNPHX
qfnhUhjRZhxDFLPH4KwKMMOMkilyz3LqTwJAS0yvY19uqXCt0JL96pD16dLuMEMJ
yTschd1uggXdCIVl5uBuI0aHEgdNLe9qyY5iFtvNVdonlfdAwApC0mpSnwJABmT9
jnC9H1dMrCl0w3ywpx8gc7DbDEHt1zPkhGprnKWcCx2bnpieAl5zlqeOZSbxL4Hd
VOBCImaMh9pqnuuBTwJATglQw9U1JbQ10Xq85MdJAVwbd1DgXmM1NAF4V+aeVpFf
V2+2cdpXmxG5Hrp9dy/oR8AGLPQ/JFeUYDxjEU5GNw==
-----END RSA PRIVATE KEY-----`
	TestNodeId1 = "QmQXjPB4DS8fNxsbqWzozSfwRiBbDDZg3t5qTxeb7R8BV5"

	TestPK2 = `-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDcGCdkB7vruKPcGXIneql989b3
pGYfE7icZBHdNrCY6QqDqhFytlLL/zVTpDJVopOQRAonEyZtbhFFPoMntIwRR7Uw
oePfJmPw/woQ7oLJtVWW1a9am15ZIMro/dCzywi1Z6qlU+Vf1uOR7YzPF1OeIymv
0SXG2DMny9m1OJ40ZwIDAQAB
-----END PUBLIC KEY-----`
	TestSK2 = `-----BEGIN RSA PRIVATE KEY-----
MIICXAIBAAKBgQDcGCdkB7vruKPcGXIneql989b3pGYfE7icZBHdNrCY6QqDqhFy
tlLL/zVTpDJVopOQRAonEyZtbhFFPoMntIwRR7UwoePfJmPw/woQ7oLJtVWW1a9a
m15ZIMro/dCzywi1Z6qlU+Vf1uOR7YzPF1OeIymv0SXG2DMny9m1OJ40ZwIDAQAB
AoGBAI2q3m/8qnEoABEEL/5Jbh+sfIoaP8FxKDtCDl2dfj5ugl4Ncf2sbc7xDpov
7lZAt0r9AKv2H54AYw13F2TPSfgD0Vw4YdAgHehG5HgzUO/pRkIy80ARywm7KCM9
RvMBXlbkVsBP/Pwc7KXxDVvlg9GFyDzzb3MgoqsZ56/FsvaBAkEA8p6hvyUTRXE8
YhLnmISXt+uT3D7mZuxgVJ0vBn0sIumCMXtYSHO8tWmlU0XysNy9H0LiTsGc+cSm
FFH1Hb8xWQJBAOg7gP11lMN9mjjHlrf1VviR/r5F9H1lU+MIsW/oxLrOHwwny9B5
FsNui9unDjG93R8ggt3tqeSDgX619YZkG78CQCDjiCGVMQuU0g6paWOvdbGk6aJN
lIYXPOe7dwh2J2mEJfX3Nnx70/TzoUmsjb2T7r8yHeN3M4RYN/tBMO0bYeECQFLm
ovZX2gIrPTmVriz/LMvROjHsQQneeSKrwMOlQU06NYUeU7iY8VJUjSKdMQj6sQvi
jDTzGVnUxA5aoEoYRHsCQG9bw6jejRyQ0CM3Xv1gurAy089LyDF04mn7GuHWFj6B
0NBQdGsuTBTxd0uXa/kMEwd3F782IEL2h1tDCBjhYAM=
-----END RSA PRIVATE KEY-----`
	TestNodeId2 = "QmRUuqP9WkNmHv2NR8P9RUyBKSdjHuz4uu79hjgM4rWri4"

	TestPK3 = `-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQC4YX5dJBVlHNh6LKAd2dTeW9yx
NWbAPGzgiCF8tOmRuJ+jfBt2VY8GkBxMwPwsEf8mp/4l1NXRp/jSJ69SRwU06rb4
ujl7NzkeBwzc7cFbFxoqoKeI0nmIJcn7tI2YCSX7HtoddnsyLZNCFFKDRB/Db4f1
5Z5rDZWMAMlxeffoGQIDAQAB
-----END PUBLIC KEY-----`
	TestSK3 = `-----BEGIN RSA PRIVATE KEY-----
MIICXQIBAAKBgQC4YX5dJBVlHNh6LKAd2dTeW9yxNWbAPGzgiCF8tOmRuJ+jfBt2
VY8GkBxMwPwsEf8mp/4l1NXRp/jSJ69SRwU06rb4ujl7NzkeBwzc7cFbFxoqoKeI
0nmIJcn7tI2YCSX7HtoddnsyLZNCFFKDRB/Db4f15Z5rDZWMAMlxeffoGQIDAQAB
AoGAL8eb7lkGblBeTLK5v2KOhhy6APX8rX47HKhKPT3IdSmpvLzRhQXA7Yt0ufMc
pfL38rV/55/S1OS5VwRPq3uZ/l44084zvBMLVrZRpGd4Map4QLNfsXhgNWE+qLne
Bj4HoNSsyzMmzUccp6KKs3FaeTwtLA3cU4qO3xV7Piq0vrECQQDyd/rr42p5Fj3R
Imxc29dSAX+Qph9DV2hDMc4U49/OVsku8yufGQagDWjvAt9jLTu41QtwM7c20asK
iJ8pEn8tAkEAwqukx3RQhhH3cdRFKqQq1sd+IgNMhpdboZhf5vs4+Z3gGKroQVKj
NfHL1SDCtDlxppgFQnI7vzaoJKFzVg2AHQJAQ/xwVwQFLr6VxrYoPEFINq5E3oI1
8ePoUC7+4cyjTG/5KTj12j5iJS6dZacgi+Z7AHB8LJHTpYNUujdkqVeOYQJBAITf
3dhad0Ab8Vcb+Z4Scj8p6dlTgR95Ho1dUVB697fB4B1WQrObsVV31paCBwQ3FXEN
4MEq8cchioF+RhhdnK0CQQCxsKE9Q6Cc+u1LzzZ3mwIJnjSmnpXAsGbP40kCEqHH
Aw0Q6CM4gQkL4N0LgqqmyWglIkfCscVOTA50OGYEioxj
-----END RSA PRIVATE KEY-----`
	TestNodeId3 = "Qmd8o58EHnsfBbDikRra4XNsCmArXjXLSdZdkYwDcdsUvQ"

	TestPK4 = `-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDjToONx0tUBD3lvXscgXkR81r8
c1doBh02g0tjseWeOehROM0hIVYHeY/6CoBtmdXSIlc0ITI/dm4wLw47egkd6gQO
ph2EeuFarzTcXJjztBZElDXEQpppj4E5vx7Qp21uc4uAv/x5o6ORy0tL90bo/n/z
bUzV0TbkOzV4pib2swIDAQAB
-----END PUBLIC KEY-----`
	TestSK4 = `-----BEGIN RSA PRIVATE KEY-----
MIICXAIBAAKBgQDjToONx0tUBD3lvXscgXkR81r8c1doBh02g0tjseWeOehROM0h
IVYHeY/6CoBtmdXSIlc0ITI/dm4wLw47egkd6gQOph2EeuFarzTcXJjztBZElDXE
Qpppj4E5vx7Qp21uc4uAv/x5o6ORy0tL90bo/n/zbUzV0TbkOzV4pib2swIDAQAB
AoGBAN2eWkMsUTRsIlFRSawETB+FTmuepVTFyUux/RoJg5+eQ/SU1eL8Vp1ZF1gp
TwgNGd0UIEOyLgSUGmCeMFkq5aDPKxwI356YdvHB7SdEJMy5I7fYEKJatEQ3FOtM
F5Il1NYPgtrwKeAAs9mSdH2Lp7/KH+UAmhH5wSPZ3MRbk5oRAkEA+iaGYvFTn1pP
PgEbun53ZNGYirVK8qqYe8lDTLYPtM9Q/HM5OP1Fojd06e+6mBtFX9A8Jm8rnmJZ
l4CgF0K86QJBAOifPTzbGC51daI0k7hBPZCdm3KBQu91gx9G07wpEPV5a6RCho18
8ugO3qGCvCHKXmrAh81sEVVPLCjPZ17h5TsCQH2uI3DMrOXwOsX9SpAtgBEQWWK/
aVN4sLnoyb5d7pA6ZQchYQun/HdfA4eRoZ9QfE+CUOZCjpi58ydyQXzOVBkCQB1j
wQDnTW7ROEN+EQu+cmDLCNC2tBY86owRDr8/EP1ykb73CLjniGj5N/d/5PT/9F3Y
ZU/2z1nP3uxpB85dC/ECQCHuQLatz4sIVuHYQNl2nRiu+1eNfgpB6wGtIh3O782/
Wk0+loBRjk4pBj9djnlephMX9gHptNqyYvJtM3LMJm0=
-----END RSA PRIVATE KEY-----`
	TestNodeId4 = "QmQ8nYaAMm5DdMzf3GaY2NPkmGneqmRyaSJDNRaFwuoxwV"

	TestPK5 = `-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDJ2H7PWt3l8CF+zvQuRssnVAyY
xyB4tZ6KpXPpb9Y7fzK4mju8pBBFyErBSNM5uqgGYrnmf9Plh3MZvwSGd5ZVHZ3b
f9qx4cttzzAMsHcjra6AJ44qo5jxX0bVIiyErqEqGvYvAqUEb7Ye0kFKs/Z2tq2e
o79CwhOI0TLHOpby8wIDAQAB
-----END PUBLIC KEY-----`
	TestSK5 = `-----BEGIN RSA PRIVATE KEY-----
MIICXAIBAAKBgQDJ2H7PWt3l8CF+zvQuRssnVAyYxyB4tZ6KpXPpb9Y7fzK4mju8
pBBFyErBSNM5uqgGYrnmf9Plh3MZvwSGd5ZVHZ3bf9qx4cttzzAMsHcjra6AJ44q
o5jxX0bVIiyErqEqGvYvAqUEb7Ye0kFKs/Z2tq2eo79CwhOI0TLHOpby8wIDAQAB
AoGAbLtbVIg2kO9Sm+UQVP194qm8P3DFZUExLq8CSfYdCd/zis5K78vRmEXVP1nj
r22Fpir4ydqCY1sb/fqQjX9OU4ZhwmvWAnHehjPTuzGgkwnKcUyDnWROX6B2wFOc
d9AaUxgZm4Dek7jXImfiKkb0YZxuMXiBZZmdqEhbxlM4QcECQQDwJH+egh6fKXMU
AbUmKuYDlH/D6m52FTozkQU9wWa5xW+/u+XJOVuX8gaaa9S15sX2gDqpB7v0HkO2
BnAqX2qlAkEA1yya9Hx/A/l1Z/rk0JRBiLSxYvdBmjLg43+9mhKCoSI+1adRyri4
26HJcnMAZa15hNWFHN0/QX3BNG+zIhgrtwJAGwJP5DkITqhvzAFBKZDLm/14vUVB
tUA/8orOBxsYfa5qGit89bvgxF8xRO751pelDktvzZEUH6nDvdZNiUaADQJAZsxq
o08vJ2jwjGKzGmsZ/APHk25pKxAPnOCUZp1dRzojJtOvIdiqiFN8+G60y97a5XlV
BPs2k0VPHowW2r0NdQJBAIVCNla0HUB1+1qtsI1qJHqzbI5eXTjEmKJ3FHxYvC9U
E0pyQvClu6PjjyAWRNMSGms4slKFgGffMJf4o1nkB1k=
-----END RSA PRIVATE KEY-----`

	TestPK6 = `-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDEyJWuNjX1BNfsbPtmgzcbfFzb
tHTbOH6I9svBaOAL8B1VyPK+RGIHtZgU5D4EOyWmFTwuXIQ98XGAkQHCJm75mKcG
Th/a3Q0HAf9guR8q5MT+mp0yckHuieMKQD30RafxoZi9H3LBKyGR2KMkNLjdhL3o
Vj+jFNeHf0gOkiQifwIDAQAB
-----END PUBLIC KEY-----`
	TestSK6 = `-----BEGIN RSA PRIVATE KEY-----
MIICXQIBAAKBgQDEyJWuNjX1BNfsbPtmgzcbfFzbtHTbOH6I9svBaOAL8B1VyPK+
RGIHtZgU5D4EOyWmFTwuXIQ98XGAkQHCJm75mKcGTh/a3Q0HAf9guR8q5MT+mp0y
ckHuieMKQD30RafxoZi9H3LBKyGR2KMkNLjdhL3oVj+jFNeHf0gOkiQifwIDAQAB
AoGARc4hyrrQSSp+rg+63pKNaeKjzgwlp95ShKOHhAR/9bwnq9asxXHclH+Gg2Kz
3SxeHpxJzOhkwNR1PvYxeX3Iv4HKtdrruY9w15i6xNCBClIj/mK4A5IygSnbz6dC
S6gs6KBVpiLmY2IP4Uh28NcKW4eFe233xoKh6S5WHLUJDIECQQD2P9bmSfOJ5GTn
mrjBF0In8MKqq+Nu5miJ14gO3XrTMK8YkoE1V6qufvYBVD5YVHZySLam7pMXAxIO
zSy1fPm1AkEAzJNS7vQokTz+Ud25SYqFVT/g82VYxGRLuOYNWNJCXIY6G86p7/VQ
iRvJ6QM2DF41MMA3ox/bX4VllG4TP2474wJBANCGZOukSehGETCTM8rHcE00MxSl
9DUwRewcKOo1oVH/kvai8WmDcFTNzHJ5rUXNWHQUoR+hPcup3PvNwQN67lUCQAod
HnSBzZ+gjFIvzAE+v+i/B7gAwqqy6qtxdCd3/Z/lYuoNBYm/bwPYQ9spNXrXDXoj
hpyh7o6CYcs8xebU5FECQQCH0ZcTXrcX452gG+WUUL+0M0uDy4tF7KOkO89Aisqm
9Pr4zSK60hTtx3dAuD83HhezMjBxRqLHjLk0QcuNgHeq
-----END RSA PRIVATE KEY-----`

	TestPK7 = `-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQC++M/QNw2iVFY6lOKZsFLu2poh
8OzNpc//BO6nfp+9ByuaaM2fL8oWt5bGGAxHRbBdt2UA6MgD5fi8ATFygHdy7RHL
4WwgaCv7VM4E4EI2LPrPMUj8ufQG+Pp1t4OQU4FdYCq6eQCc49bPkTcEKnDt24wJ
yz8mDNPU5pEP/c509QIDAQAB
-----END PUBLIC KEY-----`
	TestSK7 = `-----BEGIN RSA PRIVATE KEY-----
MIICXAIBAAKBgQC++M/QNw2iVFY6lOKZsFLu2poh8OzNpc//BO6nfp+9ByuaaM2f
L8oWt5bGGAxHRbBdt2UA6MgD5fi8ATFygHdy7RHL4WwgaCv7VM4E4EI2LPrPMUj8
ufQG+Pp1t4OQU4FdYCq6eQCc49bPkTcEKnDt24wJyz8mDNPU5pEP/c509QIDAQAB
AoGAI0MD9DlGHjQeW+DD2obxOUNJ9Hxs7SfxuO/rNSgvTJL3XSJ+3SbQ1NL/VwJ3
ue1HPHaxgrJ4xCeBfw1lWPQZmeAYsnlp3DGg1Nglr0w/aMd5cXeSqX4rYsXMD6vU
b1pY8eQ3SCO1pnge0xIoce2l0NxbZNMP8/d35FUE5ULMawECQQDq5Glsgyck7+Gu
MLM+OZEf7HgChSRwILDcxWJtrxSqUacjZz0CzZ9B4nTFDKqzcxXl7oc0QRFh6z/w
G5KZxts1AkEA0CIIhfRB+/DiGPgAi9OxR/1q5ihJ6rwxl6F3DxnIq7KSxxglXw9M
SvUyZC0j4GR/2BO+PPBvWfNE/2celEqqwQJAOAX4exAg8vdf3VryNWInkfSlfvxg
f3nclRti6YQ7qo/FDHWgIJ4IYP9xGFp4EErfqzKj/ruSOMeSPWNmKNU8DQJAaw2m
Tqg1LE5ZLTiap1EqdXneeyWr52YHKBPv9j9v3QiLwIYl6sAmoMN/uNETC/8FVvHI
vvV4gM7E5Y13yBSjwQJBAIrxmftYFKaIQI4AiHlQ+vDVxjqpern9lY/DidTZpZ6H
h1iX7Cg1KNPMmf5lX6vBAvR8jy2W4mQS81yWQucFmmw=
-----END RSA PRIVATE KEY-----`

	TestPK8 = `-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDQOJSqXyNB+Q62cXT+lx4TGNDU
Ast/pGwRFzPo+Ofef7lafQqu60gbkq4spQYqEgWyd7xr5tEw3tnQr3VEnSaQu2nS
gJDcT4yol0brUV0b2im9PNA45Q8QT+cZVILPLf3jJZtIxBFLps9q2Js65Xc8P314
UGClc2AZd8w7G7MLlwIDAQAB
-----END PUBLIC KEY-----`
	TestSK8 = `-----BEGIN RSA PRIVATE KEY-----
MIICXQIBAAKBgQDQOJSqXyNB+Q62cXT+lx4TGNDUAst/pGwRFzPo+Ofef7lafQqu
60gbkq4spQYqEgWyd7xr5tEw3tnQr3VEnSaQu2nSgJDcT4yol0brUV0b2im9PNA4
5Q8QT+cZVILPLf3jJZtIxBFLps9q2Js65Xc8P314UGClc2AZd8w7G7MLlwIDAQAB
AoGBAIzvbzjuUkgKUvoMS3szQAj/CAIoriMEYJ0kzl8HcrI4U3Y7IqsI1/LJ0pin
TkfVkQOeZevG/JsOi/HjgQVjNUERuRNEAuMWLtCBK5SQp7Re6QsjwZ+K0i/AKuup
Ph5RRPZ6osWfApFUmhhO/H/Vr+jlAAEEhLmTbjiMPStSQBD5AkEA/KFgiex7D7Lq
Ozu/tL+YqbMusNaWF4KX6wGfI+KD9RanTeRoPU6n2Hm/Gz/HW97pvKvLdsXwNJ7D
glI9S+jzGwJBANL/kKFrYkyiVNLYzrr8rAE0/bpjPU9UpiI/d4u8Vo9sAUkgYIBv
IVIQLn6yH2ZM62qsYdz4aaeJUN266EKOlTUCQCMGRJoanR0aEvtPV0652XJ9kxWV
So3L30AHo4aYGu9Zyqwz5HfLdd2/U0111C/agdFUiArZemnxMO3adQEXNM0CQBxu
gY+ux6Up7qImwtyhdZAIEvSNsNJCxswwnyw+Ka/TzuyKp1ZHI0dKlOlPmTmQvdw2
9EzxUFNaBoKKUAe/7M0CQQDqPspBjU5ihqx/HFGnFHcVvtyhpUh8SHHltKgR2igJ
5rAc8ICzKkgr5Cqd/yQDaT5R1mtYdgd+AvZ/ucEvDDgK
-----END RSA PRIVATE KEY-----`

	TestPK = `-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDQOJSqXyNB+Q62cXT+lx4TGNDU
Ast/pGwRFzPo+Ofef7lafQqu60gbkq4spQYqEgWyd7xr5tEw3tnQr3VEnSaQu2nS
gJDcT4yol0brUV0b2im9PNA45Q8QT+cZVILPLf3jJZtIxBFLps9q2Js65Xc8P314
UGClc2AZd8w7G7MLlwIDAQAB
-----END PUBLIC KEY-----`

	TestPK9 = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA57b1SHQKXl47qa/35WaX
3UNI/UTBIPxcJjJIW84hYBOMDWGwBQHE1n+l+b9FctQXRIpB/DBPasRvg8YfUSN0
5Rexv3/s+Z/nIXXlVaatwBV+tCb97iqKurYJpMuxJmOhe68ENgFGZkUfAMvByHfP
VfzKg2y8gZmsxyfPnZ0dPZpm/xSzdceNI49iWLFvCARkKEjuO0rjL8tqk3cxY5uT
DSov68UEbyhQOYTodLlByY9uwzQOF74TUWV7ZiEfDoFwaiJi7Q+60wQm9oPS/Xeo
E97IfA9tEKe7kKmBQZxoI8vyStPUQ1WEFn+wtcB67pS9dEpTRWBgmQYtLtkq4lcH
RQIDAQAB
-----END PUBLIC KEY-----`
	TestSK9 = `-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEA57b1SHQKXl47qa/35WaX3UNI/UTBIPxcJjJIW84hYBOMDWGw
BQHE1n+l+b9FctQXRIpB/DBPasRvg8YfUSN05Rexv3/s+Z/nIXXlVaatwBV+tCb9
7iqKurYJpMuxJmOhe68ENgFGZkUfAMvByHfPVfzKg2y8gZmsxyfPnZ0dPZpm/xSz
dceNI49iWLFvCARkKEjuO0rjL8tqk3cxY5uTDSov68UEbyhQOYTodLlByY9uwzQO
F74TUWV7ZiEfDoFwaiJi7Q+60wQm9oPS/XeoE97IfA9tEKe7kKmBQZxoI8vyStPU
Q1WEFn+wtcB67pS9dEpTRWBgmQYtLtkq4lcHRQIDAQABAoIBAQCcJrKzaefW4oAo
gTp4sKOk64QTkbLozMg4wWf73jSlr2aRWgSpyyBgQNOUM67UjFNF0DpZfiD23Xwc
/HX8Uv2iqU4StF35dyXmabHr/5BVwuaI90Hmr2qgGq7zDIXMThXz6OTYlBFiODCF
c8qakwr5cory+GMsn2hNKeoC2G9tJAirf+5Bj5xDx5JTwtggUr5NUR37fyeAa2Wh
atqY5cb2k3MEqr6p1zZZ5GLNADvgzXWK+XeDNA/x8AqwHM6ETpljwBd8Rx5EHwjd
6tj7K1oBW43s5lMBe4K0cOgLAGpK48Ck0DZ++8CovqQSQUe7OsIY1MX5ZSrwpKuh
vyjiDMehAoGBAP7rtxozJkhXP899UZsB9q5QuRMLTZruFNFEgjRRsEc+Qr6O25qf
cEzE3yO2EpAmSMS570eK596SvjO6Jd1GM/e/AUnGlOJ3fEOt58ICNWV9V3RdhFwi
Nmmqj3yXhiSsQU695+1NwFViajaXTDjXP/LskJP2hLpolIrkwOanKryZAoGBAOiy
F4zP/RdbhM1CjkyJXHaG8Uru/jimOuKSHhpHnANeNQrMylM4IYH33/GFIIS7hsJg
9ohH5XMnslp1CjbEvjDXuurJx5TFBMFCPy3LppQDKZBLnWyNGBsSDu6/oq+/ozIA
KVTEGlQvqZDQCaxXwvxHFB1w+xlHXAdsY0TVL7+NAoGAVXwEHeQTLWUcv969c+aX
q2LkfU9oCdFW58o6g4L1Qx7M0Qwk9lgLF6NZVKdk2DQOaPIVHH+nO8snvz7oHajC
Go1RyESwfrUk1alGs5d8AnmizyHhFehfKNYKYfSKBlhBWj9yu/A71CY5ie74n4MH
LdZIsWWUotIZJe6KBY7/VNkCgYEAscHaW6dHH+C5wlNlgPItwB21lhib+4qA0TPt
6wVpGOmOe4GVzZzDfBVu7YFVJhBbEYIg0lqZ3S4mARQHiW8iGw2xrEoYPH2E9F03
BjTcO5Vu2tvollPyZjuVTKz4CmnKsReOe0KTGlyOnCFQQmeIfE+P/i2go97vXnxe
GOcCYsECgYEAhB0srmyo49HErxPBzVKN1bVz1GczVtUUVx0tJLhvdM43vTZeXMnl
EstwZWXf3UB5FRsmHSZoBhRmkNzFQ54pM/hlGaGrirgYowMfe5j0Zd0jZBQqpApW
YET1/PeQXaF1V9k8wmmZHuLdgo8tw90x0GVCcmxs3UHlGsEJ0ggnCOk=
-----END RSA PRIVATE KEY-----`

	TestPK10 = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAw4tNkDyjmr+X+X49ydLY
Zg1xfu/pykqQtp+Uj9yOajRLuDvPJ5a7Xw/sZ5CVTC3YYeVMNXsAdi3OC9epONXp
1DWjx75Ey+DfQGjpB1ugLFMnWaundj5FSe8kQ7Q6bFM0hrmhhVT4XIabg1NRNr/X
RObHGpIgxIBkEJp88hGMhhrrJXla07p5UOmGPywSJTcxosIhuQ5AenE2anHrQJ2j
QfbJOMcdL8cujw0VNjLxlGLPh0y0F6g0ee++/aS2Mn1Jw9vZkA4YOEzTt3Qg7Scr
1WdrmYnH6eqZb/tHJmCn5E8CmN+KHhVzzuI7NB8P4tBq/N7ho/jq3RRgT1psw51g
swIDAQAB
-----END PUBLIC KEY-----`
	TestSK10 = `-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEAw4tNkDyjmr+X+X49ydLYZg1xfu/pykqQtp+Uj9yOajRLuDvP
J5a7Xw/sZ5CVTC3YYeVMNXsAdi3OC9epONXp1DWjx75Ey+DfQGjpB1ugLFMnWaun
dj5FSe8kQ7Q6bFM0hrmhhVT4XIabg1NRNr/XRObHGpIgxIBkEJp88hGMhhrrJXla
07p5UOmGPywSJTcxosIhuQ5AenE2anHrQJ2jQfbJOMcdL8cujw0VNjLxlGLPh0y0
F6g0ee++/aS2Mn1Jw9vZkA4YOEzTt3Qg7Scr1WdrmYnH6eqZb/tHJmCn5E8CmN+K
HhVzzuI7NB8P4tBq/N7ho/jq3RRgT1psw51gswIDAQABAoIBACyTqQ7kg/dXDfIW
UUedBS/eiK0DTCyNawf2wQs6oEydt1U8bTD9L6GwI5hIYYCIQveuKf1XGPfX4UzZ
0P3f5fo2cCusuEox7TLlt5mxzYXNPv82HmraLzl3hrDYeSkQnrzHvIaEpEmTdggu
CimM+in+4gywmz+wdR9D2I/maD55pfbHwFE6G8NypcY4cDJxxgHYBya1FussKToP
5dtx4Ck2VgPJycCw1naAlkeP6v3mC/tBFW+YZoezS6JpTynsxPT2XFTrNstYV8x2
UrgLtKMSU4uwlsSnHxa4Yf3Kme6WlLwamN9Qj7QoCiWT+csDYbjwpJFRK651f5Hl
HIuW7mECgYEA9wOzC42/6G01yP0fkbm0ZpWp2mqu8U24mguS/2GHRtAsvs5r0XCx
wOQiIT/CTu2qr+ISlzjAOmFPgGqaEBTXFNP+VUhsyL9SxN0G1yuvyC1ZaZL55eXz
3i+QMkHurtyHkXj0elCGoKig80pjD4BEceAEAnR0JG8hF0QnTAohTl8CgYEAyqhK
eMrWfxxYFEo8XskamnZWs8JoZbhaxYaJYkCxrm9Y9ijVwH9daSe/NeV5Balf4R0n
u2TfPF+aXH9arXZ6s9hMh9s00LJcOSEOX4LY17eQDbYpTb2KPFnbIjh3Ut312wr0
xx3uhk2Ihh7RMP9qpk9UO1bmXDKS4dTnWTCvpi0CgYAr0rkyJIzWhIGVTesK5IJv
7L98o46z+tD0a3dB3aCtXIODuoWAW9j9Wrv/YBtt+1Zb6+TWdVgNQ3RiWQdKMRhT
dqTZpoa+OstJZ9kt1W9TOVBynYO+WMSiN5gCgpYA6dkXYvkktiKcYC5l212lw2Dh
PxgXA2gTiq+5O/soz2dHSwKBgGk4Zao/zoyiv8yRGrUv/yMRrESa/K9Lv71s8+nS
oy5pW6w7WXgf6PUPEQU/xs08uq5b/+QZJJrpHHFIImGL8XttI5cqJkrxQFbdJeRL
QKEICsBDw0A82Agrs04aOUIKQntfPeYgUVbj7K2OVJj3FH2TFK3WmbZm/8JHU3MI
hzplAoGAV5TKxE+8UW36vXi+iC6BexiK1GeJmLZ+stJtiGU2vUN9ZlIdEIL60EN7
ahLL6pWe+yFhNzeA7HZErIuq3lOR+zCbyYeQYRUaWXJ6ENpBHyxvkcAkh3+BI1Ea
VHumj5cFWtDxA/G3tM2Vk0h8iIy9q4HDtun42pdtAHNJGCbKA8c=
-----END RSA PRIVATE KEY-----`

	TestPK11 = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAt0BqW4FOBWO7G+G1Qoo+
EAH9vsg4F8Q6X+AC4MR0fY/npL/5ERy0YZHYsp5AEaRqoqfySfwSXEpm+AMip6Un
eKYJM1ZoBK5UJMzvRO55UDGHSEm4KV25UZKHHQrxLzUf8nA7ZbXTdc+h5RDKX0B6
AOtXtpAqjUiYGf/ITdeJO+CS9OuKWUOiPS9oIOD+OMDkTFIrz2pC3R1Zezlmze2w
rfNQSIjXJJzAvzJb1CnAgmMl4zERXDuQLqz9g8qZt1OQ2UTnzbRQA57UAuojtooL
g6ld0MPIYhR3D7u1EmPF1npQUz+zG9QRQAfioC9I1O+IbLpDSyek1MECVbpR1PZg
2wIDAQAB
-----END PUBLIC KEY-----`
	TestSK11 = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAt0BqW4FOBWO7G+G1Qoo+EAH9vsg4F8Q6X+AC4MR0fY/npL/5
ERy0YZHYsp5AEaRqoqfySfwSXEpm+AMip6UneKYJM1ZoBK5UJMzvRO55UDGHSEm4
KV25UZKHHQrxLzUf8nA7ZbXTdc+h5RDKX0B6AOtXtpAqjUiYGf/ITdeJO+CS9OuK
WUOiPS9oIOD+OMDkTFIrz2pC3R1Zezlmze2wrfNQSIjXJJzAvzJb1CnAgmMl4zER
XDuQLqz9g8qZt1OQ2UTnzbRQA57UAuojtooLg6ld0MPIYhR3D7u1EmPF1npQUz+z
G9QRQAfioC9I1O+IbLpDSyek1MECVbpR1PZg2wIDAQABAoIBAQCljgSIduFN7TP1
lIx1eP9o5uOfoLNMhXNXesIe3l1/sqrMJMOXuh8cpu7nMCEhzzCnkqNKQ/kyd+Ve
2zZLzuFCFn7pan6++9/4/0yLMgdXc+eMX02J0arDD2YRzvjmdVBPbyW6VfKc1OCm
Wez68P1IJ1YvET/gNF1136fO65KIDGnY4rdqnyz6rbsSwzWYY+P60xSTX9KATbBo
I6mu66F7ANTMmlo6XpxITOCQhdFDL4Py8RggDusujdAIB8RS7Kz0M4mpdZIGBIR0
5KwV0l8m2guLyLCTanyMvBes4/c2tfdIbyiNgjhaPb1wOaPLnEIVzreU5mCNyiH5
qAveTdKBAoGBAPKRmeZZ5jkTNTpg+mdpwxbSMqX2vrqhSztejTAHgrJYxgk7nvTC
9SNw9NLivc3DZYIgxNHLTt1dt2yPq0mU+HbjwvxVgD9qIlHtcx5xUSGX/JE5jcTf
lbQmLdG0oY4+KHXc1WRxLhOmCd+0H5AyDmC4kvSpSDjBkM4C1V71NCCjAoGBAMFl
/0fcifbe8KRApt93Zz+YQUkCZqw9wcnBrwIDp1nl7PnJo89n4JcD1Rquf0a53jxc
DKnAjx5m3h7Q8fs9KKKeuCGmfO5FVdxJy2pvZOQAHMp9dTDOBrwIRG9HOZG5ics/
+jtPhsy9xDaKAMLRTIg8OhDxXoQ8PCOQ3uf8QOppAoGBAOqowGF/hqCgXFXli1iP
kBN7tVOoqEqTztvYVG2qVl2CU9KKwvO1xsBKfg2lHEj6RjDk0oLCU8EC8HctZV8B
pnwdSnwhmre+TQVE2KESrpH5HnS/YM6cHY7xgFHmlIOuziV3RVitxQ1tCxBGiGJO
imo3JLNbMGr3lsY1J4V9YLhRAoGAa1MADNANXAuyPWSHdoGbsYX7zNlhQvpunVk3
loWSjGf1T1Uf68x4rTV6QHlPtl8VPifS+y0Z/0QUxcMsVkFFWKF+C2aJ8+xUTpBB
K0qwEXsifxiKPVBIGnb4C0zaXM0686kIY3upkdtJlP6Wl4Zw0zWg/6AC1J1cvlv5
54FsQOkCgYB017jnQQlm4LIG7hcDj2lSwMnhe3g4bJMu5SQnQdsqQuUHfvFX3xkz
MHvuEY0DCymy3Y9tlBBNoeT0+IDR9NYkaerYpXyn0nsoJeB4OWjbvNGP6YlptaxK
7wfn/wzBsNmwvWRqVG5pIn6j7PsYJy9yfryVQaSRxJBiCTDe5QSqFg==
-----END RSA PRIVATE KEY-----`

	TestPK12 = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAwIBsfUYDenouuKQpYmNM
QflcAaugnXlaJsMxLshDP6QmMS1jFPcAIZBGPnRiznnpj0yWi5ppl+WvV8MH18n4
quPTGOs2caUM0jh80Y2FY6+/zXGGZ7Df4z0u825l1VImXB83sZjBXVProchc8WeR
ES3jsCejuFrOrbbSzgqlSK0pkKDjVSPRzvQHPIvrpGFKDx3fxJgSGq8/gYSySyAL
h3QaKkAlYl6MM48pfNMYgvmmUMAR82FE/4TmYlkNzlPTRS7pAqtpbkxOlbhsDIQ3
COCocGGpHdTN1Jc4ZoKr6TkZezXDmX7PcuSThF8SqUI30acrdHYfENuT7q5df0bX
8wIDAQAB
-----END PUBLIC KEY-----`
	TestSK12 = `-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEAwIBsfUYDenouuKQpYmNMQflcAaugnXlaJsMxLshDP6QmMS1j
FPcAIZBGPnRiznnpj0yWi5ppl+WvV8MH18n4quPTGOs2caUM0jh80Y2FY6+/zXGG
Z7Df4z0u825l1VImXB83sZjBXVProchc8WeRES3jsCejuFrOrbbSzgqlSK0pkKDj
VSPRzvQHPIvrpGFKDx3fxJgSGq8/gYSySyALh3QaKkAlYl6MM48pfNMYgvmmUMAR
82FE/4TmYlkNzlPTRS7pAqtpbkxOlbhsDIQ3COCocGGpHdTN1Jc4ZoKr6TkZezXD
mX7PcuSThF8SqUI30acrdHYfENuT7q5df0bX8wIDAQABAoIBABLLt0wYDPjlezBe
HwhTR7vdXCVxm5Ikqi/EuUWnGiHZpj0BAH6oi2O7kZBBjvA4KRzrzns9DXheXduE
2HwyZUxSSGdTeBJqmjDggRd46QBNxb2KiyQOuh2W+MGeEuVcSxCNn8OAdcjmC8jV
JnYPtbNmtqeZhzvV6f4+LqEdmvvYeRsnLliHHdU5mpS1/m8FJpE3W/d1GZQ2KVN3
nmqrSYSaqTfPI5dV7Ij9OeO37basFAKC5JfPcQ/7EY0aKi9rlgbOIUfa8+LDG2ve
X2f6t7n2OkOWpRBv+JzvrJU31k5tlTYqQJc2OfxJT1EI+Lj6fXMj9Upiosq8TR+X
8XwBVuECgYEA6+RB3Q4rYeEoWsphABYmmlYRyJp6GXMyxA9FZttFxV9SPc+HX8px
tWHeq6yCKh0+UjuVeydJH6d/XGAfwYrhxGAgY+Be1dT3fn4mWmc2iq4x/cTvW2VA
5ebuFBnDxqq8oCNCMjlZwkABOFrhPww/MzehqVQ5Kj9dXvVuYlnQXEMCgYEA0OlK
CpHNvHJ6lrXBVn3iIQu+OyMxfi7B/dds3c7rWWd0exSTlwwVK60DY6bgfxNyfWKz
zmr1L30P62CHqJ1PCOsNKAo+/BtP2EEa9ZZWC9yqGNB0t8Y37fkXTFmw31ADbfOt
Xd5zp+wlLqlMyeTQg1D0Kxk9becDO70oagu0spECgYBxzRfdTlWtjdNLIbF0Ojt7
X6SKs8PN/V5zaa6gtY5ObvMdML5tfxwmVkX3am0NZjhHsckmtcg4RjVSWmlXlOng
NEPMC1WVMX4I/1D/ciXE987UT6rt28ZYY3VeKyPg90Oyue/YjQR5iylLh8R9ByqC
SgdqymAdup4QDrWnKw8zQwKBgA78PPhnHwfeelanMPggTYErU3jwfFNdzUKFGmUK
u60NE7jkb/XMwxP/9BdI2B+laHgABX/QAkhmwyaSJQj+R7YPDkGKApyY5PBRMzrc
js2JBZaEFWs9R7PFQ1uRr3NFTQmtCgmKtGceNEiVklGFHUPeIbWZuONSR9QYLHb2
4f5RAoGAEKyhTOiYFIDT7nigSncSPaetehnYM+OmudXn2NlBXKnHTc9+3YmeJuJr
Ob0yRE9CD0mXHTYpJfb9T2y/GQQCL5xeV6JuSHUFET2bH2sas/CS+5sTtOe1ObK7
5RwvOyELHncqgGEzi9A0CsfmE0rjoK1UXquz1gBV0rTvifWy4Ts=
-----END RSA PRIVATE KEY-----`
)

var testConsensus1PKInfo = &testPKInfo{
	TestPK1,
	TestSK1,
}

var testConsensus2PKInfo = &testPKInfo{
	TestPK2,
	TestSK2,
}

var testConsensus3PKInfo = &testPKInfo{
	TestPK3,
	TestSK3,
}

var testConsensus4PKInfo = &testPKInfo{
	TestPK4,
	TestSK4,
}

var testAdmin1PKInfo = &testPKInfo{
	TestPK5,
	TestSK5,
}

var testAdmin2PKInfo = &testPKInfo{
	TestPK6,
	TestSK6,
}

var testAdmin3PKInfo = &testPKInfo{
	TestPK7,
	TestSK7,
}

var testAdmin4PKInfo = &testPKInfo{
	TestPK8,
	TestSK8,
}

// var testClient1PKInfo = &testPKInfo{
// 	TestPK9,
// 	TestSK9,
// }

// var testClient2PKInfo = &testPKInfo{
// 	TestPK10,
// 	TestSK10,
// }

// var testClient3PKInfo = &testPKInfo{
// 	TestPK11,
// 	TestSK11,
// }

// var testClient4PKInfo = &testPKInfo{
// 	TestPK12,
// 	TestSK12,
// }

const (
	testPermissionedKeyAuthType = "permissionedwithkey"
)

var testPermissionedPKChainConfig = &config.ChainConfig{
	ChainId:  testChainId,
	Version:  testVersion,
	AuthType: testPermissionedKeyAuthType,
	Sequence: 0,
	Crypto: &config.CryptoConfig{
		Hash: testPKHashType,
	},
	Block: nil,
	Core:  nil,
	Consensus: &config.ConsensusConfig{
		Type: 0,
		Nodes: []*config.OrgConfig{{
			OrgId:  testOrg1,
			NodeId: []string{TestNodeId1},
		}, {
			OrgId:  testOrg2,
			NodeId: []string{TestNodeId2},
		}, {
			OrgId:  testOrg3,
			NodeId: []string{TestNodeId3},
		}, {
			OrgId:  testOrg4,
			NodeId: []string{TestNodeId4},
		},
		},
		ExtConfig: nil,
	},
	TrustRoots: []*config.TrustRootConfig{
		{
			OrgId: testOrg1,
			Root:  []string{TestPK5},
		},
		{
			OrgId: testOrg2,
			Root:  []string{TestPK6},
		},
		{
			OrgId: testOrg3,
			Root:  []string{TestPK7},
		},
		{
			OrgId: testOrg4,
			Root:  []string{TestPK8},
		},
	},
}

func TestGetNodeIdFromPK(t *testing.T) {
	testPk := `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA1MGJxND46zq8QFWoqtOA
uVgoD3E1FJez8/hq0ks5vjzW8jbaHUQQNtc//ZnFkCLOBUkXaYKm/gDDnLxrDG3b
k96hZCn+JHyJWTU3N2eim/2Ta2LR0CQG0pgPMagP0MnHmBKoNYPGpGm6Itldg3vm
jexQ5rDhPeTb3dyowOAWM3K4fX5xVJArRV7d1IbHmTrImBaJ+5JIh5IOFBo6z8vN
od00dAsPufI2ieKJpHIRZWSdUrM3VmScF+B5kZo5FU/dV/i15psrLgfedcgQBH70
gFh3kIKkVF43OghVeK5nokm4c/2HCRn/zsGKXdLeFhpT3Gpntao8kJ8LJBbEuz0T
8QIDAQAB
-----END PUBLIC KEY-----`
	var nodeId string
	pk, err := asym.PublicKeyFromPEM([]byte(testPk))
	require.Nil(t, err)
	nodeId, err = helper.CreateLibp2pPeerIdWithPublicKey(pk)
	require.Nil(t, err)
	fmt.Println("nodeId:", nodeId)
}

type testPKInfo struct {
	pk string
	sk string
}

type testPkMemberInfo struct {
	consensus *testPKInfo
	admin     *testPKInfo
}

type testPkOrgMemberInfo struct {
	testPkMemberInfo
	orgId string
}

type testPkOrgMember struct {
	orgId      string
	acProvider protocol.AccessControlProvider
	consensus  protocol.SigningMember
	admin      protocol.SigningMember
}

var testPKOrgMemberInfoMap = map[string]*testPkOrgMemberInfo{
	testOrg1: {
		testPkMemberInfo: testPkMemberInfo{
			consensus: testConsensus1PKInfo,
			admin:     testAdmin1PKInfo,
		},
		orgId: testOrg1,
	},
	testOrg2: {
		testPkMemberInfo: testPkMemberInfo{
			consensus: testConsensus2PKInfo,
			admin:     testAdmin2PKInfo,
		},
		orgId: testOrg2,
	},
	testOrg3: {
		testPkMemberInfo: testPkMemberInfo{
			consensus: testConsensus3PKInfo,
			admin:     testAdmin3PKInfo,
		},
		orgId: testOrg3,
	},
	testOrg4: {
		testPkMemberInfo: testPkMemberInfo{
			consensus: testConsensus4PKInfo,
			admin:     testAdmin4PKInfo,
		},
		orgId: testOrg4,
	},
}

func initPKOrgMember(t *testing.T, info *testPkOrgMemberInfo) *testPkOrgMember {
	td, cleanFunc, err := createTempDirWithCleanFunc()
	require.Nil(t, err)
	defer cleanFunc()
	logger := logger.GetLogger(logger.MODULE_ACCESS)

	ppkProvider, err := newPermissionedPkACProvider(testPermissionedPKChainConfig,
		info.orgId, nil, logger)
	require.Nil(t, err)
	require.NotNil(t, ppkProvider)

	localPrivKeyFile := filepath.Join(td, info.orgId+".key")

	err = ioutil.WriteFile(localPrivKeyFile, []byte(info.consensus.sk), os.ModePerm)
	require.Nil(t, err)

	consensus, err := InitPKSigningMember(ppkProvider, info.orgId, localPrivKeyFile, "")
	require.Nil(t, err)

	err = ioutil.WriteFile(localPrivKeyFile, []byte(info.admin.sk), os.ModePerm)
	require.Nil(t, err)

	admin, err := InitPKSigningMember(ppkProvider, info.orgId, localPrivKeyFile, "")
	require.Nil(t, err)

	return &testPkOrgMember{
		orgId:      info.orgId,
		acProvider: ppkProvider,
		consensus:  consensus,
		admin:      admin,
	}
}

func testInitPermissionedPKFunc(t *testing.T) map[string]*testPkOrgMember {
	_, cleanFunc, err := createTempDirWithCleanFunc()
	require.Nil(t, err)
	defer cleanFunc()

	var testPkOrgMember = make(map[string]*testPkOrgMember, len(testPKOrgMemberInfoMap))
	for orgId, info := range testPKOrgMemberInfoMap {
		testPkOrgMember[orgId] = initPKOrgMember(t, info)
	}
	test1PermissionedPKACProvider = testPkOrgMember[testOrg1].acProvider
	test2PermissionedPKACProvider = testPkOrgMember[testOrg2].acProvider
	return testPkOrgMember
}

const (
	testPublicAuthType                   = "public"
	testPublicTrustRootOrgId             = "public"
	DPoSErc20TotalKey                    = "erc20.total"
	DPoSErc20OwnerKey                    = "erc20.owner"
	DPosErc20DecimalsKey                 = "erc20.decimals"
	DPosErc20AccountKey                  = "erc20.account:DPOS_STAKE"
	DPosStakeMinSelfDelegation           = "stake.minSelfDelegation"
	DposStakeEpochValidatorNum           = "stake.epochValidatorNum"
	DPosStakeEpochBlockNum               = "stake.epochBlockNum"
	DposStakeCompletionUnbondingEpochNum = "stake.completionUnbondingEpochNum"
	DposStakeCandidate                   = "stake.candidate:"
	DposStakeNodeId                      = "stake.nodeID:"
)

var testPublicPKChainConfig = &config.ChainConfig{
	ChainId:  testChainId,
	Version:  testVersion,
	AuthType: testPublicAuthType,
	Sequence: 0,
	Crypto: &config.CryptoConfig{
		Hash: testPKHashType,
	},
	Block: nil,
	Core:  nil,
	Consensus: &config.ConsensusConfig{
		Type: 5,
		Nodes: []*config.OrgConfig{
			{
				OrgId: DposOrgId,
				NodeId: []string{
					"QmQXjPB4DS8fNxsbqWzozSfwRiBbDDZg3t5qTxeb7R8BV5",
					"QmRUuqP9WkNmHv2NR8P9RUyBKSdjHuz4uu79hjgM4rWri4",
					"Qmd8o58EHnsfBbDikRra4XNsCmArXjXLSdZdkYwDcdsUvQ",
					"QmQ8nYaAMm5DdMzf3GaY2NPkmGneqmRyaSJDNRaFwuoxwV",
				},
			},
		},
		DposConfig: []*config.ConfigKeyValue{
			{Key: DPoSErc20TotalKey, Value: "10000000"},
			{Key: DPoSErc20OwnerKey, Value: "QmQXjPB4DS8fNxsbqWzozSfwRiBbDDZg3t5qTxeb7R8BV5"},
			{Key: DPosErc20DecimalsKey, Value: "18"},
			{Key: DPosErc20AccountKey, Value: "10000000"},
			{Key: DPosStakeMinSelfDelegation, Value: "2500000"},
			{Key: DposStakeEpochValidatorNum, Value: "4"},
			{Key: DPosStakeEpochBlockNum, Value: "10"},
			{Key: DposStakeCompletionUnbondingEpochNum, Value: "1"},
			{Key: DposStakeCandidate + "QmQXjPB4DS8fNxsbqWzozSfwRiBbDDZg3t5qTxeb7R8BV5",
				Value: "250000"},
			{Key: DposStakeNodeId + "",
				Value: "QmQXjPB4DS8fNxsbqWzozSfwRiBbDDZg3t5qTxeb7R8BV5"},
			{Key: DposStakeNodeId + "",
				Value: "QmRUuqP9WkNmHv2NR8P9RUyBKSdjHuz4uu79hjgM4rWri4"},
			{Key: DposStakeNodeId + "",
				Value: "Qmd8o58EHnsfBbDikRra4XNsCmArXjXLSdZdkYwDcdsUvQ"},
			{Key: DposStakeNodeId + "",
				Value: "QmQ8nYaAMm5DdMzf3GaY2NPkmGneqmRyaSJDNRaFwuoxwV"},
		},
	},
	TrustRoots: []*config.TrustRootConfig{
		{
			OrgId: testPublicTrustRootOrgId,
			Root:  []string{TestPK5, TestPK6, TestPK7, TestPK8},
		},
	},
}

var testPKMemberInfoMap = map[string]*testPkMemberInfo{
	testOrg1: {
		consensus: testConsensus1PKInfo,
		admin:     testAdmin1PKInfo,
	},
	testOrg2: {
		consensus: testConsensus2PKInfo,
		admin:     testAdmin2PKInfo,
	},
	testOrg3: {
		consensus: testConsensus3PKInfo,
		admin:     testAdmin3PKInfo,
	},
	testOrg4: {
		consensus: testConsensus4PKInfo,
		admin:     testAdmin4PKInfo,
	},
}

type testPkMember struct {
	acProvider protocol.AccessControlProvider
	consensus  protocol.SigningMember
	admin      protocol.SigningMember
}

func initPKMember(t *testing.T, info *testPkMemberInfo) *testPkMember {
	td, cleanFunc, err := createTempDirWithCleanFunc()
	require.Nil(t, err)
	defer cleanFunc()
	logger := logger.GetLogger(logger.MODULE_ACCESS)

	pkProvider, err := newPkACProvider(testPublicPKChainConfig, nil, logger)
	require.Nil(t, err)
	require.NotNil(t, pkProvider)

	localPrivKeyFile := filepath.Join(td, "public.key")

	err = ioutil.WriteFile(localPrivKeyFile, []byte(info.consensus.sk), os.ModePerm)
	require.Nil(t, err)

	consensus, err := InitPKSigningMember(pkProvider, "", localPrivKeyFile, "")
	require.Nil(t, err)

	err = ioutil.WriteFile(localPrivKeyFile, []byte(info.admin.sk), os.ModePerm)
	require.Nil(t, err)

	admin, err := InitPKSigningMember(pkProvider, "", localPrivKeyFile, "")
	require.Nil(t, err)

	return &testPkMember{
		acProvider: pkProvider,
		consensus:  consensus,
		admin:      admin,
	}
}

func testInitPublicPKFunc(t *testing.T) map[string]*testPkMember {
	_, cleanFunc, err := createTempDirWithCleanFunc()
	require.Nil(t, err)
	defer cleanFunc()

	var testPkMember = make(map[string]*testPkMember, len(testPKMemberInfoMap))
	for orgId, info := range testPKMemberInfoMap {
		testPkMember[orgId] = initPKMember(t, info)
	}
	test1PublicPKACProvider = testPkMember[testOrg1].acProvider
	test2PublicPKACProvider = testPkMember[testOrg2].acProvider
	return testPkMember
}

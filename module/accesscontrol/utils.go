package accesscontrol

import (
	"crypto/sha256"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"

	"chainmaker.org/chainmaker/common/v2/cert"
	bccrypto "chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"chainmaker.org/chainmaker/common/v2/crypto/pkcs11"
	"chainmaker.org/chainmaker/localconf/v2"
	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/mr-tron/base58"
)

func getP11HandleId() string {
	p11Config := localconf.ChainMakerConfig.NodeConfig.P11Config
	return p11Config.Library + p11Config.Label
}

func getP11Handle() (*pkcs11.P11Handle, error) {
	var err error
	p11Config := localconf.ChainMakerConfig.NodeConfig.P11Config
	p11Key := getP11HandleId()
	p11Handle, ok := p11HandleMap[p11Key]
	if !ok {
		p11Handle, err = pkcs11.New(p11Config.Library, p11Config.Label, p11Config.Password, p11Config.SessionCacheSize,
			p11Config.Hash)
		if err != nil {
			return nil, fmt.Errorf("fail to initialize organization with HSM: [%v]", err)
		}
		p11HandleMap[p11Key] = p11Handle
	}
	return p11Handle, nil
}

func pubkeyHash(pubkey string) string {
	pkHash := sha256.Sum256([]byte(pubkey))
	strPkHash := base58.Encode(pkHash[:])
	return strPkHash
}

func loadSyncMap(syncMap *sync.Map, key string) (interface{}, bool) {
	tempSyncMap := *syncMap
	return tempSyncMap.Load(key)
}

func InitCertSigningMember(chainConfig *config.ChainConfig, localOrgId,
	localPrivKeyFile, localPrivKeyPwd, localCertFile string) (
	protocol.SigningMember, error) {

	var certMember *certificateMember

	if localPrivKeyFile != "" && localCertFile != "" {
		certPEM, err := ioutil.ReadFile(localCertFile)
		if err != nil {
			return nil, fmt.Errorf("fail to initialize identity management service: [%s]", err.Error())
		}

		isTrustMember := false
		for _, v := range chainConfig.TrustMembers {
			certBlock, _ := pem.Decode([]byte(v.MemberInfo))
			if certBlock == nil {
				return nil, fmt.Errorf("new member failed, the trsut member cert is not PEM")
			}
			if v.MemberInfo == string(certPEM) {
				certMember, err = newCertMemberFromParam(v.OrgId, v.Role,
					chainConfig.Crypto.Hash, false, certPEM)
				if err != nil {
					return nil, fmt.Errorf("init signing member failed, init trust member failed: [%s]", err.Error())
				}
				isTrustMember = true
				break
			}
		}

		if !isTrustMember {
			certMember, err = newMemberFromCertPem(localOrgId, chainConfig.Crypto.Hash, certPEM, false)
			if err != nil {
				return nil, fmt.Errorf("fail to initialize identity management service: [%s]", err.Error())
			}
		}

		skPEM, err := ioutil.ReadFile(localPrivKeyFile)
		if err != nil {
			return nil, fmt.Errorf("fail to initialize identity management service: [%s]", err.Error())
		}

		var sk bccrypto.PrivateKey
		p11Config := localconf.ChainMakerConfig.NodeConfig.P11Config
		if p11Config.Enabled {
			var p11Handle *pkcs11.P11Handle
			p11Handle, err = getP11Handle()
			if err != nil {
				return nil, fmt.Errorf("fail to initialize identity management service: [%s]", err.Error())
			}

			sk, err = cert.ParseP11PrivKey(p11Handle, skPEM)
			if err != nil {
				return nil, fmt.Errorf("fail to initialize identity management service: [%s]", err.Error())
			}
		} else {
			sk, err = asym.PrivateKeyFromPEM(skPEM, []byte(localPrivKeyPwd))
			if err != nil {
				return nil, err
			}
		}

		return &signingCertMember{
			*certMember,
			sk,
		}, nil
	}
	return nil, nil
}

func InitPKSigningMember(ac protocol.AccessControlProvider,
	localOrgId, localPrivKeyFile, localPrivKeyPwd string) (protocol.SigningMember, error) {

	if localPrivKeyFile != "" {
		skPEM, err := ioutil.ReadFile(localPrivKeyFile)
		if err != nil {
			return nil, fmt.Errorf("fail to initialize identity management service: [%s]", err.Error())
		}

		var sk bccrypto.PrivateKey
		p11Config := localconf.ChainMakerConfig.NodeConfig.P11Config
		if p11Config.Enabled {
			//TODO 硬件加密

		} else {
			sk, err = asym.PrivateKeyFromPEM(skPEM, []byte(localPrivKeyPwd))
			if err != nil {
				return nil, fmt.Errorf("fail to initialize identity management service: [%s]", err.Error())
			}
		}

		publicKeyPEM, err := sk.PublicKey().String()
		if err != nil {
			return nil, fmt.Errorf("fail to initialize identity management service: [%s]", err.Error())
		}

		pbMember := &pbac.Member{
			OrgId:      localOrgId,
			MemberInfo: []byte(publicKeyPEM),
			MemberType: pbac.MemberType_PUBLIC_KEY,
		}

		member, err := ac.NewMember(pbMember)
		if err != nil {
			return nil, fmt.Errorf("fail to initialize identity management service: [%s]", err.Error())
		}

		return &signingPKMember{
			*(member.(*pkMember)),
			sk,
		}, nil
	}
	return nil, nil
}

func ConvertAuthType(authTypeStr string) (AuthType, error) {

	authTypeStr = strings.ToLower(authTypeStr)

	// 兼容1.x ChainConfig authType
	if authTypeStr == Identity {
		return PermissionedWithCert, nil
	}

	authType, ok := StringToAuthTypeMap[authTypeStr]
	if !ok {
		return 0, fmt.Errorf("convert auth type failed, invalid auth type in chain config")
	}
	return authType, nil
}

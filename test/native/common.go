package native

import (
	"chainmaker.org/chainmaker/common/ca"
	"chainmaker.org/chainmaker/common/crypto"
	"chainmaker.org/chainmaker/common/crypto/asym"
	apiPb "chainmaker.org/chainmaker/pb-go/api"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"fmt"
	"google.golang.org/grpc"
	"io/ioutil"
	"log"
)

const (
	CHAIN1         = "chain1"
	IP             = "localhost"
	Port           = 12301
	certPathPrefix = "../../config"
	userKeyPath    = certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key"
	userCrtPath    = certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt"
	orgId          = "wx-org1.chainmaker.org"
	prePathFmt     = certPathPrefix + "/crypto-config/wx-org%s.chainmaker.org/user/admin1/"
)

var (
	WasmPath        = ""
	WasmUpgradePath = ""
	contractName    = ""
	runtimeType     commonPb.RuntimeType
	caPaths         = []string{certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/ca"}

	conn   *grpc.ClientConn
	client apiPb.RpcNodeClient
	sk3    crypto.PrivateKey
	err    error
	txId   string
)

func init() {
	// init
	{
		conn, err = initGRPCConnect(true)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer conn.Close()

		client = apiPb.NewRpcNodeClient(conn)

		file, err := ioutil.ReadFile(userKeyPath)
		if err != nil {
			panic(err)
		}

		sk3, err = asym.PrivateKeyFromPEM(file, nil)
		if err != nil {
			panic(err)
		}
	}
}

func initGRPCConnect(useTLS bool) (*grpc.ClientConn, error) {
	url := fmt.Sprintf("%s:%d", IP, Port)
	if useTLS {
		tlsClient := ca.CAClient{
			ServerName: "chainmaker.org",
			CaPaths:    caPaths,
			CertFile:   userCrtPath,
			KeyFile:    userKeyPath,
		}

		c, err := tlsClient.GetCredentialsByCA()
		if err != nil {
			log.Fatalf("GetTLSCredentialsByCA err: %v", err)
			return nil, err
		}
		return grpc.Dial(url, grpc.WithTransportCredentials(*c))
	} else {
		return grpc.Dial(url, grpc.WithInsecure())
	}
}

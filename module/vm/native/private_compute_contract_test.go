package native

import (
	"chainmaker.org/chainmaker-go/common/crypto/asym"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

var certFile = "testdata/remote_attestation/in_teecert.pem"
var verificationKeyFile = "testdata/remote_attestation/secretkey.pem"
var encryptKeyFile = "testdata/remote_attestation/secretkeyExt.pem"

func TestExtractPubKeyPair(t *testing.T) {
	file, err := os.Open(certFile)
	if err != nil {
		t.Fatalf("open cert file error: %v", err)
	}

	certPEM, err := ioutil.ReadAll(file)
	if err != nil {
		t.Fatalf("read cert file error: %v", err)
	}

	// 读取公钥
	verificationPubKey, encryptPubKey, err := getPubkeyPairFromCert(certPEM)
	if err != nil {
		t.Fatalf("extract pub key pair error: %v", err)
	}
	verificationPubKeyBytes, _ := verificationPubKey.Bytes()
	fmt.Println("get verification pub key ok !")
	fmt.Printf(" %x \n", verificationPubKeyBytes)
	fmt.Println("get encrypt pub key !")
	encryptPubKeyStr, _ := encryptPubKey.String()
	fmt.Printf("%v \n", encryptPubKeyStr)

	//
	file, err = os.Open(verificationKeyFile)
	if err != nil {
		t.Fatalf("open verification private key file error: %v", err)
	}

	verificationPrivKeyDER, err := ioutil.ReadAll(file)
	if err != nil {
		t.Fatalf("read verification private key file error: %v", err)
	}

	verificationPrivKey, err := asym.PrivateKeyFromDER(verificationPrivKeyDER)
	if err != nil {
		t.Fatalf("decode verification private key from DER error: %v", err)
	}
	fmt.Println("get verification private key ok !")
	verificationPubKey2 := verificationPrivKey.PublicKey()
	fmt.Printf("%x \n", verificationPubKey2)
}

func TestGenRemoteAttestationProof(t *testing.T) {
	//challenge := "This is a challenge message for test."

}

func TestSaveRemoteAttestation(t *testing.T) {
	ds := map[string][]byte{}
	mockCtx := newTxContextMock(ds)

	contactRuntime := PrivateComputeRuntime{}
	params := map[string]string{}
	params["proof"] = string([]byte{ 0x01, 0x02,0x03,0x03 })
	result, err := contactRuntime.SaveRemoteAttestation(mockCtx, params)
	assert.Error(t, err, "Save remote attestation error")

	fmt.Printf("result = %v \n", string(result));
	for key, val := range ds {
		fmt.Printf("key = %v, val = %v \n", key, val)
	}
}

func TestInTeecertPemFile(t *testing.T) {
	file, err := os.Open("/Users/caizhihong/证书测试/in_teecert.pem")
	if err != nil {
		t.Fatalf("open file error: %v", err)
	}

	crtPEM, err := ioutil.ReadAll(file)
	if err != nil {
		t.Fatalf("read file error: %v", err)
	}

	signingPubkey, cryptoPubkey, err := getPubkeyPairFromCert(crtPEM)
	if err != nil {
		t.Fatalf("get pubkey pair error: %v", err)
	}
	fmt.Printf("signing pub key ==> %v \n", signingPubkey.ToStandardKey())
	fmt.Printf("crypto pub key ==> %v \n", cryptoPubkey.ToStandardKey())
}
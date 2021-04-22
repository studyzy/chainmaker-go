/*
 * Copyright (c) 2021. ChainMaker.org
 */

package precompiledContracts

import (
	"chainmaker.org/chainmaker-go/evm/evm-go/params"
)

// bn256PairingIstanbul implements a pairing pre-compile for the bn256 curve
// conforming to Istanbul consensus rules.
type bn256PairingIstanbul struct{}

//
//func (c *bn256PairingIstanbul)SetValue(v string){}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bn256PairingIstanbul) GasCost(input []byte) uint64 {
	return params.Bn256PairingBaseGasIstanbul + uint64(len(input)/192)*params.Bn256PairingPerPointGasIstanbul
}

func (c *bn256PairingIstanbul) Execute(input []byte) ([]byte, error) {
	return runBn256Pairing(input)
}

// runBn256Pairing implements the Bn256Pairing precompile, referenced by both
// Byzantium and Istanbul operations.
func runBn256Pairing(input []byte) ([]byte, error) {
	//// Handle some corner cases cheaply
	//// Convert the input into a set of coordinates
	//var (
	//	cs []*bn256.G1
	//	ts []*bn256.G2
	//)
	//for i := 0; i < len(input); i += 192 {
	//	c, err := newCurvePoint(input[i : i+64])
	//	if err != nil {
	//		return nil, err
	//	}
	//	t, err := newTwistPoint(input[i+64 : i+192])
	//	if err != nil {
	//		return nil, err
	//	}
	//	cs = append(cs, c)
	//	ts = append(ts, t)
	//}
	//// Execute the pairing checks and return the results
	//if bn256.PairingCheck(cs, ts) {
	//	return true32Byte, nil
	//}
	//return false32Byte, nil
	return nil, nil
}

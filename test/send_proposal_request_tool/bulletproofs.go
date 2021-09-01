/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/base64"
	"errors"
	"fmt"

	"chainmaker.org/chainmaker/common/v2/crypto/bulletproofs"
	"github.com/spf13/cobra"
)

var (
	// proveCMD flags
	openingStr string
	valueX     int64

	// proveMethodCMD flags
	commitmentMethod string
	valueY           int64
	commitmentXStr   string
	commitmentYStr   string
	openingXStr      string
	openingYStr      string

	// pedersenMethodCMD
	// openingMethodCMD
	// commitmentMethodCmd
	pedersenNegMethod string
)

func BulletproofsCMD() *cobra.Command {
	bulletproofsCmd := &cobra.Command{
		Use:   "bulletproofs",
		Short: "ChainMaker bulletproofs command",
		Long:  "ChainMaker bulletproofs command",
	}

	// generate opening
	bulletproofsCmd.AddCommand(genOpeningCMD())

	// generate proof, commitment, opening
	bulletproofsCmd.AddCommand(proveCMD())

	// prove method
	bulletproofsCmd.AddCommand(proveMethodCMD())

	// Verify the validity of a commitment with respect to a value-opening pair
	bulletproofsCmd.AddCommand(pedersenVerifyCMD())

	bulletproofsCmd.AddCommand(pedersenNegCMD())

	return bulletproofsCmd
}

func genOpeningCMD() *cobra.Command {
	genOpeningCmd := &cobra.Command{
		Use:   "genOpening",
		Short: "Bulletproofs generate opening command",
		Long:  "Bulletproofs generate opening command",
		RunE: func(_ *cobra.Command, _ []string) error {
			return genOpening()
		},
	}

	return genOpeningCmd
}

func genOpening() error {

	opening, err := bulletproofs.PedersenRNG()
	if err != nil {
		return err
	}

	openingStr := base64.StdEncoding.EncodeToString(opening)
	fmt.Printf("opening: [%s]\n", openingStr)

	return nil
}

func proveCMD() *cobra.Command {
	proveCmd := &cobra.Command{
		Use:   "prove",
		Short: "Bulletproofs prove command",
		Long:  "Bulletproofs prove command",
		RunE: func(_ *cobra.Command, _ []string) error {
			return prove()
		},
	}

	flags := proveCmd.Flags()
	flags.StringVarP(&openingStr, "opening", "", "", "opening")
	flags.Int64VarP(&valueX, "value", "", -1, "value")

	return proveCmd
}

func prove() error {
	if valueX == -1 {
		return errors.New("invalid input, please check it")
	}
	commitmentStr := ""
	proofStr := ""
	if openingStr == "" {
		proof, commitment, opening, err := bulletproofs.ProveRandomOpening(uint64(valueX))
		if err != nil {
			return err
		}
		proofStr = base64.StdEncoding.EncodeToString(proof)
		commitmentStr = base64.StdEncoding.EncodeToString(commitment)
		openingStr = base64.StdEncoding.EncodeToString(opening)
	} else {
		opening, err := base64.StdEncoding.DecodeString(openingStr)
		if err != nil {
			return err
		}
		proof, commitment, err := bulletproofs.ProveSpecificOpening(uint64(valueX), opening)
		if err != nil {
			return err
		}
		proofStr = base64.StdEncoding.EncodeToString(proof)
		commitmentStr = base64.StdEncoding.EncodeToString(commitment)
	}

	fmt.Printf("value: [%d]\n", uint64(valueX))
	fmt.Printf("proof: [%s]\n", proofStr)
	fmt.Printf("commitment: [%s]\n", commitmentStr)
	fmt.Printf("opening: [%s]\n", openingStr)

	return nil
}

func proveMethodCMD() *cobra.Command {
	commitmentMethodCmd := &cobra.Command{
		Use:   "proveMethod",
		Short: "Bulletproofs proveMethod command",
		Long:  "Bulletproofs proveMethod command",
		RunE: func(_ *cobra.Command, _ []string) error {
			return proveHandFunc()
		},
	}

	flags := commitmentMethodCmd.Flags()
	flags.StringVarP(&commitmentMethod, "method", "", "", "prove method: ProveAfterAddNum ProveAfterAddCommitment ProveAfterSubNum ProveAfterSubCommitment ProveAfterMulNum")
	flags.Int64VarP(&valueX, "valueX", "", -1, "valueY")
	flags.Int64VarP(&valueY, "valueY", "", -1, "valueY")
	flags.StringVarP(&commitmentXStr, "commitmentX", "", "", "")
	flags.StringVarP(&commitmentYStr, "commitmentY", "", "", "")
	flags.StringVarP(&openingXStr, "openingX", "", "", "")
	flags.StringVarP(&openingYStr, "openingY", "", "", "")

	return commitmentMethodCmd
}

func proveHandFunc() error {
	if valueX == -1 || valueY == -1 {
		return errors.New("invalid input, please check it")
	}

	if commitmentXStr == "" || openingXStr == "" {
		return errors.New("invalid input, please check it")
	}

	commitmentX, err := base64.StdEncoding.DecodeString(commitmentXStr)
	if err != nil {
		return err
	}

	openingX, err := base64.StdEncoding.DecodeString(openingXStr)
	if err != nil {
		return err
	}

	switch commitmentMethod {
	case "ProveAfterAddNum":
		return proveAfterAddNum(commitmentX, openingX)
	case "ProveAfterAddCommitment":
		return proveAfterAddCommitment(commitmentX, openingX)
	case "ProveAfterSubNum":
		return proveAfterSubNum(commitmentX, openingX)
	case "ProveAfterSubCommitment":
		return proveAfterSubCommitment(commitmentX, openingX)
	case "ProveAfterMulNum":
		return proveAfterMulNum(commitmentX, openingX)
	default:
		return errors.New("method mismatch")
	}
}

func proveAfterAddNum(commitmentX, openingX []byte) error {
	proof, commitment, err := bulletproofs.ProveAfterAddNum(uint64(valueX), uint64(valueY), openingX, commitmentX)
	if err != nil {
		return err
	}

	fmt.Printf("proof: [%s]\n", base64.StdEncoding.EncodeToString(proof))
	fmt.Printf("commtiment: [%s]\n", base64.StdEncoding.EncodeToString(commitment))
	fmt.Printf("opening: [%s]\n", openingXStr)

	return nil
}

func proveAfterAddCommitment(commitmentX, openingX []byte) error {
	if commitmentYStr == "" || openingYStr == "" {
		return errors.New("invalid input, please check it")
	}

	openingY, err := base64.StdEncoding.DecodeString(openingYStr)
	if err != nil {
		return err
	}

	commitmentY, err := base64.StdEncoding.DecodeString(commitmentYStr)
	if err != nil {
		return err
	}

	proof, commitment, opening, err := bulletproofs.ProveAfterAddCommitment(uint64(valueX), uint64(valueY), openingX, openingY, commitmentX, commitmentY)
	if err != nil {
		return err
	}

	fmt.Printf("proof: [%s]\n", base64.StdEncoding.EncodeToString(proof))
	fmt.Printf("commtiment: [%s]\n", base64.StdEncoding.EncodeToString(commitment))
	fmt.Printf("opening: [%s]\n", base64.StdEncoding.EncodeToString(opening))

	return nil
}

func proveAfterSubNum(commitmentX, openingX []byte) error {
	proof, commitment, err := bulletproofs.ProveAfterSubNum(uint64(valueX), uint64(valueY), openingX, commitmentX)
	if err != nil {
		return err
	}

	fmt.Printf("proof: [%s]\n", base64.StdEncoding.EncodeToString(proof))
	fmt.Printf("commtiment: [%s]\n", base64.StdEncoding.EncodeToString(commitment))
	fmt.Printf("opening: [%s]\n", openingXStr)

	return nil
}

func proveAfterSubCommitment(commitmentX, openingX []byte) error {
	if commitmentYStr == "" || openingYStr == "" {
		return errors.New("invalid input, please check it")
	}

	openingY, err := base64.StdEncoding.DecodeString(openingYStr)
	if err != nil {
		return err
	}

	commitmentY, err := base64.StdEncoding.DecodeString(commitmentYStr)
	if err != nil {
		return err
	}

	proof, commitment, opening, err := bulletproofs.ProveAfterSubCommitment(uint64(valueX), uint64(valueY), openingX, openingY, commitmentX, commitmentY)
	if err != nil {
		return err
	}

	fmt.Printf("proof: [%s]\n", base64.StdEncoding.EncodeToString(proof))
	fmt.Printf("commtiment: [%s]\n", base64.StdEncoding.EncodeToString(commitment))
	fmt.Printf("opening: [%s]\n", base64.StdEncoding.EncodeToString(opening))

	return nil
}

func proveAfterMulNum(commitmentX, openingX []byte) error {
	proof, commitment, opening, err := bulletproofs.ProveAfterMulNum(uint64(valueX), uint64(valueY), openingX, commitmentX)
	if err != nil {
		return err
	}

	fmt.Printf("proof: [%s]\n", base64.StdEncoding.EncodeToString(proof))
	fmt.Printf("commtiment: [%s]\n", base64.StdEncoding.EncodeToString(commitment))
	fmt.Printf("opening: [%s]\n", base64.StdEncoding.EncodeToString(opening))
	return nil
}

func pedersenVerifyCMD() *cobra.Command {
	pedersenVerifyCmd := &cobra.Command{
		Use:   "pedersenVerify",
		Short: "Bulletproofs pedersenVerify command",
		Long:  "Bulletproofs pedersenVerify command",
		RunE: func(_ *cobra.Command, _ []string) error {
			return pedersenVerify()
		},
	}

	flags := pedersenVerifyCmd.Flags()
	flags.StringVarP(&commitmentXStr, "commitment", "", "", "commitment")
	flags.StringVarP(&openingXStr, "opening", "", "", "opening")
	flags.Int64VarP(&valueX, "value", "", -1, "value")

	return pedersenVerifyCmd
}

func pedersenVerify() error {
	if valueX == -1 {
		return errors.New("invalid input, please check it")
	}

	if commitmentXStr == "" || openingXStr == "" {
		return errors.New("invalid input, please check it")
	}

	commitment, err := base64.StdEncoding.DecodeString(commitmentXStr)
	if err != nil {
		return err
	}

	opening, err := base64.StdEncoding.DecodeString(openingXStr)
	if err != nil {
		return err
	}

	ok, err := bulletproofs.PedersenVerify(commitment, opening, uint64(valueX))
	if err != nil {
		return err
	}

	if ok {
		fmt.Printf("verify: true\n")
		return nil
	} else {
		fmt.Printf("verify: false\n")
		return nil
	}
}

func pedersenNegCMD() *cobra.Command {
	negCmd := &cobra.Command{
		Use:   "neg",
		Short: "Bulletproofs pedersenNegCMD command",
		Long:  "Bulletproofs pedersenNegCMD command",
		RunE: func(_ *cobra.Command, _ []string) error {
			return pedersenNeg()
		},
	}

	flags := negCmd.Flags()
	flags.StringVarP(&pedersenNegMethod, "method", "", "", "pedersen neg method: PedersenNegCommitment PedersenNegOpening")
	flags.StringVarP(&commitmentXStr, "commitment", "", "", "")
	flags.StringVarP(&openingXStr, "opening", "", "", "")

	return negCmd
}

func pedersenNeg() error {
	if pedersenNegMethod == "PedersenNegCommitment" {
		if commitmentXStr == "" {
			return errors.New("invalid commitment, please check it")
		}
		commitment, err := base64.StdEncoding.DecodeString(commitmentXStr)
		if err != nil {
			return err
		}

		neg, err := bulletproofs.PedersenNeg(commitment)
		if err != nil {
			return err
		}

		negStr := base64.StdEncoding.EncodeToString(neg)
		fmt.Printf("commitment Neg: [%s]\n", negStr)
	} else if pedersenNegMethod == "PedersenNegOpening" {
		if openingXStr == "" {
			return errors.New("invalid commitment, please check it")
		}
		opening, err := base64.StdEncoding.DecodeString(openingXStr)
		if err != nil {
			return err
		}

		neg, err := bulletproofs.PedersenNegOpening(opening)
		if err != nil {
			return err
		}

		negStr := base64.StdEncoding.EncodeToString(neg)
		fmt.Printf("opening Neg: [%s]\n", negStr)
	} else {
		return errors.New("method mismatch")
	}

	return nil
}

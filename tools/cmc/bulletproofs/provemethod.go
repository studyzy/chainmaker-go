/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bulletproofs

import (
	"encoding/base64"
	"errors"
	"fmt"

	"chainmaker.org/chainmaker/common/v2/crypto/bulletproofs"

	"github.com/spf13/cobra"
)

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
	flags.StringVarP(&commitmentMethod, "method", "", "", "prove method: ProveAfterAddNum "+
		"ProveAfterAddCommitment ProveAfterSubNum ProveAfterSubCommitment ProveAfterMulNum")
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
	fmt.Printf("commitment: [%s]\n", base64.StdEncoding.EncodeToString(commitment))
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

	proof, commitment, opening, err := bulletproofs.ProveAfterAddCommitment(uint64(valueX), uint64(valueY),
		openingX, openingY, commitmentX, commitmentY)
	if err != nil {
		return err
	}

	fmt.Printf("proof: [%s]\n", base64.StdEncoding.EncodeToString(proof))
	fmt.Printf("commitment: [%s]\n", base64.StdEncoding.EncodeToString(commitment))
	fmt.Printf("opening: [%s]\n", base64.StdEncoding.EncodeToString(opening))

	return nil
}

func proveAfterSubNum(commitmentX, openingX []byte) error {
	proof, commitment, err := bulletproofs.ProveAfterSubNum(uint64(valueX), uint64(valueY), openingX, commitmentX)
	if err != nil {
		return err
	}

	fmt.Printf("proof: [%s]\n", base64.StdEncoding.EncodeToString(proof))
	fmt.Printf("commitment: [%s]\n", base64.StdEncoding.EncodeToString(commitment))
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

	proof, commitment, opening, err := bulletproofs.ProveAfterSubCommitment(uint64(valueX), uint64(valueY), openingX,
		openingY, commitmentX, commitmentY)
	if err != nil {
		return err
	}

	fmt.Printf("proof: [%s]\n", base64.StdEncoding.EncodeToString(proof))
	fmt.Printf("commitment: [%s]\n", base64.StdEncoding.EncodeToString(commitment))
	fmt.Printf("opening: [%s]\n", base64.StdEncoding.EncodeToString(opening))

	return nil
}

func proveAfterMulNum(commitmentX, openingX []byte) error {
	proof, commitment, opening, err := bulletproofs.ProveAfterMulNum(uint64(valueX), uint64(valueY), openingX, commitmentX)
	if err != nil {
		return err
	}

	fmt.Printf("proof: [%s]\n", base64.StdEncoding.EncodeToString(proof))
	fmt.Printf("commitment: [%s]\n", base64.StdEncoding.EncodeToString(commitment))
	fmt.Printf("opening: [%s]\n", base64.StdEncoding.EncodeToString(opening))
	return nil
}

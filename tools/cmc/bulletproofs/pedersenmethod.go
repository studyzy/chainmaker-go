package bulletproofs

import (
	"chainmaker.org/chainmaker-go/common/crypto/bulletproofs"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
)

func pedersenMethodCMD() *cobra.Command {
	pedersenMethodCmd := &cobra.Command{
		Use:   "pedersenMethod",
		Short: "Bulletproofs pedersenMethodCmd command",
		Long:  "Bulletproofs pedersenMethodCmd command",
	}

	pedersenMethodCmd.AddCommand(commitmentMethodCMD())
	pedersenMethodCmd.AddCommand(negCMD())
	pedersenMethodCmd.AddCommand(openingMethodCMD())

	return pedersenMethodCmd
}

func commitmentMethodCMD() *cobra.Command {
	commitmentMethodCmd := &cobra.Command{
		Use:   "commitmentMethod",
		Short: "Bulletproofs pedersenCommitmentMethod command",
		Long:  "Bulletproofs pedersenCommitmentMethod command",
		RunE: func(_ *cobra.Command, _ []string) error {
			return pedersenCommitmentHandleFunc()
		},
	}

	flags := commitmentMethodCmd.Flags()
	flags.StringVarP(&pedersenCommitmentMethod, "method", "", "", "pedersen commitment method: PedersenAddCommitmentWithOpening PedersenSubCommitmentWithOpening PedersenMulNumWithOpening")
	flags.Int64VarP(&valueX, "value", "", -1, "value")
	flags.StringVarP(&commitmentXStr, "commitmentX", "", "", "")
	flags.StringVarP(&commitmentYStr, "commitmentY", "", "", "")
	flags.StringVarP(&openingXStr, "openingX", "", "", "")
	flags.StringVarP(&openingYStr, "openingY", "", "", "")

	return commitmentMethodCmd
}

func negCMD() *cobra.Command {
	negCmd := &cobra.Command{
		Use:   "neg",
		Short: "Bulletproofs commitmentMethodCmd command",
		Long:  "Bulletproofs commitmentMethodCmd command",
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

func openingMethodCMD() *cobra.Command {
	openingMethodCmd := &cobra.Command{
		Use:   "openingMethod",
		Short: "Bulletproofs pedersenOpeningMethod command",
		Long:  "Bulletproofs pedersenOpeningMethod command",
		RunE: func(_ *cobra.Command, _ []string) error {
			return openingHandleFunc()
		},
	}

	flags := openingMethodCmd.Flags()
	flags.StringVarP(&pedersenOpeningMethod, "method", "", "", "pedersen opening method: PedersenAddOpening PedersenSubOpening PedersenMulOpening")
	flags.Int64VarP(&valueX, "valueX", "", -1, "valueY")
	flags.StringVarP(&openingXStr, "openingX", "", "", "")
	flags.StringVarP(&openingYStr, "openingY", "", "", "")

	return openingMethodCmd
}

func pedersenCommitmentHandleFunc() error {
	if openingXStr == "" || commitmentYStr == "" {
		return errors.New("invalid input, please check it")
	}
	var err error
	var openingX, openingY, commitmentX, commitmentY []byte
	openingX, err = base64.StdEncoding.DecodeString(openingXStr)
	commitmentX, err = base64.StdEncoding.DecodeString(commitmentXStr)
	if err != nil {
		return err
	}

	var commitment []byte
	var opening []byte
	switch pedersenCommitmentMethod {
	case "PedersenAddCommitmentWithOpening":
		if openingYStr == "" || commitmentYStr == "" {
			return errors.New("invalid input, please check it")
		}
		openingY, err = base64.StdEncoding.DecodeString(openingYStr)
		commitmentY, err = base64.StdEncoding.DecodeString(commitmentYStr)
		commitment, opening, err = bulletproofs.PedersenAddCommitmentWithOpening(commitmentX, commitmentY, openingX, openingY)
	case "PedersenSubCommitmentWithOpening":
		if openingYStr == "" || commitmentYStr == "" {
			return errors.New("invalid input, please check it")
		}
		openingY, err = base64.StdEncoding.DecodeString(openingYStr)
		commitmentY, err = base64.StdEncoding.DecodeString(commitmentYStr)
		commitment, opening, err = bulletproofs.PedersenSubCommitmentWithOpening(commitmentX, commitmentY, openingX, openingY)
	case "PedersenMulNumWithOpening":
		if openingXStr == "" || openingYStr == "" || commitmentXStr == "" || commitmentYStr == "" {
			return errors.New("invalid input, please check it")
		}
		commitment, opening, err = bulletproofs.PedersenMulNumWithOpening(commitmentX, openingX, uint64(valueX))
	default:
		return errors.New("method mismatch")
	}

	commitmentStr := base64.StdEncoding.EncodeToString(commitment)
	openingStr := base64.StdEncoding.EncodeToString(opening)

	fmt.Printf("%s:\ncommitment:[%s]\nopening:[%s]\n", pedersenCommitmentMethod, commitmentStr, openingStr)
	return nil
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

func openingHandleFunc() error {
	if openingXStr == "" {
		return errors.New("invalid input, please check it")
	}

	openingX, err := base64.StdEncoding.DecodeString(openingXStr)
	if err != nil {
		return err
	}

	var opening []byte
	switch pedersenOpeningMethod {
	case "PedersenAddOpening":
		if openingYStr == "" {
			return errors.New("invalid input, please check it")
		}
		openingY, err := base64.StdEncoding.DecodeString(openingYStr)
		if err != nil {
			return err
		}

		opening, err = bulletproofs.PedersenAddOpening(openingX, openingY)
	case "PedersenSubOpening":
		if openingYStr == "" {
			return errors.New("invalid input, please check it")
		}
		openingY, err := base64.StdEncoding.DecodeString(openingYStr)
		if err != nil {
			return err
		}

		opening, err = bulletproofs.PedersenSubOpening(openingX, openingY)
	case "PedersenMulOpening":
		if valueX == -1 {
			return errors.New("invalid input, please check it")
		}
		opening, err = bulletproofs.PedersenMulOpening(openingX, uint64(valueX))
	default:
		return errors.New("method mismatch")
	}

	if err != nil {
		return err
	}

	openingStr := base64.StdEncoding.EncodeToString(opening)
	fmt.Printf("[%s]: [%s]\n", pedersenOpeningMethod, openingStr)
	return nil
}

package bulletproofs

import (
	"chainmaker.org/chainmaker/common/crypto/bulletproofs"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
)

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

		neg, err := bulletproofs.Helper().NewBulletproofs().PedersenNeg(commitment)
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

		neg, err := bulletproofs.Helper().NewBulletproofs().PedersenNegOpening(opening)
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

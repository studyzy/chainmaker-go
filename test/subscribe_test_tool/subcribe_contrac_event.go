package main

import (
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
	"log"
)

func SubscribeContractEvent() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subscribeContractEvent",
		Short: "Subscribe ContractEvent",
		Long:  "Subscribe ContractEvent",
		RunE: func(_ *cobra.Command, _ []string) error {
			return subscribeContractEvent()
		},
	}
	return cmd
}

func subscribeContractEvent() error {
	payload := &commonPb.SubscribeContractEventPayload{
		Topic:        topic,
		ContractName: contractName,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf("marshal payload failed, %s", err.Error())
	}

	_, err = subscribeRequest(sk3, client, commonPb.TxType_SUBSCRIBE_CONTRACT_EVENT_INFO, chainId, payloadBytes)
	if err != nil {
		return err
	}

	return nil
}

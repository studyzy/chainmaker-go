package main

import (
	"log"

	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/pb-go/consts"
	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
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
	payload := &commonPb.Payload{
		Parameters: []*commonPb.KeyValuePair{
			{Key: consts.SubscribeManage_SubscribeContractEvent_TOPIC.String(), Value: []byte(topic)},
			{Key: consts.SubscribeManage_SubscribeContractEvent_CONTRACT_NAME.String(), Value: []byte(contractName)},
		},
		//Topic:        topic,
		//ContractName: contractName,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf("marshal payload failed, %s", err.Error())
	}

	_, err = subscribeRequest(sk3, client, consts.SubscribeManage_SUBSCRIBE_CONTRACT_EVENT.String(), chainId, payloadBytes)
	if err != nil {
		return err
	}

	return nil
}

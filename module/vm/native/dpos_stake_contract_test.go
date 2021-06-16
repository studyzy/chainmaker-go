package native

import (
	"github.com/stretchr/testify/require"
	"testing"
)

const (
	DelegateAddress = "GMx5CwXvH9FyGwD5CbHsCXfM6XmAyzjb9iVRDiYBTxdB"
	ValidatorAddress = "4yp3FUSrc1jyCgHMXswPeSE9N4Dnys1Hsg3NtBbzu2F4"
	NodeID = "NodeTestc1jyCgHMXswPeSE9N4Dnys1Hsg3NtBbzu2F4"
	//TotalSupply = "100000000"
	//TransferValue = "1000000"
	//TransferBigValue = "3000000"
	//ApproveValue  = "2000000"
)

func TestFormatKey(t *testing.T) {
	// test validator key
	validatorKey := "V/4yp3FUSrc1jyCgHMXswPeSE9N4Dnys1Hsg3NtBbzu2F4"
	ValidatorAddressKey := ToValidatorKey(ValidatorAddress)
	require.Equal(t, validatorKey, string(ValidatorAddressKey))

	// test validator prefix
	vPrefix := "V/"
	ValidatorPrefix := ToValidatorPrefix()
	require.Equal(t, vPrefix, string(ValidatorPrefix))

	// test delegate key
	delegationKey := "D/GMx5CwXvH9FyGwD5CbHsCXfM6XmAyzjb9iVRDiYBTxdB/4yp3FUSrc1jyCgHMXswPeSE9N4Dnys1Hsg3NtBbzu2F4"
	DelegationAddressKey := ToDelegationKey(DelegateAddress, ValidatorAddress)
	require.Equal(t, delegationKey, string(DelegationAddressKey))

	// test delegate prefix
	dPrefix := "D/GMx5CwXvH9FyGwD5CbHsCXfM6XmAyzjb9iVRDiYBTxdB"
	delegationPrefix := ToDelegationPrefix(DelegateAddress)
	require.Equal(t, dPrefix, string(delegationPrefix))

	// test epoch key
	EpochKey := ToEpochKey("99")
	epochKey := "E/99"
	require.Equal(t, epochKey, string(EpochKey))

	// test node id
	NodeIDKey := ToNodeIDKey(NodeID)
	nodeKey := "N/NodeTestc1jyCgHMXswPeSE9N4Dnys1Hsg3NtBbzu2F4"
	require.Equal(t, nodeKey, string(NodeIDKey))

	// test unbonding delegation key
	unbondingDelegationKey := "U/\u0000\u0000\u0000\u0000\u0000\u0000\u0000c/GMx5CwXvH9FyGwD5CbHsCXfM6XmAyzjb9iVRDiYBTxdB/4yp3FUSrc1jyCgHMXswPeSE9N4Dnys1Hsg3NtBbzu2F4"
	UnbondingDelegationKey := ToUnbondingDelegationKey(99, DelegateAddress, ValidatorAddress)
	require.Equal(t, unbondingDelegationKey, string(UnbondingDelegationKey))

	// test unbonding delegation prefix
	udPrefix := "U/\u0000\u0000\u0000\u0000\u0000\u0000\u0000c"
	UnbondingDelegationPrefix := ToUnbondingDelegationPrefix(99)
	require.Equal(t, udPrefix, string(UnbondingDelegationPrefix))
}

func TestStakeContractAddr(t *testing.T) {

}

func TestDPosStakeRuntime_SetNodeID(t *testing.T) {

}

func TestDPosStakeRuntime_GetAllValidator(t *testing.T) {

}

func TestDPosStakeRuntime_Delegate(t *testing.T) {

}

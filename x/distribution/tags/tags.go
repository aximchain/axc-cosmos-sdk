// nolint
package tags

import (
	sdk "github.com/aximchain/axc-cosmos-sdk/types"
)

var (
	ActionModifyWithdrawAddress       = []byte("modify-withdraw-address")
	ActionWithdrawDelegatorRewardsAll = []byte("withdraw-delegator-rewards-all")
	ActionWithdrawDelegatorReward     = []byte("withdraw-delegator-reward")
	ActionWithdrawValidatorRewardsAll = []byte("withdraw-validator-rewards-all")

	Action    = sdk.TagAction
	Validator = sdk.TagSrcValidator
	Delegator = sdk.TagDelegator
)

package oracle

import (
	sdk "github.com/aximchain/axc-cosmos-sdk/types"
	"github.com/aximchain/axc-cosmos-sdk/x/oracle/types"
)

func RegisterUpgradeBeginBlocker(keeper Keeper) {
	sdk.UpgradeMgr.RegisterBeginBlocker(sdk.LaunchAscUpgrade, func(ctx sdk.Context) {
		keeper.SetParams(ctx, types.Params{ConsensusNeeded: types.DefaultConsensusNeeded})
	})

	err := keeper.ScKeeper.RegisterChannel(types.RelayPackagesChannelName, types.RelayPackagesChannelId, nil)
	if err != nil {
		panic("register relay packages channel error")
	}
}

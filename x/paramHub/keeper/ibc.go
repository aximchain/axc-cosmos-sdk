package keeper

import (
	"github.com/aximchain/axc-cosmos-sdk/asc/rlp"
	sdk "github.com/aximchain/axc-cosmos-sdk/types"
	"github.com/aximchain/axc-cosmos-sdk/x/paramHub/types"
)

const (
	ChannelName = "params"
	ChannelId   = sdk.ChannelID(9)
)

func (keeper *Keeper) SaveParamChangeToIbc(ctx sdk.Context, sideChainId string, paramChange types.CSCParamChange) (seq uint64, sdkErr sdk.Error) {
	if keeper.ibcKeeper == nil {
		return 0, sdk.ErrInternal("the keeper is not prepared for side chain")
	}
	bz, err := rlp.EncodeToBytes(&paramChange)
	if err != nil {
		return 0, sdk.ErrInternal("failed to encode paramChange")
	}
	return keeper.ibcKeeper.CreateIBCSyncPackage(ctx, sideChainId, ChannelName, bz)
}

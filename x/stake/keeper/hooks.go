//nolint
package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Expose the hooks if present
func (k Keeper) OnValidatorCreated(ctx sdk.Context, address sdk.ValAddress) {
	if k.hooks != nil {
		k.hooks.OnValidatorCreated(ctx, address)
	}
}
func (k Keeper) OnValidatorModified(ctx sdk.Context, address sdk.ValAddress) {
	if k.hooks != nil {
		k.hooks.OnValidatorModified(ctx, address)
	}
}

func (k Keeper) OnValidatorRemoved(ctx sdk.Context, address sdk.ValAddress) {
	if k.hooks != nil {
		k.hooks.OnValidatorRemoved(ctx, address)
	}
}

func (k Keeper) OnValidatorBonded(ctx sdk.Context, address sdk.ConsAddress, operator sdk.ValAddress) {
	if k.hooks != nil {
		k.hooks.OnValidatorBonded(ctx, address, operator)
	}
}

func (k Keeper) OnSideChainValidatorBonded(ctx sdk.Context, sideConsAddr []byte, operator sdk.ValAddress) {
	if k.hooks != nil {
		k.hooks.OnSideChainValidatorBonded(ctx, sideConsAddr, operator)
	}
}

func (k Keeper) OnSelfDelDropBelowMin(ctx sdk.Context, operator sdk.ValAddress) {
	if k.hooks != nil {
		k.hooks.OnSelfDelDropBelowMin(ctx, operator)
	}
}

func (k Keeper) OnValidatorBeginUnbonding(ctx sdk.Context, address sdk.ConsAddress, operator sdk.ValAddress) {
	if k.hooks != nil {
		k.hooks.OnValidatorBeginUnbonding(ctx, address, operator)
	}
}

func (k Keeper) OnSideChainValidatorBeginUnbonding(ctx sdk.Context, sideConsAddr []byte, operator sdk.ValAddress) {
	if k.hooks != nil {
		k.hooks.OnSideChainValidatorBeginUnbonding(ctx, sideConsAddr, operator)
	}
}

func (k Keeper) OnDelegationCreated(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) {
	if k.hooks != nil {
		k.hooks.OnDelegationCreated(ctx, delAddr, valAddr)
	}
}

func (k Keeper) OnDelegationSharesModified(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) {
	if k.hooks != nil {
		k.hooks.OnDelegationSharesModified(ctx, delAddr, valAddr)
	}
}

func (k Keeper) OnDelegationRemoved(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) {
	if k.hooks != nil {
		k.hooks.OnDelegationRemoved(ctx, delAddr, valAddr)
	}
}

package sidechain

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/params"
)

// Default parameter namespace
const DefaultParamspace = "sidechain"

var (
	KeyAxcSideChainId = []byte("AxcSideChainId")
)

// ParamTypeTable for sidechain module
func ParamTypeTable() params.TypeTable {
	return params.NewTypeTable().RegisterParamSet(&Params{})
}

type Params struct {
	AxcSideChainId string `json:"axc_side_chain_id"`
}

// Implements params.ParamStruct
func (p *Params) KeyValuePairs() params.KeyValuePairs {
	return params.KeyValuePairs{
		{KeyAxcSideChainId, &p.AxcSideChainId},
	}
}

// Default parameters used by Cosmos Hub
func DefaultParams() Params {
	return Params{
		AxcSideChainId: "axc",
	}
}

func (k Keeper) AxcSideChainId(ctx sdk.Context) (sideChainId string) {
	k.paramspace.Get(ctx, KeyAxcSideChainId, &sideChainId)
	return
}

// set the params
func (k Keeper) SetParams(ctx sdk.Context, params Params) {
	k.paramspace.SetParamSet(ctx, &params)
}

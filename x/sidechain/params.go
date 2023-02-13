package sidechain

import (
	sdk "github.com/aximchain/axc-cosmos-sdk/types"
	"github.com/aximchain/axc-cosmos-sdk/x/params"
)

// Default parameter namespace
const DefaultParamspace = "sidechain"

var (
	KeyAscSideChainId = []byte("AscSideChainId")
)

// ParamTypeTable for sidechain module
func ParamTypeTable() params.TypeTable {
	return params.NewTypeTable().RegisterParamSet(&Params{})
}

type Params struct {
	AscSideChainId string `json:"asc_side_chain_id"`
}

// Implements params.ParamStruct
func (p *Params) KeyValuePairs() params.KeyValuePairs {
	return params.KeyValuePairs{
		{KeyAscSideChainId, &p.AscSideChainId},
	}
}

// Default parameters used by Cosmos Hub
func DefaultParams() Params {
	return Params{
		AscSideChainId: "asc",
	}
}

func (k Keeper) AscSideChainId(ctx sdk.Context) (sideChainId string) {
	k.paramspace.Get(ctx, KeyAscSideChainId, &sideChainId)
	return
}

// set the params
func (k Keeper) SetParams(ctx sdk.Context, params Params) {
	k.paramspace.SetParamSet(ctx, &params)
}

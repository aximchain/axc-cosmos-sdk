package ibc

import (
	"github.com/aximchain/axc-cosmos-sdk/codec"
)

func RegisterWire(cdc *codec.Codec) {
	cdc.RegisterConcrete(&Params{}, "params/IbcParamSet", nil)
}

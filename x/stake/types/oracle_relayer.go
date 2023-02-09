package types

import (
	sdk "github.com/aximchain/axc-cosmos-sdk/types"
)

type OracleRelayer struct {
	Address sdk.ValAddress `json:"address"`
	Power   int64          `json:"power"`
}

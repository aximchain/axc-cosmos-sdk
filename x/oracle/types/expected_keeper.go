package types

import (
	sdk "github.com/aximchain/axc-cosmos-sdk/types"
	"github.com/aximchain/axc-cosmos-sdk/x/stake"
)

// StakingKeeper defines the expected staking keeper
type StakingKeeper interface {
	GetValidator(ctx sdk.Context, addr sdk.ValAddress) (validator stake.Validator, found bool)
	GetLastValidatorPower(ctx sdk.Context, operator sdk.ValAddress) (power int64)
	GetLastTotalPower(ctx sdk.Context) (power int64)
	GetBondedValidatorsByPower(ctx sdk.Context) []stake.Validator
	GetOracleRelayersPower(ctx sdk.Context) map[string]int64
	CheckIsValidOracleRelayer(ctx sdk.Context, validatorAddress sdk.ValAddress) bool
}

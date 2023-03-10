package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTakeFeePoolRewards(t *testing.T) {

	// initialize
	height := int64(0)
	fp := InitialFeePool()
	vi1 := NewValidatorDistInfo(valAddr1, height)
	vi2 := NewValidatorDistInfo(valAddr2, height)
	vi3 := NewValidatorDistInfo(valAddr3, height)
	commissionRate1 := sdk.NewDecWithPrec(2, 2)
	commissionRate2 := sdk.NewDecWithPrec(3, 2)
	commissionRate3 := sdk.NewDecWithPrec(4, 2)
	validatorTokens1 := sdk.NewDecWithoutFra(10)
	validatorTokens2 := sdk.NewDecWithoutFra(40)
	validatorTokens3 := sdk.NewDecWithoutFra(50)
	totalBondedTokens := validatorTokens1.Add(validatorTokens2).Add(validatorTokens3)

	// simulate adding some stake for inflation
	height = 10
	fp.Pool = DecCoins{NewDecCoin("stake", sdk.NewDecWithoutFra(1000).RawInt())}

	vi1, fp = vi1.TakeFeePoolRewards(fp, height, totalBondedTokens, validatorTokens1, commissionRate1)
	require.True(sdk.DecEq(t, sdk.NewDecWithoutFra(900), fp.TotalValAccum.Accum))
	assert.True(sdk.DecEq(t, sdk.NewDecWithoutFra(900), fp.Pool[0].Amount))
	assert.True(sdk.DecEq(t, sdk.NewDecWithoutFra(100-2), vi1.Pool[0].Amount))
	assert.True(sdk.DecEq(t, sdk.NewDecWithoutFra(2), vi1.PoolCommission[0].Amount))

	vi2, fp = vi2.TakeFeePoolRewards(fp, height, totalBondedTokens, validatorTokens2, commissionRate2)
	require.True(sdk.DecEq(t, sdk.NewDecWithoutFra(500), fp.TotalValAccum.Accum))
	assert.True(sdk.DecEq(t, sdk.NewDecWithoutFra(500), fp.Pool[0].Amount))
	assert.True(sdk.DecEq(t, sdk.NewDecWithoutFra((400 - 12)), vi2.Pool[0].Amount))
	assert.True(sdk.DecEq(t, vi2.PoolCommission[0].Amount, sdk.NewDecWithoutFra(12)))

	// add more blocks and inflation
	height = 20
	fp.Pool[0].Amount = fp.Pool[0].Amount.Add(sdk.NewDecWithoutFra(1000))

	vi3, fp = vi3.TakeFeePoolRewards(fp, height, totalBondedTokens, validatorTokens3, commissionRate3)
	require.True(sdk.DecEq(t, sdk.NewDecWithoutFra(500), fp.TotalValAccum.Accum))
	assert.True(sdk.DecEq(t, sdk.NewDecWithoutFra(500), fp.Pool[0].Amount))
	assert.True(sdk.DecEq(t, sdk.NewDecWithoutFra(1000-40), vi3.Pool[0].Amount))
	assert.True(sdk.DecEq(t, vi3.PoolCommission[0].Amount, sdk.NewDecWithoutFra(40)))
}

func TestWithdrawCommission(t *testing.T) {

	// initialize
	height := int64(0)
	fp := InitialFeePool()
	vi := NewValidatorDistInfo(valAddr1, height)
	commissionRate := sdk.NewDecWithPrec(2, 2)
	validatorTokens := sdk.NewDecWithoutFra(10)
	totalBondedTokens := validatorTokens.Add(sdk.NewDecWithoutFra(90)) // validator-1 is 10% of total power

	// simulate adding some stake for inflation
	height = 10
	fp.Pool = DecCoins{NewDecCoin("stake", sdk.NewDecWithoutFra(1000).RawInt())}

	// for a more fun staring condition, have an non-withdraw update
	vi, fp = vi.TakeFeePoolRewards(fp, height, totalBondedTokens, validatorTokens, commissionRate)
	require.True(sdk.DecEq(t, sdk.NewDecWithoutFra(900), fp.TotalValAccum.Accum))
	assert.True(sdk.DecEq(t, sdk.NewDecWithoutFra(900), fp.Pool[0].Amount))
	assert.True(sdk.DecEq(t, sdk.NewDecWithoutFra(100-2), vi.Pool[0].Amount))
	assert.True(sdk.DecEq(t, sdk.NewDecWithoutFra(2), vi.PoolCommission[0].Amount))

	// add more blocks and inflation
	height = 20
	fp.Pool[0].Amount = fp.Pool[0].Amount.Add(sdk.NewDecWithPrec(1000, 0))

	vi, fp, commissionRecv := vi.WithdrawCommission(fp, height, totalBondedTokens, validatorTokens, commissionRate)
	require.True(sdk.DecEq(t, sdk.NewDecWithoutFra(1800), fp.TotalValAccum.Accum))
	assert.True(sdk.DecEq(t, sdk.NewDecWithoutFra(1800), fp.Pool[0].Amount))
	assert.True(sdk.DecEq(t, sdk.NewDecWithoutFra(200-4), vi.Pool[0].Amount))
	assert.Zero(t, len(vi.PoolCommission))
	assert.True(sdk.DecEq(t, sdk.NewDecWithoutFra(4), commissionRecv[0].Amount))
}

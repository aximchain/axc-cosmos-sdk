package keeper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/stake/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

// TODO integrate with test_common.go helper (CreateTestInput)
// setup helper function - creates two validators
func setupHelper(t *testing.T, amt int64) (sdk.Context, Keeper, types.Params) {

	// setup
	ctx, _, keeper := CreateTestInput(t, false, amt)
	params := keeper.GetParams(ctx)
	pool := keeper.GetPool(ctx)
	numVals := 3
	pool.LooseTokens = sdk.NewDecWithoutFra(amt * int64(numVals))

	// add numVals validators
	for i := 0; i < numVals; i++ {
		validator := types.NewValidator(addrVals[i], PKs[i], types.Description{})
		validator, pool, _ = validator.AddTokensFromDel(pool, sdk.NewDecWithoutFra(amt).RawInt())
		validator.BondIntraTxCounter = int16(i)
		pool.BondedTokens = pool.BondedTokens.Add(sdk.NewDecWithoutFra(amt))
		keeper.SetPool(ctx, pool)
		validator = TestingUpdateValidator(keeper, ctx, validator)
		keeper.SetValidatorByConsAddr(ctx, validator)
	}
	pool = keeper.GetPool(ctx)

	return ctx, keeper, params
}

//_________________________________________________________________________________

// tests Jail, Unjail
func TestRevocation(t *testing.T) {

	// setup
	ctx, keeper, _ := setupHelper(t, 10)
	addr := addrVals[0]
	consAddr := sdk.ConsAddress(PKs[0].Address())

	// initial state
	val, found := keeper.GetValidator(ctx, addr)
	require.True(t, found)
	require.False(t, val.GetJailed())

	// test jail
	keeper.Jail(ctx, consAddr)
	val, found = keeper.GetValidator(ctx, addr)
	require.True(t, found)
	require.True(t, val.GetJailed())

	// test unjail
	keeper.Unjail(ctx, consAddr)
	val, found = keeper.GetValidator(ctx, addr)
	require.True(t, found)
	require.False(t, val.GetJailed())
}

// tests slashUnbondingDelegation
func TestSlashUnbondingDelegation(t *testing.T) {
	ctx, keeper, params := setupHelper(t, 10)
	fraction := sdk.NewDecWithPrec(5, 1)

	// set an unbonding delegation
	ubd := types.UnbondingDelegation{
		DelegatorAddr:  addrDels[0],
		ValidatorAddr:  addrVals[0],
		CreationHeight: 0,
		// expiration timestamp (beyond which the unbonding delegation shouldn't be slashed)
		MinTime:        time.Unix(0, 0),
		InitialBalance: sdk.NewCoin(params.BondDenom, sdk.NewDecWithoutFra(10).RawInt()),
		Balance:        sdk.NewCoin(params.BondDenom, sdk.NewDecWithoutFra(10).RawInt()),
	}
	keeper.SetUnbondingDelegation(ctx, ubd)

	// unbonding started prior to the infraction height, stake didn't contribute
	slashAmount := keeper.slashUnbondingDelegation(ctx, ubd, 1, fraction)
	require.Equal(t, int64(0), slashAmount.RawInt())

	// after the expiration time, no longer eligible for slashing
	ctx = ctx.WithBlockHeader(abci.Header{Time: time.Unix(10, 0)})
	keeper.SetUnbondingDelegation(ctx, ubd)
	slashAmount = keeper.slashUnbondingDelegation(ctx, ubd, 0, fraction)
	require.Equal(t, int64(0), slashAmount.RawInt())

	// test valid slash, before expiration timestamp and to which stake contributed
	oldPool := keeper.GetPool(ctx)
	ctx = ctx.WithBlockHeader(abci.Header{Time: time.Unix(0, 0)})
	keeper.SetUnbondingDelegation(ctx, ubd)
	slashAmount = keeper.slashUnbondingDelegation(ctx, ubd, 0, fraction)
	require.Equal(t, sdk.NewDecWithoutFra(5), slashAmount)
	ubd, found := keeper.GetUnbondingDelegation(ctx, addrDels[0], addrVals[0])
	require.True(t, found)

	// initialbalance unchanged
	require.Equal(t, sdk.NewCoin(params.BondDenom, sdk.NewDecWithoutFra(10).RawInt()), ubd.InitialBalance)

	// balance decreased
	require.Equal(t, sdk.NewCoin(params.BondDenom, sdk.NewDecWithoutFra(5).RawInt()), ubd.Balance)
	newPool := keeper.GetPool(ctx)
	require.Equal(t, sdk.NewDecWithoutFra(5), oldPool.LooseTokens.Sub(newPool.LooseTokens))
}

// tests slashRedelegation
func TestSlashRedelegation(t *testing.T) {
	ctx, keeper, params := setupHelper(t, 10)
	fraction := sdk.NewDecWithPrec(5, 1)

	// set a redelegation
	rd := types.Redelegation{
		DelegatorAddr:    addrDels[0],
		ValidatorSrcAddr: addrVals[0],
		ValidatorDstAddr: addrVals[1],
		CreationHeight:   0,
		// expiration timestamp (beyond which the redelegation shouldn't be slashed)
		MinTime:        time.Unix(0, 0),
		SharesSrc:      sdk.NewDecWithoutFra(10),
		SharesDst:      sdk.NewDecWithoutFra(10),
		InitialBalance: sdk.NewCoin(params.BondDenom, sdk.NewDecWithoutFra(10).RawInt()),
		Balance:        sdk.NewCoin(params.BondDenom, sdk.NewDecWithoutFra(10).RawInt()),
	}
	keeper.SetRedelegation(ctx, rd)

	// set the associated delegation
	del := types.Delegation{
		DelegatorAddr: addrDels[0],
		ValidatorAddr: addrVals[1],
		Shares:        sdk.NewDecWithoutFra(10),
	}
	keeper.SetDelegation(ctx, del)

	// started redelegating prior to the current height, stake didn't contribute to infraction
	validator, found := keeper.GetValidator(ctx, addrVals[1])
	require.True(t, found)
	slashAmount := keeper.slashRedelegation(ctx, validator, rd, 1, fraction)
	require.Equal(t, int64(0), slashAmount.RawInt())

	// after the expiration time, no longer eligible for slashing
	ctx = ctx.WithBlockHeader(abci.Header{Time: time.Unix(10, 0)})
	keeper.SetRedelegation(ctx, rd)
	validator, found = keeper.GetValidator(ctx, addrVals[1])
	require.True(t, found)
	slashAmount = keeper.slashRedelegation(ctx, validator, rd, 0, fraction)
	require.Equal(t, int64(0), slashAmount.RawInt())

	// test valid slash, before expiration timestamp and to which stake contributed
	oldPool := keeper.GetPool(ctx)
	ctx = ctx.WithBlockHeader(abci.Header{Time: time.Unix(0, 0)})
	keeper.SetRedelegation(ctx, rd)
	validator, found = keeper.GetValidator(ctx, addrVals[1])
	require.True(t, found)
	slashAmount = keeper.slashRedelegation(ctx, validator, rd, 0, fraction)
	require.Equal(t, sdk.NewDecWithoutFra(5), slashAmount)
	rd, found = keeper.GetRedelegation(ctx, addrDels[0], addrVals[0], addrVals[1])
	require.True(t, found)

	// end block
	_, updates := keeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.Equal(t, 1, len(updates))

	// initialbalance unchanged
	require.Equal(t, sdk.NewCoin(params.BondDenom, sdk.NewDecWithoutFra(10).RawInt()), rd.InitialBalance)

	// balance decreased
	require.Equal(t, sdk.NewCoin(params.BondDenom, sdk.NewDecWithoutFra(5).RawInt()), rd.Balance)

	// shares decreased
	del, found = keeper.GetDelegation(ctx, addrDels[0], addrVals[1])
	require.True(t, found)
	require.Equal(t, sdk.NewDecWithoutFra(5), del.Shares)

	// pool bonded tokens decreased
	newPool := keeper.GetPool(ctx)
	require.Equal(t, sdk.NewDecWithoutFra(5), oldPool.BondedTokens.Sub(newPool.BondedTokens))
}

// tests Slash at a future height (must panic)
func TestSlashAtFutureHeight(t *testing.T) {
	ctx, keeper, _ := setupHelper(t, 10)
	consAddr := sdk.ConsAddress(PKs[0].Address())
	fraction := sdk.NewDecWithPrec(5, 1)
	require.Panics(t, func() { keeper.Slash(ctx, consAddr, 1, sdk.NewDecWithoutFra(10).RawInt(), fraction) })
}

// test slash at a negative height
// this just represents pre-genesis and should have the same effect as slashing at height 0
func TestSlashAtNegativeHeight(t *testing.T) {
	ctx, keeper, _ := setupHelper(t, 10)
	consAddr := sdk.ConsAddress(PKs[0].Address())
	fraction := sdk.NewDecWithPrec(5, 1)

	oldPool := keeper.GetPool(ctx)
	validator, found := keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.True(t, found)
	keeper.Slash(ctx, consAddr, -2, sdk.NewDecWithoutFra(10).RawInt(), fraction)

	// read updated state
	validator, found = keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.True(t, found)
	newPool := keeper.GetPool(ctx)

	// end block
	_, updates := keeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.Equal(t, 1, len(updates), "cons addr: %v, updates: %v", []byte(consAddr), updates)

	validator = keeper.mustGetValidator(ctx, validator.OperatorAddr)
	// power decreased
	require.Equal(t, sdk.NewDecWithoutFra(5), validator.GetPower())
	// pool bonded shares decreased
	require.Equal(t, sdk.NewDecWithoutFra(5), oldPool.BondedTokens.Sub(newPool.BondedTokens))
}

// tests Slash at the current height
func TestSlashValidatorAtCurrentHeight(t *testing.T) {
	ctx, keeper, _ := setupHelper(t, 10)
	consAddr := sdk.ConsAddress(PKs[0].Address())
	fraction := sdk.NewDecWithPrec(5, 1)

	oldPool := keeper.GetPool(ctx)
	validator, found := keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.True(t, found)
	keeper.Slash(ctx, consAddr, ctx.BlockHeight(), sdk.NewDecWithoutFra(10).RawInt(), fraction)

	// read updated state
	validator, found = keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.True(t, found)
	newPool := keeper.GetPool(ctx)

	// end block
	_, updates := keeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.Equal(t, 1, len(updates), "cons addr: %v, updates: %v", []byte(consAddr), updates)

	validator = keeper.mustGetValidator(ctx, validator.OperatorAddr)
	// power decreased
	require.Equal(t, sdk.NewDecWithoutFra(5), validator.GetPower())
	// pool bonded shares decreased
	require.Equal(t, sdk.NewDecWithoutFra(5), oldPool.BondedTokens.Sub(newPool.BondedTokens))
}

// tests Slash at a previous height with an unbonding delegation
func TestSlashWithUnbondingDelegation(t *testing.T) {
	ctx, keeper, params := setupHelper(t, 10)
	consAddr := sdk.ConsAddress(PKs[0].Address())
	fraction := sdk.NewDecWithPrec(5, 1)

	// set an unbonding delegation
	ubd := types.UnbondingDelegation{
		DelegatorAddr:  addrDels[0],
		ValidatorAddr:  addrVals[0],
		CreationHeight: 11,
		// expiration timestamp (beyond which the unbonding delegation shouldn't be slashed)
		MinTime:        time.Unix(0, 0),
		InitialBalance: sdk.NewCoin(params.BondDenom, sdk.NewDecWithoutFra(4).RawInt()),
		Balance:        sdk.NewCoin(params.BondDenom, sdk.NewDecWithoutFra(4).RawInt()),
	}
	keeper.SetUnbondingDelegation(ctx, ubd)

	// slash validator for the first time
	ctx = ctx.WithBlockHeight(12)
	oldPool := keeper.GetPool(ctx)
	validator, found := keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.True(t, found)
	keeper.Slash(ctx, consAddr, 10, sdk.NewDecWithoutFra(10).RawInt(), fraction)

	// end block
	_, updates := keeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.Equal(t, 1, len(updates))

	// read updating unbonding delegation
	ubd, found = keeper.GetUnbondingDelegation(ctx, addrDels[0], addrVals[0])
	require.True(t, found)
	// balance decreased
	require.Equal(t, sdk.NewDecWithoutFra(2).RawInt(), ubd.Balance.Amount)
	// read updated pool
	newPool := keeper.GetPool(ctx)
	// bonded tokens burned
	require.Equal(t, sdk.NewDecWithoutFra(3), oldPool.BondedTokens.Sub(newPool.BondedTokens))
	// read updated validator
	validator, found = keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.True(t, found)
	// power decreased by 3 - 6 stake originally bonded at the time of infraction
	// was still bonded at the time of discovery and was slashed by half, 4 stake
	// bonded at the time of discovery hadn't been bonded at the time of infraction
	// and wasn't slashed
	require.Equal(t, sdk.NewDecWithoutFra(7), validator.GetPower())

	// slash validator again
	ctx = ctx.WithBlockHeight(13)
	keeper.Slash(ctx, consAddr, 9, sdk.NewDecWithoutFra(10).RawInt(), fraction)
	ubd, found = keeper.GetUnbondingDelegation(ctx, addrDels[0], addrVals[0])
	require.True(t, found)
	// balance decreased again
	require.Equal(t, int64(0), ubd.Balance.Amount)
	// read updated pool
	newPool = keeper.GetPool(ctx)
	// bonded tokens burned again
	require.Equal(t, sdk.NewDecWithoutFra(6), oldPool.BondedTokens.Sub(newPool.BondedTokens))
	// read updated validator
	validator, found = keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.True(t, found)
	// power decreased by 3 again
	require.Equal(t, sdk.NewDecWithoutFra(4), validator.GetPower())

	// slash validator again
	// all originally bonded stake has been slashed, so this will have no effect
	// on the unbonding delegation, but it will slash stake bonded since the infraction
	// this may not be the desirable behaviour, ref https://github.com/cosmos/cosmos-sdk/issues/1440
	ctx = ctx.WithBlockHeight(13)
	keeper.Slash(ctx, consAddr, 9, sdk.NewDecWithoutFra(10).RawInt(), fraction)
	ubd, found = keeper.GetUnbondingDelegation(ctx, addrDels[0], addrVals[0])
	require.True(t, found)
	// balance unchanged
	require.Equal(t, int64(0), ubd.Balance.Amount)
	// read updated pool
	newPool = keeper.GetPool(ctx)
	// bonded tokens burned again
	require.Equal(t, sdk.NewDecWithoutFra(9), oldPool.BondedTokens.Sub(newPool.BondedTokens))
	// read updated validator
	validator, found = keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.True(t, found)
	// power decreased by 3 again
	require.Equal(t, sdk.NewDecWithoutFra(1), validator.GetPower())

	// slash validator again
	// all originally bonded stake has been slashed, so this will have no effect
	// on the unbonding delegation, but it will slash stake bonded since the infraction
	// this may not be the desirable behaviour, ref https://github.com/cosmos/cosmos-sdk/issues/1440
	ctx = ctx.WithBlockHeight(13)
	keeper.Slash(ctx, consAddr, 9, sdk.NewDecWithoutFra(10).RawInt(), fraction)
	ubd, found = keeper.GetUnbondingDelegation(ctx, addrDels[0], addrVals[0])
	require.True(t, found)
	// balance unchanged
	require.Equal(t, int64(0), ubd.Balance.Amount)
	// read updated pool
	newPool = keeper.GetPool(ctx)
	// just 1 bonded token burned again since that's all the validator now has
	require.Equal(t, sdk.NewDecWithoutFra(10), oldPool.BondedTokens.Sub(newPool.BondedTokens))
	// apply TM updates
	keeper.ApplyAndReturnValidatorSetUpdates(ctx)
	// read updated validator
	// power decreased by 1 again, validator is out of stake
	// ergo validator should have been removed from the store
	_, found = keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.False(t, found)
}

// tests Slash at a previous height with a redelegation
func TestSlashWithRedelegation(t *testing.T) {
	ctx, keeper, params := setupHelper(t, 10)
	consAddr := sdk.ConsAddress(PKs[0].Address())
	fraction := sdk.NewDecWithPrec(5, 1)

	// set a redelegation
	rd := types.Redelegation{
		DelegatorAddr:    addrDels[0],
		ValidatorSrcAddr: addrVals[0],
		ValidatorDstAddr: addrVals[1],
		CreationHeight:   11,
		MinTime:          time.Unix(0, 0),
		SharesSrc:        sdk.NewDecWithoutFra(6),
		SharesDst:        sdk.NewDecWithoutFra(6),
		InitialBalance:   sdk.NewCoin(params.BondDenom, sdk.NewDecWithoutFra(6).RawInt()),
		Balance:          sdk.NewCoin(params.BondDenom, sdk.NewDecWithoutFra(6).RawInt()),
	}
	keeper.SetRedelegation(ctx, rd)

	// set the associated delegation
	del := types.Delegation{
		DelegatorAddr: addrDels[0],
		ValidatorAddr: addrVals[1],
		Shares:        sdk.NewDecWithoutFra(6),
	}
	keeper.SetDelegation(ctx, del)

	// update bonded tokens
	pool := keeper.GetPool(ctx)
	pool.BondedTokens = pool.BondedTokens.Add(sdk.NewDecWithoutFra(6))
	keeper.SetPool(ctx, pool)

	// slash validator
	ctx = ctx.WithBlockHeight(12)
	oldPool := keeper.GetPool(ctx)
	validator, found := keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.True(t, found)
	keeper.Slash(ctx, consAddr, 10, sdk.NewDecWithoutFra(10).RawInt(), fraction)

	// read updating redelegation
	rd, found = keeper.GetRedelegation(ctx, addrDels[0], addrVals[0], addrVals[1])
	require.True(t, found)
	// balance decreased
	require.Equal(t, sdk.NewDecWithoutFra(3).RawInt(), rd.Balance.Amount)
	// read updated pool
	newPool := keeper.GetPool(ctx)
	// bonded tokens burned
	require.Equal(t, sdk.NewDecWithoutFra(5), oldPool.BondedTokens.Sub(newPool.BondedTokens))
	// read updated validator
	validator, found = keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.True(t, found)
	// power decreased by 2 - 4 stake originally bonded at the time of infraction
	// was still bonded at the time of discovery and was slashed by half, 4 stake
	// bonded at the time of discovery hadn't been bonded at the time of infraction
	// and wasn't slashed
	require.Equal(t, sdk.NewDecWithoutFra(8), validator.GetPower())

	// slash the validator again
	ctx = ctx.WithBlockHeight(12)
	validator, found = keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.True(t, found)
	require.NotPanics(t, func() { keeper.Slash(ctx, consAddr, 10, sdk.NewDecWithoutFra(10).RawInt(), sdk.OneDec()) })

	// read updating redelegation
	rd, found = keeper.GetRedelegation(ctx, addrDels[0], addrVals[0], addrVals[1])
	require.True(t, found)
	// balance decreased, now zero
	require.Equal(t, int64(0), rd.Balance.Amount)
	// read updated pool
	newPool = keeper.GetPool(ctx)
	// seven bonded tokens burned
	require.Equal(t, sdk.NewDecWithoutFra(12), oldPool.BondedTokens.Sub(newPool.BondedTokens))
	// read updated validator
	validator, found = keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.True(t, found)
	// power decreased by 4
	require.Equal(t, sdk.NewDecWithoutFra(4), validator.GetPower())

	// slash the validator again, by 100%
	ctx = ctx.WithBlockHeight(12)
	validator, found = keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.True(t, found)
	keeper.Slash(ctx, consAddr, 10, sdk.NewDecWithoutFra(10).RawInt(), sdk.OneDec())

	// read updating redelegation
	rd, found = keeper.GetRedelegation(ctx, addrDels[0], addrVals[0], addrVals[1])
	require.True(t, found)
	// balance still zero
	require.Equal(t, int64(0), rd.Balance.Amount)
	// read updated pool
	newPool = keeper.GetPool(ctx)
	// four more bonded tokens burned
	require.Equal(t, sdk.NewDecWithoutFra(16), oldPool.BondedTokens.Sub(newPool.BondedTokens))
	// apply TM updates
	keeper.ApplyAndReturnValidatorSetUpdates(ctx)
	// read updated validator
	// validator decreased to zero power, should have been removed from the store
	_, found = keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.False(t, found)

	// slash the validator again, by 100%
	// no stake remains to be slashed
	ctx = ctx.WithBlockHeight(12)
	// validator no longer in the store
	_, found = keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.False(t, found)
	keeper.Slash(ctx, consAddr, 10, sdk.NewDecWithoutFra(10).RawInt(), sdk.OneDec())

	// read updating redelegation
	rd, found = keeper.GetRedelegation(ctx, addrDels[0], addrVals[0], addrVals[1])
	require.True(t, found)
	// balance still zero
	require.Equal(t, int64(0), rd.Balance.Amount)
	// read updated pool
	newPool = keeper.GetPool(ctx)
	// no more bonded tokens burned
	require.Equal(t, sdk.NewDecWithoutFra(16), oldPool.BondedTokens.Sub(newPool.BondedTokens))
	// read updated validator
	// power still zero, still not in the store
	_, found = keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.False(t, found)
}

// tests Slash at a previous height with both an unbonding delegation and a redelegation
func TestSlashBoth(t *testing.T) {
	ctx, keeper, params := setupHelper(t, 10)
	fraction := sdk.NewDecWithPrec(5, 1)

	// set a redelegation
	rdA := types.Redelegation{
		DelegatorAddr:    addrDels[0],
		ValidatorSrcAddr: addrVals[0],
		ValidatorDstAddr: addrVals[1],
		CreationHeight:   11,
		// expiration timestamp (beyond which the redelegation shouldn't be slashed)
		MinTime:        time.Unix(0, 0),
		SharesSrc:      sdk.NewDecWithoutFra(6),
		SharesDst:      sdk.NewDecWithoutFra(6),
		InitialBalance: sdk.NewCoin(params.BondDenom, sdk.NewDecWithoutFra(6).RawInt()),
		Balance:        sdk.NewCoin(params.BondDenom, sdk.NewDecWithoutFra(6).RawInt()),
	}
	keeper.SetRedelegation(ctx, rdA)

	// set the associated delegation
	delA := types.Delegation{
		DelegatorAddr: addrDels[0],
		ValidatorAddr: addrVals[1],
		Shares:        sdk.NewDecWithoutFra(6),
	}
	keeper.SetDelegation(ctx, delA)

	// set an unbonding delegation
	ubdA := types.UnbondingDelegation{
		DelegatorAddr:  addrDels[0],
		ValidatorAddr:  addrVals[0],
		CreationHeight: 11,
		// expiration timestamp (beyond which the unbonding delegation shouldn't be slashed)
		MinTime:        time.Unix(0, 0),
		InitialBalance: sdk.NewCoin(params.BondDenom, sdk.NewDecWithoutFra(4).RawInt()),
		Balance:        sdk.NewCoin(params.BondDenom, sdk.NewDecWithoutFra(4).RawInt()),
	}
	keeper.SetUnbondingDelegation(ctx, ubdA)

	// slash validator
	ctx = ctx.WithBlockHeight(12)
	oldPool := keeper.GetPool(ctx)
	validator, found := keeper.GetValidatorByConsAddr(ctx, sdk.GetConsAddress(PKs[0]))
	require.True(t, found)
	consAddr0 := sdk.ConsAddress(PKs[0].Address())
	keeper.Slash(ctx, consAddr0, 10, sdk.NewDecWithoutFra(10).RawInt(), fraction)

	// read updating redelegation
	rdA, found = keeper.GetRedelegation(ctx, addrDels[0], addrVals[0], addrVals[1])
	require.True(t, found)
	// balance decreased
	require.Equal(t, sdk.NewDecWithoutFra(3).RawInt(), rdA.Balance.Amount)
	// read updated pool
	newPool := keeper.GetPool(ctx)
	// loose tokens burned
	require.Equal(t, sdk.NewDecWithoutFra(2), oldPool.LooseTokens.Sub(newPool.LooseTokens))
	// bonded tokens burned
	require.Equal(t, sdk.NewDecWithoutFra(3), oldPool.BondedTokens.Sub(newPool.BondedTokens))
	// read updated validator
	validator, found = keeper.GetValidatorByConsAddr(ctx, sdk.GetConsAddress(PKs[0]))
	require.True(t, found)
	// power not decreased, all stake was bonded since
	require.Equal(t, sdk.NewDecWithoutFra(10), validator.GetPower())
}

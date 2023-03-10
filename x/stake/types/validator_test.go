package types

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tmtypes "github.com/tendermint/tendermint/types"
)

func TestValidatorEqual(t *testing.T) {
	val1 := NewValidator(addr1, pk1, Description{})
	val2 := NewValidator(addr1, pk1, Description{})

	ok := val1.Equal(val2)
	require.True(t, ok)

	val2 = NewValidator(addr2, pk2, Description{})

	ok = val1.Equal(val2)
	require.False(t, ok)
}

func TestUpdateDescription(t *testing.T) {
	d1 := Description{
		Moniker: "d1",
		Website: "https://validator.cosmos",
		Details: "Test validator",
	}

	d2 := Description{
		Moniker:  DoNotModifyDesc,
		Identity: DoNotModifyDesc,
		Website:  DoNotModifyDesc,
		Details:  DoNotModifyDesc,
	}

	d3 := Description{
		Moniker:  "d3",
		Identity: "",
		Website:  "",
		Details:  "",
	}

	d, err := d1.UpdateDescription(d2)
	require.Nil(t, err)
	require.Equal(t, d, d1)

	d, err = d1.UpdateDescription(d3)
	require.Nil(t, err)
	require.Equal(t, d, d3)
}

func TestABCIValidatorUpdate(t *testing.T) {
	validator := NewValidator(addr1, pk1, Description{})

	abciVal := validator.ABCIValidatorUpdate()
	require.Equal(t, tmtypes.TM2PB.PubKey(validator.ConsPubKey), abciVal.PubKey)
	require.Equal(t, validator.BondedTokens().RawInt(), abciVal.Power)
}

func TestABCIValidatorUpdateZero(t *testing.T) {
	validator := NewValidator(addr1, pk1, Description{})

	abciVal := validator.ABCIValidatorUpdateZero()
	require.Equal(t, tmtypes.TM2PB.PubKey(validator.ConsPubKey), abciVal.PubKey)
	require.Equal(t, int64(0), abciVal.Power)
}

func TestRemoveTokens(t *testing.T) {

	validator := Validator{
		OperatorAddr:    addr1,
		ConsPubKey:      pk1,
		Status:          sdk.Bonded,
		Tokens:          sdk.NewDecWithoutFra(100),
		DelegatorShares: sdk.NewDecWithoutFra(100),
	}

	pool := InitialPool()
	pool.LooseTokens = sdk.NewDecWithoutFra(10)
	pool.BondedTokens = validator.BondedTokens()

	validator, pool = validator.UpdateStatus(pool, sdk.Bonded)
	require.Equal(t, sdk.Bonded, validator.Status)

	// remove tokens and test check everything
	validator, pool = validator.RemoveTokens(pool, sdk.NewDecWithoutFra(10))
	require.Equal(t, sdk.NewDecWithoutFra(90), validator.Tokens)
	require.Equal(t, sdk.NewDecWithoutFra(90), pool.BondedTokens)
	require.Equal(t, sdk.NewDecWithoutFra(20), pool.LooseTokens)

	// update validator to unbonded and remove some more tokens
	validator, pool = validator.UpdateStatus(pool, sdk.Unbonded)
	require.Equal(t, sdk.Unbonded, validator.Status)
	require.Equal(t, int64(0), pool.BondedTokens.RawInt())
	require.Equal(t, sdk.NewDecWithoutFra(110), pool.LooseTokens)

	validator, pool = validator.RemoveTokens(pool, sdk.NewDecWithoutFra(10))
	require.Equal(t, sdk.NewDecWithoutFra(80), validator.Tokens)
	require.Equal(t, int64(0), pool.BondedTokens.RawInt())
	require.Equal(t, sdk.NewDecWithoutFra(110), pool.LooseTokens)
}

func TestAddTokensValidatorBonded(t *testing.T) {
	pool := InitialPool()
	pool.LooseTokens = sdk.NewDecWithoutFra(10)
	validator := NewValidator(addr1, pk1, Description{})
	validator, pool = validator.UpdateStatus(pool, sdk.Bonded)
	validator, pool, delShares := validator.AddTokensFromDel(pool, sdk.NewDecWithoutFra(10).RawInt())

	require.Equal(t, sdk.OneDec(), validator.DelegatorShareExRate())

	assert.True(sdk.DecEq(t, sdk.NewDecWithoutFra(10), delShares))
	assert.True(sdk.DecEq(t, sdk.NewDecWithoutFra(10), validator.BondedTokens()))
}

func TestAddTokensValidatorUnbonding(t *testing.T) {
	pool := InitialPool()
	pool.LooseTokens = sdk.NewDecWithoutFra(10)
	validator := NewValidator(addr1, pk1, Description{})
	validator, pool = validator.UpdateStatus(pool, sdk.Unbonding)
	validator, pool, delShares := validator.AddTokensFromDel(pool, sdk.NewDecWithoutFra(10).RawInt())

	require.Equal(t, sdk.OneDec(), validator.DelegatorShareExRate())

	assert.True(sdk.DecEq(t, sdk.NewDecWithoutFra(10), delShares))
	assert.Equal(t, sdk.Unbonding, validator.Status)
	assert.True(sdk.DecEq(t, sdk.NewDecWithoutFra(10), validator.Tokens))
}

func TestAddTokensValidatorUnbonded(t *testing.T) {
	pool := InitialPool()
	pool.LooseTokens = sdk.NewDecWithoutFra(10)
	validator := NewValidator(addr1, pk1, Description{})
	validator, pool = validator.UpdateStatus(pool, sdk.Unbonded)
	validator, pool, delShares := validator.AddTokensFromDel(pool, sdk.NewDecWithoutFra(10).RawInt())

	require.Equal(t, sdk.OneDec(), validator.DelegatorShareExRate())

	assert.True(sdk.DecEq(t, sdk.NewDecWithoutFra(10), delShares))
	assert.Equal(t, sdk.Unbonded, validator.Status)
	assert.True(sdk.DecEq(t, sdk.NewDecWithoutFra(10), validator.Tokens))
}

// TODO refactor to make simpler like the AddToken tests above
func TestRemoveDelShares(t *testing.T) {
	valA := Validator{
		OperatorAddr:    addr1,
		ConsPubKey:      pk1,
		Status:          sdk.Bonded,
		Tokens:          sdk.NewDecWithoutFra(100),
		DelegatorShares: sdk.NewDecWithoutFra(100),
	}
	poolA := InitialPool()
	poolA.LooseTokens = sdk.NewDecWithoutFra(10)
	poolA.BondedTokens = valA.BondedTokens()
	require.Equal(t, valA.DelegatorShareExRate(), sdk.OneDec())

	// Remove delegator shares
	valB, poolB, coinsB := valA.RemoveDelShares(poolA, sdk.NewDecWithoutFra(10))
	assert.Equal(t, sdk.NewDecWithoutFra(10).RawInt(), coinsB.RawInt())
	assert.Equal(t, sdk.NewDecWithoutFra(90).RawInt(), valB.DelegatorShares.RawInt())
	assert.Equal(t, sdk.NewDecWithoutFra(90).RawInt(), valB.BondedTokens().RawInt())
	assert.Equal(t, sdk.NewDecWithoutFra(90).RawInt(), poolB.BondedTokens.RawInt())
	assert.Equal(t, sdk.NewDecWithoutFra(20).RawInt(), poolB.LooseTokens.RawInt())

	// conservation of tokens
	require.True(sdk.DecEq(t,
		poolB.LooseTokens.Add(poolB.BondedTokens),
		poolA.LooseTokens.Add(poolA.BondedTokens)))

	// specific case from random tests
	poolTokens := sdk.NewDecWithoutFra(5102)
	delShares := sdk.NewDecWithoutFra(115)
	validator := Validator{
		OperatorAddr:    addr1,
		ConsPubKey:      pk1,
		Status:          sdk.Bonded,
		Tokens:          poolTokens,
		DelegatorShares: delShares,
	}
	pool := Pool{
		BondedTokens: sdk.NewDecWithoutFra(248305),
		LooseTokens:  sdk.NewDecWithoutFra(232147),
	}
	shares := sdk.NewDecWithoutFra(29)
	_, newPool, tokens := validator.RemoveDelShares(pool, shares)

	exp, err := sdk.NewDecFromStr("128659130434")
	require.NoError(t, err)

	require.True(sdk.DecEq(t, exp, tokens))

	require.True(sdk.DecEq(t,
		newPool.LooseTokens.Add(newPool.BondedTokens),
		pool.LooseTokens.Add(pool.BondedTokens)))
}

func TestUpdateStatus(t *testing.T) {
	pool := InitialPool()
	pool.LooseTokens = sdk.NewDecWithoutFra(100)

	validator := NewValidator(addr1, pk1, Description{})
	validator, pool, _ = validator.AddTokensFromDel(pool, sdk.NewDecWithoutFra(100).RawInt())
	require.Equal(t, sdk.Unbonded, validator.Status)
	require.Equal(t, sdk.NewDecWithoutFra(100), validator.Tokens)
	require.Equal(t, int64(0), pool.BondedTokens.RawInt())
	require.Equal(t, sdk.NewDecWithoutFra(100), pool.LooseTokens)

	validator, pool = validator.UpdateStatus(pool, sdk.Bonded)
	require.Equal(t, sdk.Bonded, validator.Status)
	require.Equal(t, sdk.NewDecWithoutFra(100), validator.Tokens)
	require.Equal(t, sdk.NewDecWithoutFra(100), pool.BondedTokens)
	require.Equal(t, int64(0), pool.LooseTokens.RawInt())

	validator, pool = validator.UpdateStatus(pool, sdk.Unbonding)
	require.Equal(t, sdk.Unbonding, validator.Status)
	require.Equal(t, sdk.NewDecWithoutFra(100), validator.Tokens)
	require.Equal(t, int64(0), pool.BondedTokens.RawInt())
	require.Equal(t, sdk.NewDecWithoutFra(100), pool.LooseTokens)
}

func TestPossibleOverflow(t *testing.T) {
	poolTokens := sdk.NewDecWithoutFra(2159)
	delShares := sdk.NewDecWithoutFra(39143257068).Quo(sdk.NewDecWithoutFra(4011301))
	validator := Validator{
		OperatorAddr:    addr1,
		ConsPubKey:      pk1,
		Status:          sdk.Bonded,
		Tokens:          poolTokens,
		DelegatorShares: delShares,
	}
	pool := Pool{
		LooseTokens:  sdk.NewDecWithoutFra(100),
		BondedTokens: poolTokens,
	}
	tokens := int64(71)
	msg := fmt.Sprintf("validator %#v", validator)
	newValidator, _, _ := validator.AddTokensFromDel(pool, sdk.NewDecWithoutFra(tokens).RawInt())

	msg = fmt.Sprintf("Added %d tokens to %s", tokens, msg)
	require.False(t, newValidator.DelegatorShareExRate().LT(sdk.ZeroDec()),
		"Applying operation \"%s\" resulted in negative DelegatorShareExRate(): %v",
		msg, newValidator.DelegatorShareExRate())
}

func TestHumanReadableString(t *testing.T) {
	validator := NewValidator(addr1, pk1, Description{})

	// NOTE: Being that the validator's keypair is random, we cannot test the
	// actual contents of the string.
	valStr, err := validator.HumanReadableString()
	require.Nil(t, err)
	require.NotEmpty(t, valStr)
}

func TestValidatorMarshalUnmarshalJSON(t *testing.T) {
	validator := NewValidator(addr1, pk1, Description{})
	js, err := codec.Cdc.MarshalJSON(validator)
	require.NoError(t, err)
	require.NotEmpty(t, js)
	require.Contains(t, string(js), "\"consensus_pubkey\":\"cosmosvalconspu")
	got := &Validator{}
	err = codec.Cdc.UnmarshalJSON(js, got)
	assert.NoError(t, err)
	assert.Equal(t, validator, *got)
}

func TestValidatorSetInitialCommission(t *testing.T) {
	val := NewValidator(addr1, pk1, Description{})
	testCases := []struct {
		validator   Validator
		commission  Commission
		expectedErr bool
	}{
		{val, NewCommission(sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec()), false},
		{val, NewCommission(sdk.ZeroDec(), sdk.NewDecWithPrec(-1, 1), sdk.ZeroDec()), true},
		{val, NewCommission(sdk.ZeroDec(), sdk.NewDecWithoutFra(15000000000), sdk.ZeroDec()), true},
		{val, NewCommission(sdk.NewDecWithPrec(-1, 1), sdk.ZeroDec(), sdk.ZeroDec()), true},
		{val, NewCommission(sdk.NewDecWithPrec(2, 1), sdk.NewDecWithPrec(1, 1), sdk.ZeroDec()), true},
		{val, NewCommission(sdk.ZeroDec(), sdk.ZeroDec(), sdk.NewDecWithPrec(-1, 1)), true},
		{val, NewCommission(sdk.ZeroDec(), sdk.NewDecWithPrec(1, 1), sdk.NewDecWithPrec(2, 1)), true},
	}

	for i, tc := range testCases {
		val, err := tc.validator.SetInitialCommission(tc.commission)

		if tc.expectedErr {
			require.Error(t, err,
				"expected error for test case #%d with commission: %s", i, tc.commission,
			)
		} else {
			require.NoError(t, err,
				"unexpected error for test case #%d with commission: %s", i, tc.commission,
			)
			require.Equal(t, tc.commission, val.Commission,
				"invalid validator commission for test case #%d with commission: %s", i, tc.commission,
			)
		}
	}
}

func TestMarshalValidator(t *testing.T) {
	validator := NewValidator(addr1, pk1, Description{})
	validator.Tokens = sdk.NewDec(100)
	validator.DelegatorShares = sdk.NewDec(100)
	validator.SideConsAddr = randAddr(t, 20)
	bz := MustMarshalValidator(MsgCdc, validator)
	getVal, err := UnmarshalValidator(MsgCdc, bz)
	require.Nil(t, err)
	require.EqualValues(t, validator.FeeAddr, getVal.FeeAddr)
	require.EqualValues(t, validator.OperatorAddr, getVal.OperatorAddr)
	require.EqualValues(t, validator.ConsPubKey, getVal.ConsPubKey)
	require.EqualValues(t, validator.Tokens, getVal.Tokens)
	require.EqualValues(t, validator.DelegatorShares, getVal.DelegatorShares)
	require.EqualValues(t, validator.Jailed, getVal.Jailed)
	require.EqualValues(t, validator.Status, getVal.Status)
	require.EqualValues(t, validator.SideConsAddr, getVal.SideConsAddr)
}

func randAddr(t *testing.T, size int64) []byte {
	addr := make([]byte, size)
	n, err := rand.Read(addr)
	require.NoError(t, err)
	require.Equal(t, 20, n)
	require.Equal(t, 20, len(addr))
	return addr
}

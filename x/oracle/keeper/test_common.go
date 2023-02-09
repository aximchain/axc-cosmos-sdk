package keeper

import (
	"testing"

	"github.com/aximchain/axc-cosmos-sdk/x/ibc"
	"github.com/aximchain/axc-cosmos-sdk/x/sidechain"

	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"

	sdk "github.com/aximchain/axc-cosmos-sdk/types"
	"github.com/aximchain/axc-cosmos-sdk/x/bank"
	"github.com/aximchain/axc-cosmos-sdk/x/gov"
	"github.com/aximchain/axc-cosmos-sdk/x/mock"
	"github.com/aximchain/axc-cosmos-sdk/x/params"
	"github.com/aximchain/axc-cosmos-sdk/x/stake"
)

// initialize the mock application for this module
func getMockApp(t *testing.T, numGenAccs int) (*mock.App, bank.BaseKeeper, Keeper, stake.Keeper, []sdk.AccAddress, []crypto.PubKey, []crypto.PrivKey) {
	mapp := mock.NewApp()

	stake.RegisterCodec(mapp.Cdc)

	keyGlobalParams := sdk.NewKVStoreKey("params")
	tkeyGlobalParams := sdk.NewTransientStoreKey("transient_params")
	keyStake := sdk.NewKVStoreKey("stake")
	keyStakeReward := sdk.NewKVStoreKey("stake_reward")
	tkeyStake := sdk.NewTransientStoreKey("transient_stake")
	keyOracle := sdk.NewKVStoreKey("oracle")
	keyIbc := sdk.NewKVStoreKey("ibc")
	keySideChain := sdk.NewKVStoreKey("side")

	pk := params.NewKeeper(mapp.Cdc, keyGlobalParams, tkeyGlobalParams)
	ck := bank.NewBaseKeeper(mapp.AccountKeeper)
	sk := stake.NewKeeper(mapp.Cdc, keyStake, keyStakeReward, tkeyStake, ck, nil, pk.Subspace(stake.DefaultParamspace), mapp.RegisterCodespace(stake.DefaultCodespace), sdk.ChainID(0), "")
	scK := sidechain.NewKeeper(keySideChain, pk.Subspace(sidechain.DefaultParamspace), mapp.Cdc)
	ibcKeeper := ibc.NewKeeper(keyIbc, pk.Subspace(ibc.DefaultParamspace), ibc.DefaultCodespace, scK)

	mapp.SetInitChainer(getInitChainer(mapp, sk))

	require.NoError(t, mapp.CompleteSetup(keyStake, tkeyStake, keyOracle, keyGlobalParams, tkeyGlobalParams))
	genAccs, addrs, pubKeys, privKeys := mock.CreateGenAccounts(numGenAccs, sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 5000e8)})

	mock.SetGenesis(mapp, genAccs)
	oracleKeeper := NewKeeper(mapp.Cdc, keyOracle, pk.Subspace("testoracle"), sk, scK, ibcKeeper, bank.NewBaseKeeper(mapp.AccountKeeper), &sdk.Pool{})

	return mapp, ck, oracleKeeper, sk, addrs, pubKeys, privKeys
}

func getInitChainer(mapp *mock.App, stakeKeeper stake.Keeper) sdk.InitChainer {
	return func(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
		mapp.InitChainer(ctx, req)

		stakeGenesis := stake.DefaultGenesisState()
		stakeGenesis.Pool.LooseTokens = sdk.NewDecWithoutFra(100000)

		validators, err := stake.InitGenesis(ctx, stakeKeeper, stakeGenesis)
		if err != nil {
			panic(err)
		}
		return abci.ResponseInitChain{
			Validators: validators,
		}
	}
}

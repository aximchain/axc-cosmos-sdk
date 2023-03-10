package baseapp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	abci "github.com/tendermint/tendermint/abci/types"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	// make some cap keys
	capKey1 = sdk.NewKVStoreKey("key1")
	capKey2 = sdk.NewKVStoreKey("key2")
)

//------------------------------------------------------------------------------------------
// Helpers for setup. Most tests should be able to use setupBaseApp

func defaultLogger() log.Logger {
	return log.NewTMLogger(log.NewSyncWriter(os.Stdout)).With("module", "sdk/app")
}

func newBaseApp(name string, options ...func(*BaseApp)) *BaseApp {
	logger := defaultLogger()
	db := dbm.NewMemDB()
	codec := codec.New()
	registerTestCodec(codec)
	return NewBaseApp(name, logger, db, testTxDecoder(codec), sdk.CollectConfig{}, options...)
}

func registerTestCodec(cdc *codec.Codec) {
	// register Tx, Msg
	sdk.RegisterCodec(cdc)

	// register test types
	cdc.RegisterConcrete(&txTest{}, "cosmos-sdk/baseapp/txTest", nil)
	cdc.RegisterConcrete(&msgCounter{}, "cosmos-sdk/baseapp/msgCounter", nil)
	cdc.RegisterConcrete(&msgCounter2{}, "cosmos-sdk/baseapp/msgCounter2", nil)
	cdc.RegisterConcrete(&msgNoRoute{}, "cosmos-sdk/baseapp/msgNoRoute", nil)
}

// simple one store baseapp
func setupBaseApp(t *testing.T, options ...func(*BaseApp)) *BaseApp {
	app := newBaseApp(t.Name(), options...)
	require.Equal(t, t.Name(), app.Name())

	// no stores are mounted
	require.Panics(t, func() { app.LoadLatestVersion(capKey1) })

	app.MountStoresIAVL(capKey1, capKey2)

	// stores are mounted
	err := app.LoadLatestVersion(capKey1)
	require.Nil(t, err)
	return app
}

type MockBaseApp struct {
	*BaseApp
}

// The following methods just keep same logic with original cosmos code before we fork to make test can pass
func (app *MockBaseApp) LoadLatestVersion(mainKey sdk.StoreKey) error {
	err := app.cms.LoadLatestVersion()
	if err != nil {
		return err
	}
	return app.initFromStore(mainKey)
}

func (app *MockBaseApp) LoadVersion(version int64, mainKey sdk.StoreKey) error {
	err := app.cms.LoadVersion(version)
	if err != nil {
		return err
	}
	return app.initFromStore(mainKey)
}

func (app *MockBaseApp) initFromStore(mainKey sdk.StoreKey) error {
	// main store should exist.
	// TODO: we don't actually need the main store here
	main := app.cms.GetKVStore(mainKey)
	if main == nil {
		return errors.New("baseapp expects MultiStore with 'main' KVStore")
	}

	app.SetCheckState(abci.Header{})

	return nil
}

func NewMockBaseApp(name string, logger log.Logger, db dbm.DB, txDecoder sdk.TxDecoder, collect sdk.CollectConfig, options ...func(*BaseApp)) *MockBaseApp {
	return &MockBaseApp{NewBaseApp(name, logger, db, txDecoder, collect, options...)}
}

//------------------------------------------------------------------------------------------
// test mounting and loading stores

func TestMountStores(t *testing.T) {
	app := setupBaseApp(t)

	// check both stores
	store1 := app.cms.GetCommitKVStore(capKey1)
	require.NotNil(t, store1)
	store2 := app.cms.GetCommitKVStore(capKey2)
	require.NotNil(t, store2)
}

// Test that we can make commits and then reload old versions.
// Test that LoadLatestVersion actually does.
func TestLoadVersion(t *testing.T) {
	logger := defaultLogger()
	db := dbm.NewMemDB()
	name := t.Name()
	app := NewMockBaseApp(name, logger, db, nil, sdk.CollectConfig{})

	// make a cap key and mount the store
	capKey := sdk.NewKVStoreKey("main")
	app.MountStoresIAVL(capKey)
	err := app.LoadLatestVersion(capKey) // needed to make stores non-nil
	require.Nil(t, err)

	emptyCommitID := sdk.CommitID{}

	// fresh store has zero/empty last commit
	lastHeight := app.LastBlockHeight()
	lastID := app.LastCommitID()
	require.Equal(t, int64(0), lastHeight)
	require.Equal(t, emptyCommitID, lastID)

	// execute a block, collect commit ID
	header := abci.Header{Height: 1}
	app.BeginBlock(abci.RequestBeginBlock{Header: header})
	res := app.Commit()
	commitID1 := sdk.CommitID{1, res.Data}

	// execute a block, collect commit ID
	header = abci.Header{Height: 2}
	app.BeginBlock(abci.RequestBeginBlock{Header: header})
	res = app.Commit()
	commitID2 := sdk.CommitID{2, res.Data}

	// reload with LoadLatestVersion
	app = NewMockBaseApp(name, logger, db, nil, sdk.CollectConfig{})
	app.MountStoresIAVL(capKey)
	err = app.LoadLatestVersion(capKey)
	require.Nil(t, err)
	testLoadVersionHelper(t, app, int64(2), commitID2)

	// reload with LoadVersion, see if you can commit the same block and get
	// the same result
	app = NewMockBaseApp(name, logger, db, nil, sdk.CollectConfig{})
	app.MountStoresIAVL(capKey)
	err = app.LoadVersion(1, capKey)
	require.Nil(t, err)
	testLoadVersionHelper(t, app, int64(1), commitID1)
	app.BeginBlock(abci.RequestBeginBlock{Header: header})
	app.Commit()
	testLoadVersionHelper(t, app, int64(2), commitID2)
}

func testLoadVersionHelper(t *testing.T, app *MockBaseApp, expectedHeight int64, expectedID sdk.CommitID) {
	lastHeight := app.LastBlockHeight()
	lastID := app.LastCommitID()
	require.Equal(t, expectedHeight, lastHeight)
	require.Equal(t, expectedID, lastID)
}

func TestOptionFunction(t *testing.T) {
	logger := defaultLogger()
	db := dbm.NewMemDB()
	bap := NewMockBaseApp("starting name", logger, db, nil, sdk.CollectConfig{}, testChangeNameHelper("new name"))
	require.Equal(t, bap.name, "new name", "BaseApp should have had name changed via option function")
}

func testChangeNameHelper(name string) func(*BaseApp) {
	return func(bap *BaseApp) {
		bap.name = name
	}
}

// Test that the app hash is static
// TODO: https://github.com/cosmos/cosmos-sdk/issues/520
/*func TestStaticAppHash(t *testing.T) {
	app := newBaseApp(t.Name())

	// make a cap key and mount the store
	capKey := sdk.NewKVStoreKey("main")
	app.MountStoresIAVL(capKey)
	err := app.LoadLatestVersion(capKey) // needed to make stores non-nil
	require.Nil(t, err)

	// execute some blocks
	header := abci.Header{Height: 1}
	app.BeginBlock(abci.RequestBeginBlock{Header: header})
	res := app.Commit()
	commitID1 := sdk.CommitID{1, res.Data}

	header = abci.Header{Height: 2}
	app.BeginBlock(abci.RequestBeginBlock{Header: header})
	res = app.Commit()
	commitID2 := sdk.CommitID{2, res.Data}

	require.Equal(t, commitID1.Hash, commitID2.Hash)
}
*/

//------------------------------------------------------------------------------------------
// test some basic abci/baseapp functionality

// Test that txs can be unmarshalled and read and that
// correct error codes are returned when not
func TestTxDecoder(t *testing.T) {
	// TODO
}

// Test that Info returns the latest committed state.
func TestInfo(t *testing.T) {
	app := newBaseApp(t.Name())

	// ----- test an empty response -------
	reqInfo := abci.RequestInfo{}
	res := app.Info(reqInfo)

	// should be empty
	assert.Equal(t, "", res.Version)
	assert.Equal(t, t.Name(), res.GetData())
	assert.Equal(t, int64(0), res.LastBlockHeight)
	require.Equal(t, []uint8(nil), res.LastBlockAppHash)

	// ----- test a proper response -------
	// TODO
}

//------------------------------------------------------------------------------------------
// InitChain, BeginBlock, EndBlock

func TestInitChainer(t *testing.T) {
	name := t.Name()
	// keep the db and logger ourselves so
	// we can reload the same  app later
	db := dbm.NewMemDB()
	logger := defaultLogger()
	app := NewMockBaseApp(name, logger, db, nil, sdk.CollectConfig{})
	capKey := sdk.NewKVStoreKey("main")
	capKey2 := sdk.NewKVStoreKey("key2")
	app.MountStoresIAVL(capKey, capKey2)

	// set a value in the store on init chain
	key, value := []byte("hello"), []byte("goodbye")
	var initChainer sdk.InitChainer = func(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
		store := ctx.KVStore(capKey)
		store.Set(key, value)
		return abci.ResponseInitChain{}
	}

	query := abci.RequestQuery{
		Path: "/store/main/key",
		Data: key,
	}

	// initChainer is nil - nothing happens
	app.InitChain(abci.RequestInitChain{})
	res := app.Query(query)
	require.Equal(t, 0, len(res.Value))

	// set initChainer and try again - should see the value
	app.SetInitChainer(initChainer)

	// stores are mounted and private members are set - sealing baseapp
	err := app.LoadLatestVersion(capKey) // needed to make stores non-nil
	require.Nil(t, err)

	app.InitChain(abci.RequestInitChain{AppStateBytes: []byte("{}"), ChainId: "test-chain-id"}) // must have valid JSON genesis file, even if empty

	// assert that chainID is set correctly in InitChain
	chainID := app.DeliverState.Ctx.ChainID()
	require.Equal(t, "test-chain-id", chainID, "ChainID in DeliverState not set correctly in InitChain")

	chainID = app.CheckState.Ctx.ChainID()
	require.Equal(t, "test-chain-id", chainID, "ChainID in CheckState not set correctly in InitChain")

	app.Commit()
	res = app.Query(query)
	require.Equal(t, value, res.Value)

	// reload app
	app = NewMockBaseApp(name, logger, db, nil, sdk.CollectConfig{})
	app.SetInitChainer(initChainer)
	app.MountStoresIAVL(capKey, capKey2)
	err = app.LoadLatestVersion(capKey) // needed to make stores non-nil
	require.Nil(t, err)

	// ensure we can still query after reloading
	res = app.Query(query)
	require.Equal(t, value, res.Value)

	// commit and ensure we can still query
	app.BeginBlock(abci.RequestBeginBlock{})
	app.Commit()
	res = app.Query(query)
	require.Equal(t, value, res.Value)
}

//------------------------------------------------------------------------------------------
// Mock tx, msgs, and mapper for the baseapp tests.
// Self-contained, just uses counters.
// We don't care about signatures, coins, accounts, etc. in the baseapp.

// Simple tx with a list of Msgs.
type txTest struct {
	Msgs    []sdk.Msg
	Counter int64
}

// Implements Tx
func (tx txTest) GetMsgs() []sdk.Msg { return tx.Msgs }

const (
	routeMsgCounter  = "msgCounter"
	routeMsgCounter2 = "msgCounter2"
)

// ValidateBasic() fails on negative counters.
// Otherwise it's up to the handlers
type msgCounter struct {
	Counter int64
}

// Implements Msg
func (msg msgCounter) Route() string                { return routeMsgCounter }
func (msg msgCounter) Type() string                 { return "counter1" }
func (msg msgCounter) GetSignBytes() []byte         { return nil }
func (msg msgCounter) GetSigners() []sdk.AccAddress { return nil }
func (msg msgCounter) ValidateBasic() sdk.Error {
	if msg.Counter >= 0 {
		return nil
	}
	return sdk.ErrInvalidSequence("counter should be a non-negative integer.")
}
func (msg msgCounter) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.GetSigners()
}

func newTxCounter(txInt int64, msgInts ...int64) *txTest {
	var msgs []sdk.Msg
	for _, msgInt := range msgInts {
		msgs = append(msgs, msgCounter{msgInt})
	}
	return &txTest{msgs, txInt}
}

// a msg we dont know how to route
type msgNoRoute struct {
	msgCounter
}

func (tx msgNoRoute) Route() string { return "noroute" }

// a msg we dont know how to decode
type msgNoDecode struct {
	msgCounter
}

func (tx msgNoDecode) Route() string { return routeMsgCounter }

// Another counter msg. Duplicate of msgCounter
type msgCounter2 struct {
	Counter int64
}

// Implements Msg
func (msg msgCounter2) Route() string                { return routeMsgCounter2 }
func (msg msgCounter2) Type() string                 { return "counter2" }
func (msg msgCounter2) GetSignBytes() []byte         { return nil }
func (msg msgCounter2) GetSigners() []sdk.AccAddress { return nil }
func (msg msgCounter2) ValidateBasic() sdk.Error {
	if msg.Counter >= 0 {
		return nil
	}
	return sdk.ErrInvalidSequence("counter should be a non-negative integer.")
}
func (msg msgCounter2) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.GetSigners()
}

// amino decode
func testTxDecoder(cdc *codec.Codec) sdk.TxDecoder {
	return func(txBytes []byte) (sdk.Tx, sdk.Error) {
		var tx txTest
		if len(txBytes) == 0 {
			return nil, sdk.ErrTxDecode("txBytes are empty")
		}
		err := cdc.UnmarshalBinaryLengthPrefixed(txBytes, &tx)
		if err != nil {
			return nil, sdk.ErrTxDecode("").TraceSDK(err.Error())
		}
		return tx, nil
	}
}

func anteHandlerTxTest(t *testing.T, capKey *sdk.KVStoreKey, storeKey []byte) sdk.AnteHandler {
	return func(ctx sdk.Context, tx sdk.Tx, mode sdk.RunTxMode) (newCtx sdk.Context, res sdk.Result, abort bool) {
		store := ctx.KVStore(capKey)
		msgCounter := tx.(txTest).Counter
		res = incrementingCounter(t, store, storeKey, msgCounter)
		return
	}
}

func handlerMsgCounter(t *testing.T, capKey *sdk.KVStoreKey, deliverKey []byte) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		store := ctx.KVStore(capKey)
		var msgCount int64
		switch m := msg.(type) {
		case *msgCounter:
			msgCount = m.Counter
		case *msgCounter2:
			msgCount = m.Counter
		}
		return incrementingCounter(t, store, deliverKey, msgCount)
	}
}

//-----------------------------------------------------------------
// simple int mapper

func i2b(i int64) []byte {
	return []byte{byte(i)}
}

func getIntFromStore(store sdk.KVStore, key []byte) int64 {
	bz := store.Get(key)
	if len(bz) == 0 {
		return 0
	}
	i, err := binary.ReadVarint(bytes.NewBuffer(bz))
	if err != nil {
		panic(err)
	}
	return i
}

func setIntOnStore(store sdk.KVStore, key []byte, i int64) {
	bz := make([]byte, 8)
	n := binary.PutVarint(bz, i)
	store.Set(key, bz[:n])
}

// check counter matches what's in store.
// increment and store
func incrementingCounter(t *testing.T, store sdk.KVStore, counterKey []byte, counter int64) (res sdk.Result) {
	storedCounter := getIntFromStore(store, counterKey)
	require.Equal(t, storedCounter, counter)
	setIntOnStore(store, counterKey, counter+1)
	return
}

//---------------------------------------------------------------------
// Tx processing - CheckTx, DeliverTx, SimulateTx.
// These tests use the serialized tx as input, while most others will use the
// Check(), Deliver(), Simulate() methods directly.
// Ensure that Check/Deliver/Simulate work as expected with the store.

// Test that successive CheckTx can see each others' effects
// on the store within a block, and that the CheckTx state
// gets reset to the latest committed state during Commit
func TestCheckTx(t *testing.T) {
	// This ante handler reads the key and checks that the value matches the current counter.
	// This ensures changes to the kvstore persist across successive CheckTx.
	counterKey := []byte("counter-key")

	anteOpt := func(bapp *BaseApp) { bapp.SetAnteHandler(anteHandlerTxTest(t, capKey1, counterKey)) }
	routerOpt := func(bapp *BaseApp) {
		// TODO: can remove this once CheckTx doesnt process msgs.
		bapp.Router().AddRoute(routeMsgCounter, func(ctx sdk.Context, msg sdk.Msg) sdk.Result { return sdk.Result{} })
	}

	app := setupBaseApp(t, anteOpt, routerOpt)

	nTxs := int64(5)

	app.InitChain(abci.RequestInitChain{})

	// Create same codec used in TxDecoder
	codec := codec.New()
	registerTestCodec(codec)

	for i := int64(0); i < nTxs; i++ {
		tx := newTxCounter(i, 0)
		txBytes, err := codec.MarshalBinaryLengthPrefixed(tx)
		require.NoError(t, err)
		r := app.CheckTx(abci.RequestCheckTx{Tx: txBytes})
		assert.True(t, r.IsOK(), fmt.Sprintf("%v", r))
	}

	checkStateStore := app.CheckState.Ctx.KVStore(capKey1)
	storedCounter := getIntFromStore(checkStateStore, counterKey)

	// Ensure AnteHandler ran
	require.Equal(t, nTxs, storedCounter)

	// If a block is committed, CheckTx state should be reset.
	app.BeginBlock(abci.RequestBeginBlock{})
	app.EndBlock(abci.RequestEndBlock{})
	app.Commit()

	checkStateStore = app.CheckState.Ctx.KVStore(capKey1)
	storedBytes := checkStateStore.Get(counterKey)
	require.Nil(t, storedBytes)
}

// Test that successive DeliverTx can see each others' effects
// on the store, both within and across blocks.
func TestDeliverTx(t *testing.T) {
	// test increments in the ante
	anteKey := []byte("ante-key")
	anteOpt := func(bapp *BaseApp) { bapp.SetAnteHandler(anteHandlerTxTest(t, capKey1, anteKey)) }

	// test increments in the handler
	deliverKey := []byte("deliver-key")
	routerOpt := func(bapp *BaseApp) {
		bapp.Router().AddRoute(routeMsgCounter, handlerMsgCounter(t, capKey1, deliverKey))
	}

	app := setupBaseApp(t, anteOpt, routerOpt)

	// Create same codec used in TxDecoder
	codec := codec.New()
	registerTestCodec(codec)

	nBlocks := 3
	txPerHeight := 5
	for blockN := 0; blockN < nBlocks; blockN++ {
		app.BeginBlock(abci.RequestBeginBlock{})
		for i := 0; i < txPerHeight; i++ {
			counter := int64(blockN*txPerHeight + i)
			tx := newTxCounter(counter, counter)
			txBytes, err := codec.MarshalBinaryLengthPrefixed(tx)
			require.NoError(t, err)
			res := app.DeliverTx(abci.RequestDeliverTx{Tx: txBytes})
			require.True(t, res.IsOK(), fmt.Sprintf("%v", res))
		}
		app.EndBlock(abci.RequestEndBlock{})
		app.Commit()
	}
}

// Number of messages doesn't matter to CheckTx.
func TestMultiMsgCheckTx(t *testing.T) {
	// TODO: ensure we get the same results
	// with one message or many
}

// One call to DeliverTx should process all the messages, in order.
func TestMultiMsgDeliverTx(t *testing.T) {
	// increment the tx counter
	anteKey := []byte("ante-key")
	anteOpt := func(bapp *BaseApp) { bapp.SetAnteHandler(anteHandlerTxTest(t, capKey1, anteKey)) }

	// increment the msg counter
	deliverKey := []byte("deliver-key")
	deliverKey2 := []byte("deliver-key2")
	routerOpt := func(bapp *BaseApp) {
		bapp.Router().AddRoute(routeMsgCounter, handlerMsgCounter(t, capKey1, deliverKey))
		bapp.Router().AddRoute(routeMsgCounter2, handlerMsgCounter(t, capKey1, deliverKey2))
	}

	app := setupBaseApp(t, anteOpt, routerOpt)

	// Create same codec used in TxDecoder
	codec := codec.New()
	registerTestCodec(codec)

	// run a multi-msg tx
	// with all msgs the same route
	{
		app.BeginBlock(abci.RequestBeginBlock{})
		tx := newTxCounter(0, 0, 1, 2)
		txBytes, err := codec.MarshalBinaryLengthPrefixed(tx)
		require.NoError(t, err)
		res := app.DeliverTx(abci.RequestDeliverTx{Tx:txBytes})
		require.True(t, res.IsErr(), fmt.Sprintf("%v", res))
	}
}

// Interleave calls to Check and Deliver and ensure
// that there is no cross-talk. Check sees results of the previous Check calls
// and Deliver sees that of the previous Deliver calls, but they don't see eachother.
func TestConcurrentCheckDeliver(t *testing.T) {
	// TODO
}

func TestPreCheckTx(t *testing.T) {
	counterKey := []byte("counter-key")

	anteOpt := func(bapp *BaseApp) { bapp.SetAnteHandler(anteHandlerTxTest(t, capKey1, counterKey)) }
	routerOpt := func(bapp *BaseApp) {
		// TODO: can remove this once CheckTx doesnt process msgs.
		bapp.Router().AddRoute(routeMsgCounter, func(ctx sdk.Context, msg sdk.Msg) sdk.Result { return sdk.Result{} })
	}

	app := setupBaseApp(t, anteOpt, routerOpt)
	app.SetPreChecker(func(ctx sdk.Context, txBytes []byte, tx sdk.Tx) sdk.Result {
		return sdk.ErrInternal("Must Fail").Result()
	})
	nTxs := int64(5)

	app.InitChain(abci.RequestInitChain{})

	// Create same codec used in TxDecoder
	codec := codec.New()
	registerTestCodec(codec)

	for i := int64(0); i < nTxs; i++ {
		tx := newTxCounter(i, 0)
		txBytes, err := codec.MarshalBinaryLengthPrefixed(tx)
		require.NoError(t, err)
		r := app.PreCheckTx(abci.RequestCheckTx{Tx: txBytes})
		assert.False(t, r.IsOK(), fmt.Sprintf("%v", r))
	}

	checkStateStore := app.CheckState.Ctx.KVStore(capKey1)
	storedCounter := getIntFromStore(checkStateStore, counterKey)

	// Ensure AnteHandler ran
	require.NotEqual(t, nTxs, storedCounter)
	assert.Equal(t, 0, app.txMsgCache.Len())

	app.SetPreChecker(func(ctx sdk.Context, txBytes []byte, tx sdk.Tx) sdk.Result {
		return sdk.Result{}
	})

	tx := newTxCounter(0, 0)
	txBytes, _ := codec.MarshalBinaryLengthPrefixed(tx)
	r := app.PreCheckTx(abci.RequestCheckTx{Tx: txBytes})
	assert.True(t, r.IsOK(), fmt.Sprintf("%v", r))
	assert.Equal(t, 1, app.txMsgCache.Len())
}

// Simulate() and Query("/app/simulate", txBytes) should give
// the same results.
func TestSimulateTx(t *testing.T) {

	anteOpt := func(bapp *BaseApp) {
		bapp.SetAnteHandler(func(ctx sdk.Context, tx sdk.Tx, mode sdk.RunTxMode) (newCtx sdk.Context, res sdk.Result, abort bool) {
			newCtx = ctx
			return
		})
	}

	routerOpt := func(bapp *BaseApp) {
		bapp.Router().AddRoute(routeMsgCounter, func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
			return sdk.Result{}
		})
	}

	app := setupBaseApp(t, anteOpt, routerOpt)

	app.InitChain(abci.RequestInitChain{})

	// Create same codec used in TxDecoder
	cdc := codec.New()
	registerTestCodec(cdc)

	nBlocks := 3
	for blockN := 0; blockN < nBlocks; blockN++ {
		count := int64(blockN + 1)
		app.BeginBlock(abci.RequestBeginBlock{})

		tx := newTxCounter(count, count)

		// simulate a message
		result := app.Simulate(nil, tx)
		require.True(t, result.IsOK(), result.Log)

		// simulate again, same result
		result = app.Simulate(nil, tx)
		require.True(t, result.IsOK(), result.Log)

		// simulate by calling Query with encoded tx
		txBytes, err := cdc.MarshalBinaryLengthPrefixed(tx)
		require.Nil(t, err)
		query := abci.RequestQuery{
			Path: "/app/simulate",
			Data: txBytes,
		}
		queryResult := app.Query(query)
		require.True(t, queryResult.IsOK(), queryResult.Log)

		var res sdk.Result
		codec.Cdc.MustUnmarshalBinaryLengthPrefixed(queryResult.Value, &res)
		require.Nil(t, err, "Result unmarshalling failed")
		require.True(t, res.IsOK(), res.Log)
		app.EndBlock(abci.RequestEndBlock{})
		app.Commit()
	}
}

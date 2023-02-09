package auth

import (
	"github.com/aximchain/axc-cosmos-sdk/codec"
	"github.com/aximchain/axc-cosmos-sdk/types"
)

// Register concrete types on codec codec for default AppAccount
func RegisterCodec(cdc *codec.Codec) {
	cdc.RegisterInterface((*types.Account)(nil), nil)
	cdc.RegisterConcrete(&BaseAccount{}, "auth/Account", nil)
	cdc.RegisterConcrete(StdTx{}, "auth/StdTx", nil)
}

var msgCdc = codec.New()

func init() {
	RegisterCodec(msgCdc)
	codec.RegisterCrypto(msgCdc)
}

package tx

import (
	"fmt"

	modulev1 "cosmossdk.io/api/cosmos/tx/module/v1"
	"cosmossdk.io/core/appmodule"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/depinject"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/cosmos/cosmos-sdk/x/auth/posthandler"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	feegrantkeeper "github.com/cosmos/cosmos-sdk/x/feegrant/keeper"
)

func init() {
	appmodule.Register(&modulev1.Module{},
		appmodule.Provide(provideModule),
	)
}

type txInputs struct {
	depinject.In

	Config              *modulev1.Module
	ProtoCodecMarshaler codec.ProtoCodecMarshaler
	TxConfig            *client.TxConfig

	AccountKeeper  authkeeper.AccountKeeper `key:"cosmos.auth.v1.AccountKeeper" optional:"true"`
	BankKeeper     bankkeeper.Keeper        `key:"cosmos.bank.v1.Keeper" optional:"true"`
	FeeGrantKeeper *feegrantkeeper.Keeper   `key:"cosmos.feegrant.v1.Keeper" optional:"true"`
}

type txOutputs struct {
	depinject.Out

	// TxConfig      *client.TxConfig
	BaseAppOption func(*baseapp.BaseApp)
}

func provideModule(in txInputs) txOutputs {
	// var txConfig client.TxConfig
	// if in.TxConfig == nil {
	// txConfig = tx.NewTxConfig(in.ProtoCodecMarshaler, tx.DefaultSignModes)
	// } else {
	// txConfig = in.TxConfig
	// }

	baseAppOption := func(app *baseapp.BaseApp) {

		if !in.Config.SkipAnteHandler {
			// AnteHandlers
			anteHandler, err := newAnteHandler(in.TxConfig, in)
			if err != nil {
				panic(err)
			}
			app.SetAnteHandler(anteHandler)
		}

		if !in.Config.SkipPostHandler {
			// PostHandlers
			// In v0.46, the SDK introduces _postHandlers_. PostHandlers are like
			// antehandlers, but are run _after_ the `runMsgs` execution. They are also
			// defined as a chain, and have the same signature as antehandlers.
			//
			// In baseapp, postHandlers are run in the same store branch as `runMsgs`,
			// meaning that both `runMsgs` and `postHandler` state will be committed if
			// both are successful, and both will be reverted if any of the two fails.
			//
			// The SDK exposes a default empty postHandlers chain.
			//
			// Please note that changing any of the anteHandler or postHandler chain is
			// likely to be a state-machine breaking change, which needs a coordinated
			// upgrade.
			postHandler, err := posthandler.NewPostHandler(
				posthandler.HandlerOptions{},
			)
			if err != nil {
				panic(err)
			}
			app.SetPostHandler(postHandler)
		}

		// TxDecoder
		app.SetTxDecoder((*in.TxConfig).TxDecoder())
	}

	return txOutputs{BaseAppOption: baseAppOption}
}

func newAnteHandler(txConfig *client.TxConfig, in txInputs) (sdk.AnteHandler, error) {
	if in.BankKeeper == nil {
		return nil, fmt.Errorf("both AccountKeeper and BankKeeper are required")
	}

	anteHandler, err := ante.NewAnteHandler(
		ante.HandlerOptions{
			AccountKeeper:   in.AccountKeeper,
			BankKeeper:      in.BankKeeper,
			SignModeHandler: (*txConfig).SignModeHandler(),
			FeegrantKeeper:  in.FeeGrantKeeper,
			SigGasConsumer:  ante.DefaultSigVerificationGasConsumer,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ante handler: %w", err)
	}

	return anteHandler, nil
}
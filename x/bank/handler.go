package bank

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/certikfoundation/shentu/x/auth/vesting"
	"github.com/certikfoundation/shentu/x/bank/types"
)

// NewHandler returns a handler for "auth" type messages.
func NewHandler(k Keeper, ak types.AccountKeeper) sdk.Handler {
	cosmosHandler := bank.NewHandler(k)
	return func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		switch msg := msg.(type) {
		case types.MsgLockedSend:
			return handleMsgLockedSend(ctx, k, ak, msg)
		default:
			return cosmosHandler(ctx, msg)
		}
	}
}

func handleMsgLockedSend(ctx sdk.Context, k Keeper, ak types.AccountKeeper, msg types.MsgLockedSend) (*sdk.Result, error) {
	// preliminary checks
	from := ak.GetAccount(ctx, msg.From)
	if from == nil {
		return nil, sdkerrors.Wrapf(sdkerrors.ErrUnknownAddress, "sender account %s does not exist", msg.From)
	}
	if msg.To.Equals(msg.Unlocker) {
		return nil, sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "recipient cannot be the unlocker")
	}

	acc := ak.GetAccount(ctx, msg.To)

	var toAcc *vesting.ManualVestingAccount
	if acc == nil {
		acc = ak.NewAccountWithAddress(ctx, msg.To)
		baseAcc := auth.NewBaseAccount(msg.To, sdk.NewCoins(), acc.GetPubKey(), acc.GetAccountNumber(), acc.GetSequence())
		if msg.Unlocker.Empty() {
			return nil, sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid unlocker address provided")
		}
		toAcc = vesting.NewManualVestingAccount(baseAcc, sdk.NewCoins(), msg.Unlocker)
	} else {
		var ok bool
		toAcc, ok = acc.(*vesting.ManualVestingAccount)
		if !ok {
			return nil, sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "receiver account is not a ManualVestingAccount")
		}
		if !msg.Unlocker.Empty() {
			return nil, sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "cannot change the unlocker for existing ManualVestingAccount")
		}
	}

	// add to receiver account as normally done
	// but make the added amount vesting (OV := Vesting + Vested)
	toAcc.OriginalVesting = toAcc.OriginalVesting.Add(msg.Amount...)
	newCoins := toAcc.Coins.Add(msg.Amount...)
	if newCoins.IsAnyNegative() {
		return nil, sdkerrors.Wrapf(
			sdkerrors.ErrInsufficientFunds, "insufficient account funds; %s < %s", toAcc.Coins, msg.Amount,
		)
	}
	toAcc.Coins = newCoins
	ak.SetAccount(ctx, toAcc)

	// subtract from sender account (as normally done)
	_, err := k.SubtractCoins(ctx, msg.From, msg.Amount)
	if err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeLockedSendToVestingAccount,
			sdk.NewAttribute(bank.AttributeKeySender, msg.From.String()),
			sdk.NewAttribute(bank.AttributeKeyRecipient, msg.To.String()),
			sdk.NewAttribute(types.AttributeKeyUnlocker, msg.Unlocker.String()),
			sdk.NewAttribute(sdk.AttributeKeyAmount, msg.Amount.String()),
		),
	)
	return &sdk.Result{Events: ctx.EventManager().Events()}, nil
}

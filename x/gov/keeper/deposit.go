package keeper

import (
	"encoding/hex"
	"fmt"

	"github.com/tendermint/tendermint/crypto/tmhash"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govTypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/certikfoundation/shentu/x/gov/types"
	"github.com/certikfoundation/shentu/x/shield"
)

// GetDeposit gets the deposit of a specific depositor on a specific proposal.
func (k Keeper) GetDeposit(ctx sdk.Context, proposalID uint64, depositorAddr sdk.AccAddress) (deposit types.Deposit, found bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(govTypes.DepositKey(proposalID, depositorAddr))
	if bz == nil {
		return deposit, false
	}

	k.cdc.MustUnmarshalBinaryLengthPrefixed(bz, &deposit)
	return deposit, true
}

// SetDeposit sets the deposit to KVStore.
func (k Keeper) SetDeposit(ctx sdk.Context, deposit types.Deposit) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshalBinaryLengthPrefixed(deposit)
	store.Set(govTypes.DepositKey(deposit.ProposalID, deposit.Depositor), bz)
}

// AddDeposit adds or updates a deposit of a specific depositor on a specific proposal.
// When proposer is a council member, it's not depositable.
// Activates voting period when appropriate.
func (k Keeper) AddDeposit(ctx sdk.Context, proposalID uint64, depositorAddr sdk.AccAddress, depositAmount sdk.Coins) (bool, error) {
	// checks to see if proposal exists
	proposal, ok := k.GetProposal(ctx, proposalID)
	if !ok {
		return false, sdkerrors.Wrap(govTypes.ErrUnknownProposal, fmt.Sprint(proposalID))
	}
	// check if proposal is still depositable or if proposer is council member
	if (proposal.Status != types.StatusDepositPeriod) || proposal.IsProposerCouncilMember {
		return false, sdkerrors.Wrap(govTypes.ErrAlreadyActiveProposal, fmt.Sprint(proposalID))
	}

	// update the governance module's account coins pool
	err := k.supplyKeeper.SendCoinsFromAccountToModule(ctx, depositorAddr, govTypes.ModuleName, depositAmount)
	if err != nil {
		return false, err
	}
	// update proposal
	proposal.TotalDeposit = proposal.TotalDeposit.Add(depositAmount...)
	k.SetProposal(ctx, proposal)

	// check if deposit has provided sufficient total funds to transition the proposal into the voting period
	activatedVotingPeriod := false
	if proposal.Status == types.StatusDepositPeriod && proposal.TotalDeposit.IsAllGTE(k.GetDepositParams(ctx).MinDeposit) ||
		proposal.ProposalType() == shield.ProposalTypeShieldClaim {
		k.ActivateVotingPeriod(ctx, proposal)
		activatedVotingPeriod = true
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			govTypes.EventTypeProposalDeposit,
			sdk.NewAttribute(sdk.AttributeKeyAmount, depositAmount.String()),
			sdk.NewAttribute(govTypes.AttributeKeyProposalID, fmt.Sprintf("%d", proposalID)),
			sdk.NewAttribute(types.AttributeKeyDepositor, depositorAddr.String()),
		),
	)

	k.upsertDeposit(ctx, proposalID, depositorAddr, depositAmount)

	return activatedVotingPeriod, nil
}

// upsertDeposit updates or inserts a deposit to a proposal.
func (k Keeper) upsertDeposit(ctx sdk.Context, proposalID uint64, depositorAddr sdk.AccAddress, depositAmount sdk.Coins) {
	// add or update deposit object
	deposit, found := k.GetDeposit(ctx, proposalID, depositorAddr)
	if found {
		deposit.Amount = deposit.Amount.Add(depositAmount...)
	} else {
		deposit = types.NewDeposit(proposalID, depositorAddr, depositAmount, hex.EncodeToString(tmhash.Sum(ctx.TxBytes())))
	}

	k.SetDeposit(ctx, deposit)
}

// GetAllDeposits returns all the deposits from the store.
func (k Keeper) GetAllDeposits(ctx sdk.Context) (deposits types.Deposits) {
	k.IterateAllDeposits(ctx, func(deposit types.Deposit) bool {
		deposits = append(deposits, deposit)
		return false
	})
	return
}

// GetDepositsByProposalID returns all the deposits from a proposal.
func (k Keeper) GetDepositsByProposalID(ctx sdk.Context, proposalID uint64) (deposits types.Deposits) {
	k.IterateDeposits(ctx, proposalID, func(deposit types.Deposit) bool {
		deposits = append(deposits, deposit)
		return false
	})
	return
}

// GetDepositsIteratorByProposalID gets all the deposits on a specific proposal as an sdk.Iterator.
func (k Keeper) GetDepositsIteratorByProposalID(ctx sdk.Context, proposalID uint64) sdk.Iterator {
	store := ctx.KVStore(k.storeKey)
	return sdk.KVStorePrefixIterator(store, govTypes.DepositsKey(proposalID))
}

// RefundDepositsByProposalID refunds and deletes all the deposits on a specific proposal.
func (k Keeper) RefundDepositsByProposalID(ctx sdk.Context, proposalID uint64) {
	store := ctx.KVStore(k.storeKey)

	k.IterateDeposits(ctx, proposalID, func(deposit types.Deposit) bool {
		err := k.supplyKeeper.SendCoinsFromModuleToAccount(ctx, govTypes.ModuleName, deposit.Depositor, deposit.Amount)
		if err != nil {
			panic(err)
		}

		store.Delete(govTypes.DepositKey(proposalID, deposit.Depositor))
		return false
	})
}

// DeleteDepositsByProposalID deletes all the deposits on a specific proposal without refunding them.
func (k Keeper) DeleteDepositsByProposalID(ctx sdk.Context, proposalID uint64) {
	store := ctx.KVStore(k.storeKey)

	k.IterateDeposits(ctx, proposalID, func(deposit types.Deposit) bool {
		err := k.supplyKeeper.BurnCoins(ctx, govTypes.ModuleName, deposit.Amount)
		if err != nil {
			panic(err)
		}

		store.Delete(govTypes.DepositKey(proposalID, deposit.Depositor))
		return false
	})
}

// GetDeposits returns all the deposits from a proposal.
func (k Keeper) GetDeposits(ctx sdk.Context, proposalID uint64) (deposits types.Deposits) {
	k.IterateDeposits(ctx, proposalID, func(deposit types.Deposit) bool {
		deposits = append(deposits, deposit)
		return false
	})
	return
}

// IterateDeposits iterates over the all the proposals deposits and performs a callback function.
func (k Keeper) IterateDeposits(ctx sdk.Context, proposalID uint64, cb func(deposit types.Deposit) (stop bool)) {
	iterator := k.GetDepositsIteratorByProposalID(ctx, proposalID)

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var deposit types.Deposit
		k.cdc.MustUnmarshalBinaryLengthPrefixed(iterator.Value(), &deposit)

		if cb(deposit) {
			break
		}
	}
}

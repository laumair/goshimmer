package mana

import (
	"github.com/iotaledger/goshimmer/packages/datastructure"
)

type Balance struct {
	calculator      *Calculator
	transferHistory *datastructure.DoublyLinkedList
}

func NewBalance(calculator *Calculator) *Balance {
	return &Balance{
		calculator:      calculator,
		transferHistory: &datastructure.DoublyLinkedList{},
	}
}

// Returns the current mana balance.
func (balance *Balance) GetValue() uint64 {
	if lastBalanceHistoryEntry, err := balance.transferHistory.GetLast(); datastructure.ErrNoSuchElement.Equals(err) {
		return 0
	} else {
		return lastBalanceHistoryEntry.(*BalanceHistoryEntry).balance
	}
}

// Returns the timestamp of the last mana erosion.
func (balance *Balance) GetLastErosion() uint64 {
	if lastBalanceHistoryEntry, err := balance.transferHistory.GetLast(); datastructure.ErrNoSuchElement.Equals(err) {
		return 0
	} else {
		return lastBalanceHistoryEntry.(*BalanceHistoryEntry).transfer.spentTime
	}
}

// Books a new transfer to the balance.
func (balance *Balance) AddTransfer(transfer *Transfer) {
	// check if we need to rollback transfers (to prevent rounding errors)
	rolledBackTransactions := balance.rollbackTransfers(transfer.spentTime)

	// apply new transfer
	balance.applyTransfer(transfer)

	// replay rolled back transfers (in reverse order)
	for i := len(rolledBackTransactions) - 1; i >= 0; i-- {
		balance.applyTransfer(rolledBackTransactions[i])
	}
}

// Rolls back transfers that have their spentTime after the given referenceTime and returns a slice containing the
// rolled back transfers.
//
// Since the mana calculations use floats, we will see rounding errors. To allow all nodes to have consensus on the
// current mana balance, we need to make nodes use the exact same formulas and apply them in the exact same order.
// Because of the asynchronous nature of the tangle, nodes will see different transactions at different times and will
// therefore process their mana gains in a different order. This could lead to discrepancies in the balance due to
// accumulated rounding errors. To work around this problem, we keep a history of the latest transfers (up till a
// certain age), that can be rolled back. This allows us to apply all mana changes in the exact same order which will
// lead to a network wide consensus on the mana balances.
func (balance *Balance) rollbackTransfers(referenceTime uint64) (result []*Transfer) {
	result = make([]*Transfer, 0)

	for {
		if lastListEntry, err := balance.transferHistory.GetLast(); err != nil {
			if !datastructure.ErrNoSuchElement.Equals(err) {
				panic(err)
			}

			return
		} else if lastTransfer := lastListEntry.(*BalanceHistoryEntry).transfer; lastTransfer.spentTime < referenceTime {
			return
		} else {
			result = append(result, lastTransfer)

			if _, err := balance.transferHistory.RemoveLast(); err != nil {
				panic(err)
			}
		}
	}
}

// Applies the balance changes of the given transfer.
func (balance *Balance) applyTransfer(transfer *Transfer) {
	// retrieve current values
	var currentBalance, lastErosion uint64
	if lastListEntry, err := balance.transferHistory.GetLastEntry(); err != nil {
		if !datastructure.ErrNoSuchElement.Equals(err) {
			panic(err)
		}

		currentBalance = 0
		lastErosion = 0
	} else {
		lastBalanceHistoryEntry := lastListEntry.GetValue().(*BalanceHistoryEntry)

		currentBalance = lastBalanceHistoryEntry.balance
		lastErosion = lastBalanceHistoryEntry.transfer.spentTime
	}

	// erode if we have a balance
	if currentBalance != 0 {
		currentBalance, _ = balance.calculator.ErodeMana(currentBalance, transfer.spentTime-lastErosion)
	}

	// calculate mana gains
	gainedMana, roundingError := balance.calculator.GenerateMana(transfer.movedCoins, transfer.spentTime-transfer.receivedTime)

	// store results
	balance.transferHistory.AddLast(&BalanceHistoryEntry{
		transfer:                 transfer,
		balance:                  currentBalance + gainedMana,
		accumulatedRoundingError: roundingError,
	})
}
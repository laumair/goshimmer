package wallet

import (
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/address"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/balance"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/transaction"
)

// Output is a wallet specific representation of an output in the IOTA network.
type Output struct {
	address        address.Address
	transactionID  transaction.ID
	balances       map[balance.Color]uint64
	inclusionState InclusionState
}

// InclusionState is a container for the different flags of an output that define if it was accepted in the network.
type InclusionState struct {
	Liked       bool
	Confirmed   bool
	Rejected    bool
	Conflicting bool
	Spent       bool
}

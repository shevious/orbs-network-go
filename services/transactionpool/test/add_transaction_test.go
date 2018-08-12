package test

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/services/transactionpool"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestForwardsANewValidTransactionUsingGossip(t *testing.T) {
	h := newHarness()

	tx := builders.TransferTransaction().Build()
	h.expectTransactionToBeForwarded(tx)

	_, err := h.addNewTransaction(tx)

	require.NoError(t, err, "a valid transaction was not added to pool")
	require.NoError(t, h.verifyMocks(), "mock gossip was not called as expected")
}

func TestDoesNotForwardInvalidTransactionsUsingGossip(t *testing.T) {
	h := newHarness()

	tx := builders.TransferTransaction().WithInvalidContent().Build()
	h.expectNoTransactionsToBeForwarded()

	_, err := h.addNewTransaction(tx)

	require.Error(t, err, "an invalid transaction was added to the pool")
	require.NoError(t, h.verifyMocks(), "mock gossip was not called (as expected)")
}

func TestDoesNotAddTransactionsThatFailedPreOrderChecks(t *testing.T) {
	h := newHarness()
	tx := builders.TransferTransaction().Build()
	expectedStatus := protocol.TRANSACTION_STATUS_REJECTED_SMART_CONTRACT_PRE_ORDER

	h.failPreOrderCheckFor(tx, expectedStatus)
	h.ignoringForwardMessages()

	out, err := h.addNewTransaction(tx)
	//TODO assert block height and timestamp from empty receipt as per spec

	require.NotNil(t, out, "output must not be nil even on errors")

	require.Error(t, err, "an transaction that failed pre-order checks was added to the pool")
	require.IsType(t, &transactionpool.ErrTransactionRejected{}, err, "error was not of the expected type")

	typedError := err.(*transactionpool.ErrTransactionRejected)
	require.Equal(t, expectedStatus, typedError.TransactionStatus, "error did not contain expected transaction status")

	require.NoError(t, h.verifyMocks(), "mocks were not called as expected")

}

func TestDoesNotAddTheSameTransactionTwice(t *testing.T) {
	h := newHarness()

	tx := builders.TransferTransaction().Build()
	h.ignoringForwardMessages()

	h.addNewTransaction(tx)
	_, err := h.addNewTransaction(tx)
	require.Error(t, err, "a transaction was added twice to the pool")
}

func TestReturnsReceiptForTransactionThatHasAlreadyBeenCommitted(t *testing.T) {
	h := newHarness()

	tx := builders.TransferTransaction().Build()
	h.ignoringForwardMessages()

	h.addNewTransaction(tx)
	h.reportTransactionAsCommitted(tx)

	receipt, err := h.addNewTransaction(tx)

	require.NoError(t, err, "a committed transaction that was added again was wrongly rejected")
	require.Equal(t, protocol.TRANSACTION_STATUS_DUPLCIATE_TRANSACTION_ALREADY_COMMITTED, receipt.TransactionStatus, "expected transaction status to be committed")
	require.Equal(t, digest.CalcTxHash(tx.Transaction()), receipt.TransactionReceipt.Txhash(), "expected transaction receipt to contain transaction hash")
}

func TestDoesNotAddTransactionIfPoolIsFull(t *testing.T) {
	h := newHarnessWithSizeLimit(1)

	h.expectNoTransactionsToBeForwarded()

	tx := builders.TransferTransaction().Build()
	_, err := h.addNewTransaction(tx)

	require.Error(t, err, "a transaction was added to a full pool")
	require.NoError(t, h.verifyMocks(), "mock gossip was not called (as expected)")
}

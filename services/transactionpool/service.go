package transactionpool

import (
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

type Config interface {
	PendingPoolSizeInBytes() uint32
	NodePublicKey() primitives.Ed25519PublicKey
}

type service struct {
	pendingTransactions chan *protocol.SignedTransaction
	gossip              gossiptopics.TransactionRelay
	virtualMachine      services.VirtualMachine
	reporting           instrumentation.BasicLogger
	config              Config

	lastCommittedBlockHeight primitives.BlockHeight
	pendingPool              *pendingTxPool
	committedPool            *committedTxPool
}

func NewTransactionPool(gossip gossiptopics.TransactionRelay, virtualMachine services.VirtualMachine, config Config, reporting instrumentation.BasicLogger) services.TransactionPool {
	s := &service{
		pendingTransactions: make(chan *protocol.SignedTransaction, 10),
		gossip:              gossip,
		virtualMachine:      virtualMachine,
		config:              config,
		reporting:           reporting.For(instrumentation.Service("transaction-pool")),

		pendingPool:   NewPendingPool(config),
		committedPool: NewCommittedPool(),
	}
	gossip.RegisterTransactionRelayHandler(s)
	return s
}

func (s *service) GetTransactionsForOrdering(input *services.GetTransactionsForOrderingInput) (*services.GetTransactionsForOrderingOutput, error) {
	out := &services.GetTransactionsForOrderingOutput{}
	out.SignedTransactions = make([]*protocol.SignedTransaction, input.MaxNumberOfTransactions)
	for i := uint32(0); i < input.MaxNumberOfTransactions; i++ {
		out.SignedTransactions[i] = <-s.pendingTransactions
	}
	return out, nil
}

func (s *service) GetCommittedTransactionReceipt(input *services.GetCommittedTransactionReceiptInput) (*services.GetCommittedTransactionReceiptOutput, error) {
	panic("Not implemented")
}

func (s *service) ValidateTransactionsForOrdering(input *services.ValidateTransactionsForOrderingInput) (*services.ValidateTransactionsForOrderingOutput, error) {
	panic("Not implemented")
}

func (s *service) CommitTransactionReceipts(input *services.CommitTransactionReceiptsInput) (*services.CommitTransactionReceiptsOutput, error) {
	if input.LastCommittedBlockHeight != s.lastCommittedBlockHeight+1 {
		return &services.CommitTransactionReceiptsOutput{
			NextDesiredBlockHeight:   s.lastCommittedBlockHeight + 1,
			LastCommittedBlockHeight: s.lastCommittedBlockHeight,
		}, nil
	}

	for _, receipt := range input.TransactionReceipts {
		s.committedPool.add(receipt)
		s.pendingPool.remove(receipt.Txhash())
	}

	s.lastCommittedBlockHeight = input.LastCommittedBlockHeight

	return &services.CommitTransactionReceiptsOutput{
		NextDesiredBlockHeight:   s.lastCommittedBlockHeight + 1,
		LastCommittedBlockHeight: s.lastCommittedBlockHeight,
	}, nil
}

func (s *service) RegisterTransactionResultsHandler(handler handlers.TransactionResultsHandler) {
	panic("Not implemented")
}

func (s *service) HandleForwardedTransactions(input *gossiptopics.ForwardedTransactionsInput) (*gossiptopics.EmptyOutput, error) {
	for _, tx := range input.Message.SignedTransactions {
		s.pendingTransactions <- tx
	}
	return nil, nil
}

func (s *service) isTransactionInPendingPool(transaction *protocol.SignedTransaction) bool {
	return s.pendingPool.has(transaction)
}

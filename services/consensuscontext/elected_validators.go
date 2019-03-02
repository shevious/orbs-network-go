package consensuscontext

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Elections"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
)

func (s *service) getElectedValidators(ctx context.Context, currentBlockHeight primitives.BlockHeight) ([]primitives.NodeAddress, error) {
	lastCommittedBlockHeight := currentBlockHeight - 1

	federationNodes := s.config.FederationNodes(uint64(lastCommittedBlockHeight))
	federationNodesAddresses := toNodeAddresses(federationNodes)

	// genesis
	if lastCommittedBlockHeight == 0 {
		return federationNodesAddresses, nil
	}

	electedValidatorsAddresses, err := s.callElectionsSystemContract(ctx, lastCommittedBlockHeight)
	if err != nil {
		if ctx.Err() == nil { // this may fail rightfully on graceful shutdown (ctx.Done), we don't want to report an error in this case
			s.logger.Error("cannot get elected validators from system contract", log.Error(err), log.BlockHeight(lastCommittedBlockHeight))
		}
		return nil, err
	}
	s.logger.Info("queried elected validators", log.Int("num-results", len(electedValidatorsAddresses)), log.BlockHeight(lastCommittedBlockHeight))

	// elections not active yet
	if len(electedValidatorsAddresses) == 0 {
		return federationNodesAddresses, nil
	}

	return electedValidatorsAddresses, nil
}

func (s *service) callElectionsSystemContract(ctx context.Context, blockHeight primitives.BlockHeight) ([]primitives.NodeAddress, error) {
	systemContractName := primitives.ContractName(elections_systemcontract.CONTRACT_NAME)
	systemMethodName := primitives.MethodName(elections_systemcontract.METHOD_GET_ELECTED_VALIDATORS)

	output, err := s.virtualMachine.CallSystemContract(ctx, &services.CallSystemContractInput{
		BlockHeight:        blockHeight,
		BlockTimestamp:     0, // unfortunately we don't know the timestamp here, this limits which contract SDK API can be used
		ContractName:       systemContractName,
		MethodName:         systemMethodName,
		InputArgumentArray: (&protocol.ArgumentArrayBuilder{}).Build(),
	})
	if err != nil {
		return nil, err
	}
	if output.CallResult != protocol.EXECUTION_RESULT_SUCCESS {
		return nil, errors.Errorf("_Elections.getElectedValidators call result is %s", output.CallResult)
	}

	argIterator := output.OutputArgumentArray.ArgumentsIterator()
	if !argIterator.HasNext() {
		return nil, errors.Errorf("call system _Elections.getElectedValidators returned corrupt output value")
	}
	arg0 := argIterator.NextArguments()
	if !arg0.IsTypeBytesValue() {
		return nil, errors.Errorf("call system _Elections.getElectedValidators returned corrupt output value")
	}
	joinedAddresses := arg0.BytesValue()

	numAddresses := len(joinedAddresses) / digest.NODE_ADDRESS_SIZE_BYTES
	res := make([]primitives.NodeAddress, numAddresses)
	for i := 0; i < numAddresses; i++ {
		res[i] = joinedAddresses[20*i : 20*(i+1)]
	}
	return res, nil
}

func toNodeAddresses(nodes map[string]config.FederationNode) []primitives.NodeAddress {
	nodeAddresses := make([]primitives.NodeAddress, len(nodes))
	i := 0
	for _, value := range nodes {
		nodeAddresses[i] = value.NodeAddress()
		i++
	}
	return nodeAddresses
}

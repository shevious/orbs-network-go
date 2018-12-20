package gamma

import (
	"context"
	"github.com/orbs-network/orbs-network-go/bootstrap/inmemory"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	ethereumAdapter "github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	gossipAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	nativeProcessorAdapter "github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/harness/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
)

type ethereumConfig struct {
	endpoint string
}

func (e *ethereumConfig) EthereumEndpoint() string {
	return e.endpoint
}

func NewDevelopmentNetwork(ctx context.Context, logger log.BasicLogger) inmemory.NetworkDriver {
	numNodes := 2
	consensusAlgo := consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS
	logger.Info("creating development network")

	leaderKeyPair := keys.EcdsaSecp256K1KeyPairForTests(0)

	federationNodes := make(map[string]config.FederationNode)
	for i := 0; i < int(numNodes); i++ {
		nodeAddress := keys.EcdsaSecp256K1KeyPairForTests(i).NodeAddress()
		federationNodes[nodeAddress.KeyForMap()] = config.NewHardCodedFederationNode(nodeAddress)
	}

	sharedTransport := gossipAdapter.NewMemoryTransport(ctx, logger, federationNodes)
	ethereumConfig := &ethereumConfig{}
	ethereumConnection := ethereumAdapter.NewEthereumRpcConnection(ethereumConfig, logger)

	network := &inmemory.Network{
		Logger:             logger,
		Transport:          sharedTransport,
		EthereumConnection: ethereumConnection,
	}

	for i := 0; i < numNodes; i++ {
		keyPair := keys.EcdsaSecp256K1KeyPairForTests(i)
		cfg := config.ForGamma(
			federationNodes,
			keyPair.NodeAddress(),
			keyPair.PrivateKey(),
			leaderKeyPair.NodeAddress(),
			consensusAlgo,
		)

		// This is awful
		ethereumConfig.endpoint = cfg.EthereumEndpoint()

		metricRegistry := metric.NewRegistry()
		nodeLogger := logger.WithTags(log.Node(cfg.NodeAddress().String()))
		blockPersistence := adapter.NewInMemoryBlockPersistence(nodeLogger, metricRegistry)
		compiler := nativeProcessorAdapter.NewNativeCompiler(cfg, nodeLogger)

		network.AddNode(keyPair.EcdsaSecp256K1KeyPair, cfg, compiler, blockPersistence, metricRegistry, nodeLogger)
	}

	network.CreateAndStartNodes(ctx, numNodes) // must call network.Start(ctx) to actually start the nodes in the network

	return network
}

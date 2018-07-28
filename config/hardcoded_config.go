package config

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
)

//TODO introduce FileSystemConfig
type hardcodedConfig struct {
	networkSize                                  uint32
	nodePublicKey                                primitives.Ed25519PublicKey
	nodePrivateKey                               primitives.Ed25519PrivateKey
	constantConsensusLeader                      primitives.Ed25519PublicKey
	activeConsensusAlgo                          consensus.ConsensusAlgoType
	benchmarkConsensusRoundRetryIntervalMillisec uint32
}

func NewHardCodedConfig(
	networkSize uint32,
	nodePublicKey primitives.Ed25519PublicKey,
	nodePrivateKey primitives.Ed25519PrivateKey,
	constantConsensusLeader primitives.Ed25519PublicKey,
	activeConsensusAlgo consensus.ConsensusAlgoType,
	benchmarkConsensusRoundRetryIntervalMillisec uint32,
) NodeConfig {

	return &hardcodedConfig{
		networkSize:                                  networkSize,
		nodePublicKey:                                nodePublicKey,
		nodePrivateKey:                               nodePrivateKey,
		constantConsensusLeader:                      constantConsensusLeader,
		activeConsensusAlgo:                          activeConsensusAlgo,
		benchmarkConsensusRoundRetryIntervalMillisec: benchmarkConsensusRoundRetryIntervalMillisec,
	}
}

func (c *hardcodedConfig) NetworkSize(asOfBlock uint64) uint32 {
	return c.networkSize
}

func (c *hardcodedConfig) NodePublicKey() primitives.Ed25519PublicKey {
	return c.nodePublicKey
}

func (c *hardcodedConfig) NodePrivateKey() primitives.Ed25519PrivateKey {
	return c.nodePrivateKey
}

func (c *hardcodedConfig) ConstantConsensusLeader() primitives.Ed25519PublicKey {
	return c.constantConsensusLeader
}

func (c *hardcodedConfig) ActiveConsensusAlgo() consensus.ConsensusAlgoType {
	return c.activeConsensusAlgo
}

func (c *hardcodedConfig) BenchmarkConsensusRoundRetryIntervalMillisec() uint32 {
	return c.benchmarkConsensusRoundRetryIntervalMillisec
}

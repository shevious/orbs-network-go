// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package config

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

func emptyConfig() mutableNodeConfig {
	return &config{
		kv: make(map[string]NodeConfigValue),
	}
}

func (c *config) ForNode(nodeAddress primitives.NodeAddress, privateKey primitives.EcdsaSecp256K1PrivateKey) NodeConfig {

	cloned := c.Clone()
	cloned.SetNodeAddress(nodeAddress)
	cloned.SetNodePrivateKey(privateKey)
	return cloned
}

func (c *config) MergeWithFileConfig(source string) (mutableNodeConfig, error) {
	return newFileConfig(c, source)
}

func (c *config) Clone() mutableNodeConfig {
	return &config{
		activeConsensusAlgo:     c.activeConsensusAlgo,
		constantConsensusLeader: c.constantConsensusLeader,
		genesisValidatorNodes:   c.genesisValidatorNodes,
		gossipPeers:             c.gossipPeers,
		nodePrivateKey:          c.nodePrivateKey,
		nodeAddress:             c.nodeAddress,
		kv:                      cloneMap(c.kv),
	}
}

func cloneMap(kv map[string]NodeConfigValue) (cloned map[string]NodeConfigValue) {
	cloned = make(map[string]NodeConfigValue)
	for k, v := range kv {
		cloned[k] = v
	}
	return cloned
}

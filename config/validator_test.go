// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package config

import (
	"encoding/hex"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func genesisValidators() map[string]ValidatorNode {
	return map[string]ValidatorNode{
		"v1": NewHardCodedValidatorNode([]byte{0x0}),
		"v2": NewHardCodedValidatorNode([]byte{0x1}),
	}
}

func TestValidateConfig(t *testing.T) {
	v := validator{log.DefaultTestingLogger(t)}
	cfg := defaultProductionConfig()
	cfg.SetNodeAddress(defaultNodeAddress())
	cfg.SetNodePrivateKey(defaultPrivateKey())
	cfg.SetGenesisValidatorNodes(genesisValidators())

	require.NotPanics(t, func() {
		v.ValidateNodeLogic(cfg)
	})
}

func TestValidateConfig_PanicsOnInvalidValue(t *testing.T) {
	v := validator{log.DefaultTestingLogger(t)}

	cfg := defaultProductionConfig()
	cfg.SetGenesisValidatorNodes(genesisValidators())
	cfg.SetDuration(BLOCK_SYNC_NO_COMMIT_INTERVAL, 1*time.Millisecond)

	require.Panics(t, func() {
		v.ValidateNodeLogic(cfg)
	})
}

func TestValidateConfig_DoesNotPanicOnProperKeys(t *testing.T) {
	v := validator{log.DefaultTestingLogger(t)}

	cfg := defaultProductionConfig()
	cfg.SetGenesisValidatorNodes(genesisValidators())
	cfg.SetNodeAddress(defaultNodeAddress())
	cfg.SetNodePrivateKey(defaultPrivateKey())

	require.NotPanics(t, func() {
		v.ValidateNodeLogic(cfg)
	})
}

func TestValidateConfig_PanicsOnInvalidKeys(t *testing.T) {
	v := validator{log.DefaultTestingLogger(t)}

	cfg := defaultProductionConfig()
	cfg.SetGenesisValidatorNodes(genesisValidators())
	cfg.SetNodeAddress(defaultNodeAddress())
	cfg.SetNodePrivateKey(wrongPrivateKey())

	require.Panics(t, func() {
		v.ValidateNodeLogic(cfg)
	})
}

func defaultNodeAddress() primitives.NodeAddress {
	addr, _ := hex.DecodeString("a328846cd5b4979d68a8c58a9bdfeee657b34de7")
	return primitives.NodeAddress(addr)
}

func defaultPrivateKey() primitives.EcdsaSecp256K1PrivateKey {
	key, _ := hex.DecodeString("901a1a0bfbe217593062a054e561e708707cb814a123474c25fd567a0fe088f8")
	return primitives.EcdsaSecp256K1PrivateKey(key)
}

func wrongPrivateKey() primitives.EcdsaSecp256K1PrivateKey {
	key, _ := hex.DecodeString("00001a0bfbe217593062a054e561e708707cb814a123474c25fd567a0fe088f8")
	return primitives.EcdsaSecp256K1PrivateKey(key)
}

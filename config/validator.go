// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package config

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/scribe/log"
	"reflect"
	"runtime"
	"strings"
	"time"
)

type validator struct {
	logger log.Logger
}

func NewValidator(logger log.Logger) *validator {
	return &validator{logger: logger}
}

func (v *validator) ValidateNodeLogic(cfg NodeConfig) {
	v.requireGT(cfg.BlockSyncNoCommitInterval, cfg.BenchmarkConsensusRetryInterval, "node sync timeout must be greater than benchmark consensus timeout")
	v.requireGT(cfg.BlockSyncNoCommitInterval, cfg.LeanHelixConsensusRoundTimeoutInterval, "node sync timeout must be greater than lean helix round timeout")
	v.requireNonEmpty(cfg.NodeAddress(), "node address must not be empty")
	v.requireNonEmpty(cfg.NodePrivateKey(), "node private key must not be empty")
	v.requireNonEmptyValidatorMap(cfg.GenesisValidatorNodes(), "genesis validator list must not be empty")
	v.requireCorrectNodeAddressAndPrivateKey(cfg.NodeAddress(), cfg.NodePrivateKey())
}

func (v *validator) ValidateMainNode(cfg NodeConfig) {
	v.requireNonEmptyPeerMap(cfg.GossipPeers(), "gossip peer list must not be empty")
}

func (v *validator) requireGT(d1 func() time.Duration, d2 func() time.Duration, msg string) {
	if d1() < d2() {
		panic(fmt.Sprintf("%s; %s=%s is greater than %s=%s", msg, funcName(d1), d1(), funcName(d2), d2()))
	}
}

func (v *validator) requireNonEmpty(bytes []byte, msg string) {
	if len(bytes) == 0 {
		panic(msg)
	}
}

func (v *validator) requireNonEmptyValidatorMap(nodes map[string]ValidatorNode, msg string) {
	if len(nodes) == 0 {
		panic(msg)
	}
}

func (v *validator) requireNonEmptyPeerMap(gossipPeers map[string]GossipPeer, msg string) {
	if len(gossipPeers) == 0 {
		panic(msg)
	}
}

func (v *validator) requireCorrectNodeAddressAndPrivateKey(address primitives.NodeAddress, key primitives.EcdsaSecp256K1PrivateKey) {
	msg := []byte{
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
	}

	sign, err := signature.SignEcdsaSecp256K1(key, msg)
	if err != nil {
		panic(fmt.Sprintf("could not create test sign: %s", err))
	}

	recoveredPublicKey, err := signature.RecoverEcdsaSecp256K1(msg, sign)
	if err != nil {
		panic(fmt.Sprintf("could not recover public key from test sign: %s", err))
	}

	recoveredNodeAddress := digest.CalcNodeAddressFromPublicKey(recoveredPublicKey)
	if bytes.Compare(address, recoveredNodeAddress) != 0 {
		panic(fmt.Sprintf("node address %s derived from secret key does not match provided node address %s",
			hex.EncodeToString(recoveredNodeAddress), hex.EncodeToString(address)))
	}
}

func funcName(i interface{}) string {
	fullName := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	lastDot := strings.LastIndex(fullName, ".")
	return fullName[lastDot+1:]
}

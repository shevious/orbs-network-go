// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package info_systemcontract

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/address"
)

func isAlive() string {
	return "alive"
}

func getSignerAddress() []byte {
	return address.GetSignerAddress()
}

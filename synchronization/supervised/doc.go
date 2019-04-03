// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

// Package supervised provides basic supervision abilities for running goroutines,
// namely making sure that panics are not swallowed and that long-running goroutines
// are restarted if they crash
package supervised

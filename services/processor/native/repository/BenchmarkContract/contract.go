package benchmarkcontract

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/state"
)

var EXPORTS = sdk.Export(add, set, get, argTypes, throw)

func _init() {
	state.WriteUint64ByKey("initialized", 1)
}

func nop() {
}

func add(a uint64, b uint64) uint64 {
	return a + b
}

func set(a uint64) {
	state.WriteUint64ByKey("example-key", a)
}

func get() uint64 {
	return state.ReadUint64ByKey("example-key")
}

func argTypes(a1 uint32, a2 uint64, a3 string, a4 []byte) (uint32, uint64, string, []byte) {
	return a1 + 1, a2 + 1, a3 + "1", append(a4, 0x01)
}

func throw() {
	panic("example error returned by contract")
}

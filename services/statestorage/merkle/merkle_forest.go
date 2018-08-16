package merkle

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"strings"
)

type TrieId uint64
type Proof []*Node

const trieRadix = 256 // base of the merkle trie. TODO change to 16

var zeroValueHash = hash.CalcSha256([]byte{})

type Node struct {
	path     string // TODO replace with []byte + parity bool when moving to trieRadix = 16
	value    primitives.Sha256
	branches [trieRadix]primitives.MerkleSha256
}

var emptyNode = &Node{value: zeroValueHash}
var emptyNodeHash = emptyNode.hash()

func createNode(path string, valueHash primitives.Sha256) *Node {
	return &Node{
		path:     path,
		value:    valueHash,
		branches: [trieRadix]primitives.MerkleSha256{},
	}
}
func (n *Node) hasValue() bool {
	return !zeroValueHash.Equal(n.value)
}
func (n *Node) hash() primitives.MerkleSha256 {
	// TODO replace this with membuffers serialization. Sprintf should not reach production
	serializedNode := fmt.Sprintf("%+v", n)
	return primitives.MerkleSha256(hash.CalcSha256([]byte(serializedNode)))
}
func (n *Node) clone() *Node {
	newBranches := [trieRadix]primitives.MerkleSha256{}
	copy(newBranches[:], n.branches[:])
	result := &Node{
		path:     n.path,
		value:    n.value, // TODO - copy?
		branches: newBranches,
	}
	for k, v := range n.branches {
		result.branches[k] = v // TODO - copy?
	}
	return result
}
func (n *Node) hasChildren() bool {
	for _, v := range n.branches {
		if len(v) != 0 {
			return true
		}
	}
	return false
}

type Forest struct {
	roots   map[TrieId]primitives.MerkleSha256
	nodes   map[string]*Node
	topRoot TrieId
}

func NewForest() *Forest {
	return &Forest{
		roots: map[TrieId]primitives.MerkleSha256{0: emptyNodeHash},
		nodes: map[string]*Node{emptyNodeHash.KeyForMap(): emptyNode},
	}
}

// return the merkle trie root hash for one trie in f.
// typically there is a merkle trie per committed block, reflecting the state snapshot for each block height.
func (f *Forest) GetRootHash(trieId TrieId) (primitives.MerkleSha256, error) {
	return f.roots[trieId], nil
}

// return the merkle trie root hash for the last committed block
func (f *Forest) GetTopRootHash() (primitives.MerkleSha256, error) {
	return f.roots[f.topRoot], nil
}

func (f *Forest) connectChildToParentAndSaveChild(childNode, parentNode *Node, selector byte) {
	childHash := childNode.hash()
	parentNode.branches[selector] = childHash
	f.nodes[childHash.KeyForMap()] = childNode
}

func (f *Forest) addSingleEntry(path string, valueHash primitives.Sha256) TrieId {
	currentRoot := f.nodes[f.roots[f.topRoot].KeyForMap()]
	newRoot := f.add(currentRoot, path, valueHash)
	sha256s := newRoot.hash()
	f.nodes[sha256s.KeyForMap()] = newRoot
	f.topRoot++
	f.roots[f.topRoot] = sha256s
	return f.topRoot
}

func (f *Forest) add(currentNode *Node, path string, valueHash primitives.Sha256) *Node {
	newNode := currentNode.clone()
	if currentNode.path == path { // existing leaf node updated
		newNode.value = valueHash
		return newNode
	}

	if strings.HasPrefix(path, currentNode.path) {
		if !currentNode.hasValue() && !currentNode.hasChildren() { // this node has no children and no value, replace it
			newNode = createNode(path, valueHash)
			return newNode
		}

		branchSelector := path[len(currentNode.path)]
		childPath := path[len(currentNode.path)+1:]
		var newChild *Node
		if branchHash := currentNode.branches[branchSelector]; len(branchHash) != 0 {
			newChild = f.add(f.nodes[branchHash.KeyForMap()], childPath, valueHash)
		} else {
			newChild = createNode(childPath, valueHash)
		}
		f.connectChildToParentAndSaveChild(newChild, newNode, branchSelector)
		return newNode
	}

	if strings.HasPrefix(currentNode.path, path) { // "insert" a leaf node along the path of currentNode
		branchSelector := newNode.path[len(path)]
		newNode.path = newNode.path[len(path)+1:]
		newParent := createNode(path, valueHash)
		f.connectChildToParentAndSaveChild(newNode, newParent, branchSelector)
		return newParent
	}

	// current node replaced by a new branch node, so that current node is one child and new node is second child
	i := 0
	for i = 0; i < len(currentNode.path) && i < len(path) && currentNode.path[i] == path[i]; i++ {
	}
	newCommonPath := path[:i]
	newParent := createNode(newCommonPath, zeroValueHash)
	newChild := createNode(path[i+1:], valueHash)
	f.connectChildToParentAndSaveChild(newChild, newParent, path[i])

	newNode.path = newNode.path[i+1:]
	f.connectChildToParentAndSaveChild(newNode, newParent, currentNode.path[i])
	return newParent
}

// appends diffs to the top trie building new nodes as needed and returns the new trie id
func (f *Forest) Update(diffs []*protocol.ContractStateDiff) TrieId {
	for _, diff := range diffs {
		contract := diff.StringContractName()
		for i := diff.StateDiffsIterator(); i.HasNext(); {
			record := i.NextStateDiffs()
			path := contract + record.StringKey()
			f.addSingleEntry(path, hash.CalcSha256([]byte(record.StringValue())))
		}
	}
	return f.topRoot
}

func (f *Forest) updateStringEntries(keyValues ...string) TrieId {
	if len(keyValues)%2 != 0 {
		panic("expected key value pairs")
	}
	for i := 0; i < len(keyValues); i = i + 2 {
		f.addSingleEntry(keyValues[i], hash.CalcSha256([]byte(keyValues[i+1])))
	}
	return f.topRoot
}

// extract and return a verifiable proof for the value of key in the state snapshot reflected by trieId (corresponding to some block height)
func (f *Forest) GetProof(trieId TrieId, contract string, key string) (Proof, error) {
	fullPath := contract + key
	root := f.roots[trieId]
	currentNode, exists := f.nodes[root.KeyForMap()]
	proof := make(Proof, 0, 10)
	proof = append(proof, currentNode)

	for p := fullPath; exists && strings.HasPrefix(p, currentNode.path); {
		p = p[len(currentNode.path):]

		if len(p) != 0 {
			currentNode, exists = f.nodes[currentNode.branches[p[0]].KeyForMap()]
			if exists {
				proof = append(proof, currentNode)
			}
			p = p[1:]
		} else {
			break
		}
	}
	return proof, nil
}

// return true if proof and merkle rootHash validate value for key. false if it confirms value does not match key
// return an error if the proof is inconsistent internally, or, with key, value or rootHash
func (f *Forest) Verify(rootHash primitives.MerkleSha256, proof Proof, contract string, key string, value string) (bool, error) {
	//TODO split the case where we compare against zero value - to simplify determineValueHashByProof
	valueSha256 := hash.CalcSha256([]byte(value))
	expectedHash, err := determineValueHashByProof(proof, contract+key, rootHash)
	if err != nil {
		return false, err
	}
	return valueSha256.Equal(expectedHash), nil

}

func determineValueHashByProof(proof Proof, path string, parentHash primitives.MerkleSha256) (primitives.Sha256, error) {
	if len(proof) == 0 { // proof has ended before a positive conclusion could be reached
		return nil, errors.Errorf("Proof incomplete")
	}

	node := proof[0] // each iteration inspects the top (remaining) node in the proof

	if !node.hash().Equal(parentHash) { // validate current node against expected hash
		return nil, errors.Errorf("Merkle root mismatch or proof may have been tampered with")
	}

	if path == node.path { // current node consumes the remainder of the key. check hasValue value
		if node.hasValue() { // key is in trie
			return node.value, nil
		} else { // key is not in trie
			return zeroValueHash, nil
		}
	} else if len(path) <= len(node.path) { // key is not in trie
		return zeroValueHash, nil
	}

	if !strings.HasPrefix(path, node.path) { // key is not in trie
		return zeroValueHash, nil
	}

	// follow branch: get the hash code of the next expected node for our key
	nextHash := node.branches[path[len(node.path)]]

	if len(nextHash) == 0 { // key is not in trie
		return zeroValueHash, nil
	}

	// current top node passes validation, proceed to the next node
	return determineValueHashByProof(proof[1:], path[len(node.path)+1:], nextHash)

}
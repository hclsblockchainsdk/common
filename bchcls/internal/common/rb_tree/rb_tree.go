/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

package rb_tree

import (
	"bytes"
	"encoding/json"
	"os"

	"common/bchcls/cached_stub"
	"common/bchcls/custom_errors"
	"common/bchcls/internal/common/global"
	"common/bchcls/utils"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/ledger/queryresult"
	"github.com/pkg/errors"
)

var logger = shim.NewLogger("rb_tree")

// Init sets up the datastore package by adding default ledger connection.
func Init(stub cached_stub.CachedStubInterface, logLevel ...shim.LoggingLevel) ([]byte, error) {
	if len(logLevel) > 0 {
		logger.SetLevel(logLevel[0])
	}
	return nil, nil
}

/*
 * The following implementation of Red-Black tree has been implemented
 * following the implementation example of the following site:
 * https://en.wikipedia.org/wiki/Red%E2%80%93black_tree
 */
var DEBUG_PRINT_TREE = false
var DEBUG_TREE = false

const PREFIX = "RBT_"
const LEAF = ""
const RED = "R"
const BLACK = "B"

type node struct {
	Key    string `json:"key"`
	Value  []byte `json:"value"`
	Parent string `json:"parent"`
	Color  string `json:"color"`
	Left   string `json:"left"`
	Right  string `json:"right"`
}

var NULL = node{Key: "", Value: nil, Parent: "", Color: BLACK, Left: LEAF, Right: LEAF}

type RBTree struct {
	Stub     cached_stub.CachedStubInterface
	Cache    map[string][]byte
	ToDelete map[string]bool
	TreeName string
	Prefix   string
	Root     string
}

func (n *node) IsNull() bool {
	if len(n.Key) == 0 {
		return true
	} else {
		return false
	}
}

func (n *node) LedgerKey(prefix string) string {
	if len(n.Key) == 0 {
		return ""
	}
	return prefix + n.Key
}

func (t *RBTree) GetNodeLedgerKey(n node) string {
	if len(n.Key) == 0 {
		return ""
	}
	return t.Prefix + n.Key
}
func (t *RBTree) GetStateCached(key string) ([]byte, error) {
	if val, ok := t.Cache[key]; ok {
		// return from cache
		return val, nil
	} else {
		value, err := t.Stub.GetState(key)
		if err != nil {
			return value, err
		} else {
			//save the value to cache
			t.Cache[key] = value
			return value, nil
		}
	}
}

func (t *RBTree) PutStateCached(key string, value []byte) error {
	delete(t.Cache, key)
	t.Cache[key] = value
	t.ToDelete[key] = false
	return t.Stub.PutState(key, value)
}

func (t *RBTree) DelStateCached(key string) error {
	t.Cache[key] = nil
	t.ToDelete[key] = true
	return t.Stub.DelState(key)
}

func (t *RBTree) GetNode(key string) (node, error) {
	var n = NULL
	if len(key) == 0 {
		return NULL, nil
	}

	var nodeBytes []byte
	var err error
	nodeBytes, err = t.GetStateCached(t.Prefix + key)

	if err != nil {
		err = errors.Wrap(err, "Failed to get ledger key")
		return NULL, err
	} else if nodeBytes != nil {
		err = json.Unmarshal(nodeBytes, &n)
		if err != nil {
			err = errors.WithStack(&custom_errors.UnmarshalError{Type: "node"})
			return NULL, err
		}
	}

	return n, nil
}

func (t *RBTree) SaveNode(n node) error {
	if n.IsNull() {
		return nil
	}
	nodeBytes, err := json.Marshal(&n)
	if err != nil {
		return errors.WithStack(&custom_errors.MarshalError{Type: "node"})
	}
	return t.PutStateCached(n.LedgerKey(t.Prefix), nodeBytes)
}

func (t *RBTree) DelNode(n node) error {
	if n.IsNull() {
		return nil
	}
	return t.DelStateCached(n.LedgerKey(t.Prefix))
}

func (t *RBTree) GetRoot() (node, error) {
	root := NULL
	rootIDBytes, err := t.GetStateCached(t.Prefix)
	if err != nil {
		return NULL, err
	}
	if rootIDBytes != nil {
		root, err = t.GetNode(string(rootIDBytes))
		if err != nil {
			return NULL, err
		}
	} else {
		return NULL, nil
	}
	t.Root = root.Key
	return root, nil
}

func (t *RBTree) SaveRoot(r node) error {
	oldr := t.Root
	t.Root = r.Key
	if r.IsNull() {
		return t.DelStateCached(t.Prefix)
	}

	if r.Key != oldr {
		rootBytes := []byte(r.Key)
		return t.PutStateCached(t.Prefix, rootBytes)
	}
	return nil
}

// find root starting from a node n
func (t *RBTree) FindRoot(n node) (node, error) {
	root := n
	p, err := t.Parent(root)
	if err != nil {
		return NULL, err
	}
	for !p.IsNull() {
		root = p
		p, err = t.Parent(root)
		if err != nil {
			return NULL, err
		}
	}
	return root, nil
}

// Return Parent
func (t *RBTree) Parent(n node) (node, error) {
	return t.GetNode(n.Parent)
}

func (t *RBTree) Grandparent(n node) (node, error) {
	p, err := t.Parent(n)
	if err != nil {
		return NULL, err
	}
	//no parent, no grandparent
	if p.IsNull() {
		return NULL, nil
	}
	return t.Parent(p)
}

func (t *RBTree) Sibling(n node) (node, error) {
	p, err := t.Parent(n)
	if err != nil {
		return NULL, err
	}
	//No parent, no sibling
	if p.IsNull() {
		return NULL, nil
	}
	if n.Key == p.Left {
		return t.GetNode(p.Right)
	} else {
		return t.GetNode(p.Left)
	}
}

func (t *RBTree) Uncle(n node) (node, error) {
	p, err := t.Parent(n)
	if err != nil {
		return NULL, err
	}
	//No parent, no Uncle
	if p.IsNull() {
		return NULL, nil
	}
	return t.Sibling(p)
}

func (t *RBTree) RotateLeft(n node) error {
	if DEBUG_TREE {
		logger.Debugf("rotateLeft : node: %v", n)
	}
	nRight, err := t.GetNode(n.Right)
	if err != nil {
		return err
	}
	if nRight.IsNull() {
		return errors.New("since the leaves of a red-black tree are empty, they cannot become internal nodes")
	}
	p, err := t.Parent(n)
	if err != nil {
		return err
	}
	n.Right = nRight.Left
	nRight.Left = n.Key
	n.Parent = nRight.Key

	nRightLeft, err := t.GetNode(n.Right)
	if err != nil {
		return err
	}
	if !nRightLeft.IsNull() {
		nRightLeft.Parent = n.Key
		err := t.SaveNode(nRightLeft)
		if err != nil {
			return err
		}
	}

	if !p.IsNull() {
		if n.Key == p.Left {
			p.Left = nRight.Key
		} else if n.Key == p.Right {
			p.Right = nRight.Key
		}
		err := t.SaveNode(p)
		if err != nil {
			return err
		}
	}
	nRight.Parent = p.Key
	err = t.SaveNode(nRight)
	if err != nil {
		return err
	}
	if p.IsNull() {
		//now nRight is the new root
		err := t.SaveRoot(nRight)
		if err != nil {
			return nil
		}
	}
	return t.SaveNode(n)
}

func (t *RBTree) RotateRight(n node) error {
	if DEBUG_TREE {
		logger.Debugf("rotateRight : node: %v", n)
	}
	nLeft, err := t.GetNode(n.Left)
	if err != nil {
		return err
	}
	if nLeft.IsNull() {
		return errors.New("since the leaves of a red-black tree are empty, they cannot become internal nodes")
	}
	p, err := t.Parent(n)
	if err != nil {
		return err
	}
	n.Left = nLeft.Right
	nLeft.Right = n.Key
	n.Parent = nLeft.Key

	nLeftRight, err := t.GetNode(n.Left)
	if err != nil {
		return err
	}
	if !nLeftRight.IsNull() {
		nLeftRight.Parent = n.Key
		err := t.SaveNode(nLeftRight)
		if err != nil {
			return err
		}
	}

	if !p.IsNull() {
		if n.Key == p.Left {
			p.Left = nLeft.Key
		} else if n.Key == p.Right {
			p.Right = nLeft.Key
		}
		err := t.SaveNode(p)
		if err != nil {
			return err
		}
	}
	nLeft.Parent = p.Key
	err = t.SaveNode(nLeft)
	if err != nil {
		return err
	}
	if p.IsNull() {
		//now nLeft is the new root
		err := t.SaveRoot(nLeft)
		if err != nil {
			return nil
		}
	}
	return t.SaveNode(n)
}

func (t *RBTree) Insert(key string, value []byte) error {
	if DEBUG_TREE {
		logger.Debugf("-->insert : %v", key)
	}
	// NULL key ignore
	if len(key) == 0 {
		return nil
	}

	//check if node already exist
	n, err := t.GetNode(key)
	if err == nil && !n.IsNull() {
		//update only value
		if !bytes.Equal(n.Value, value) {
			n.Value = value
			return t.SaveNode(n)
		}
		return nil
	} else {
		n = NULL
		n.Key = key
		n.Value = value
	}

	root, err := t.GetRoot()
	if err != nil {
		return err
	}

	// insert new node into the current tree
	err = t.insertRecurse(root, n)
	if err != nil {
		return err
	}

	// repair the tree in case any of the red-black properties have been violated
	// get updated n
	n, err = t.GetNode(n.Key)
	err = t.insertRepairTree(n)
	if err != nil {
		return err
	}

	// find the new root
	// get updated n
	n, err = t.GetNode(n.Key)
	newRoot, err := t.FindRoot(n)
	if err != nil {
		return err
	}
	if DEBUG_TREE {
		logger.Debug("new Root/Node:", newRoot, n)
	}

	// update root
	err = t.SaveRoot(newRoot)
	if err != nil {
		return err
	}

	if DEBUG_TREE {
		logger.Debug("<--insert ")
	}
	return nil
}

func (t *RBTree) insertRecurse(root node, n node) error {
	if DEBUG_TREE {
		logger.Debugf("insertRecurse : root: %v : node: %v", root, n)
	}
	// recursively descend the tree until a leaf is found
	if !root.IsNull() && n.Key < root.Key {
		if root.Left != LEAF {
			rLeft, err := t.GetNode(root.Left)
			if err != nil {
				return err
			}
			return t.insertRecurse(rLeft, n)
		} else {
			root.Left = n.Key
			err := t.SaveNode(root)
			if err != nil {
				return err
			}
		}
	} else if !root.IsNull() {
		if root.Right != LEAF {
			rRight, err := t.GetNode(root.Right)
			if err != nil {
				return err
			}
			return t.insertRecurse(rRight, n)
		} else {
			root.Right = n.Key
			err := t.SaveNode(root)
			if err != nil {
				return err
			}
		}
	}

	// insert new node n
	n.Parent = root.Key
	n.Left = LEAF
	n.Right = LEAF
	n.Color = RED
	return t.SaveNode(n)
}

func (t *RBTree) insertRepairTree(n node) error {
	if DEBUG_TREE {
		logger.Debugf("insertRepairTree : node: %v", n)
	}
	p, err := t.Parent(n)
	if err != nil {
		return err
	}
	if p.IsNull() {
		err = t.insertCase1(n)
	} else if p.Color == BLACK {
		err = t.insertCase2(n)
	} else {
		u, err2 := t.Uncle(n)
		if err2 != nil {
			return err
		}
		if u.Color == RED {
			err = t.insertCase3(n)
		} else {
			err = t.insertCase4(n)
		}
	}
	return err
}

// N is at the root of the tree
func (t *RBTree) insertCase1(n node) error {
	if DEBUG_TREE {
		logger.Debugf("insertCase1 : node: %v", n)
	}
	if len(n.Parent) == 0 {
		n.Color = BLACK
		return t.SaveNode(n)
	}
	return nil
}

// P is black
func (t *RBTree) insertCase2(n node) error {
	if DEBUG_TREE {
		logger.Debugf("insertCase2 : node: %v", n)
	}
	// Do nothing since tree is still valid
	return nil
}

// Both the parent P and the uncle U are red
func (t *RBTree) insertCase3(n node) error {
	if DEBUG_TREE {
		logger.Debugf("insertCase3 : node: %v", n)
	}
	p, err := t.Parent(n)
	if err != nil {
		return err
	}
	p.Color = BLACK
	err = t.SaveNode(p)
	if err != nil {
		return err
	}

	u, err2 := t.Uncle(n)
	if err2 != nil {
		return err
	}
	u.Color = BLACK
	err = t.SaveNode(u)
	if err != nil {
		return err
	}

	g, err := t.Parent(p)
	if err != nil {
		return err
	}
	g.Color = RED
	err = t.SaveNode(g)
	if err != nil {
		return err
	}

	return t.insertRepairTree(g)
}

// The parent P is red but the uncle U is black
func (t *RBTree) insertCase4(n node) error {
	if DEBUG_TREE {
		logger.Debugf("insertCase4 : node: %v", n)
	}
	p, err := t.Parent(n)
	if err != nil {
		return err
	}
	g, err := t.Parent(p)
	if err != nil {
		return err
	}
	if g.Left == p.Key && p.Right == n.Key {
		err = t.RotateLeft(p)
		if err != nil {
			return err
		}

		n, _ = t.GetNode(n.Key) // get updated n
		n, err = t.GetNode(n.Left)
		if err != nil {
			return err
		}
	} else if g.Right == p.Key && p.Left == n.Key {
		err = t.RotateRight(p)
		if err != nil {
			return err
		}

		n, _ = t.GetNode(n.Key) // get updated n
		n, err = t.GetNode(n.Right)
		if err != nil {
			return err
		}
	}

	return t.insertCase4step2(n)
}

//The current node N is now certain to be on the "outside" of the subtree under G (left of left child or right of right child)
func (t *RBTree) insertCase4step2(n node) error {
	if DEBUG_TREE {
		logger.Debugf("insertCase4step2 : node: %v", n)
	}
	p, err := t.Parent(n)
	if err != nil {
		return err
	}
	g, err := t.Parent(p)
	if err != nil {
		return err
	}
	if p.Left == n.Key {
		err = t.RotateRight(g)
		if err != nil {
			return err
		}
	} else {
		err = t.RotateLeft(g)
		if err != nil {
			return err
		}
	}

	p, _ = t.GetNode(p.Key) // get updated p
	p.Color = BLACK
	err = t.SaveNode(p)
	if err != nil {
		return err
	}

	g, _ = t.GetNode(g.Key) // get updated g
	g.Color = RED
	return t.SaveNode(g)
}

func (t *RBTree) Remove(key string) error {
	if DEBUG_TREE {
		logger.Debugf("-->remove : %v", key)
	}
	n, err := t.GetNode(key)
	if err != nil {
		return err
	}
	// don't need to remove if we don't have a node
	if n.IsNull() {
		return nil
	}

	if n.Left != LEAF && n.Right != LEAF { //both children are present
		r, err := t.GetNode(n.Right)
		if err != nil {
			return err
		}
		m, err := t.FindMin(r)
		if err != nil {
			return err
		}
		if DEBUG_TREE {
			logger.Debugf("min node %v", m)
		}
		// remove the min node
		err = t.deleteOneChild(m)
		if err != nil {
			return err
		}

		newN, err := t.GetNode(n.Key)
		if err != nil {
			return err
		}

		p, err := t.Parent(newN)
		if err != nil {
			return err
		}

		//update current key with min
		newN.Key = m.Key
		newN.Value = m.Value
		err = t.SaveNode(newN)
		if err != nil {
			return err
		}

		//update parent
		if !p.IsNull() {
			if p.Left == n.Key {
				p.Left = m.Key
			} else {
				p.Right = m.Key
			}
			err := t.SaveNode(p)
			if err != nil {
				return err
			}
		} else {
			//newN is the root
			err := t.SaveRoot(newN)
			if err != nil {
				return err
			}
		}

		//update children
		if newN.Right != LEAF {
			c, err := t.GetNode(newN.Right)
			if err != nil {
				return err
			}
			c.Parent = newN.Key
			err = t.SaveNode(c)
			if err != nil {
				return err
			}
		}
		if newN.Left != LEAF {
			c, err := t.GetNode(newN.Left)
			if err != nil {
				return err
			}
			c.Parent = newN.Key
			err = t.SaveNode(c)
			if err != nil {
				return err
			}
		}
		if DEBUG_TREE {
			logger.Debugf("replace %v with %v", n.Key, newN)
		}
		err = t.DelNode(n)
		if err != nil {
			return err
		}
	} else { //at most one non-leaf child
		err := t.deleteOneChild(n)
		if err != nil {
			return err
		}
	}
	if DEBUG_TREE {
		logger.Debug("<--remove")
	}
	return nil
}

func (t *RBTree) FindMin(n node) (node, error) {
	m := n
	var err error
	for m.Left != LEAF {
		m, err = t.GetNode(m.Left)
		if err != nil {
			return NULL, err
		}
	}
	return m, nil
}

// replace current node n with node child
// returns updated child node
func (t *RBTree) replaceNode(n node, child node) (node, error) {
	if DEBUG_TREE {
		logger.Debugf("replaceNode: n %v, chiid %v", n, child)
	}
	child.Parent = n.Parent
	p, err := t.Parent(n)
	if err != nil {
		return child, err
	}

	if n.Key == p.Left {
		p.Left = child.Key
	} else {
		p.Right = child.Key
	}

	if !child.IsNull() {
		err := t.SaveNode(child)
		if err != nil {
			return child, err
		}
	}
	if !p.IsNull() {
		err := t.SaveNode(p)
		if err != nil {
			return child, err
		}
	}
	return child, nil
}

//precondition: n has at most one none-leaf child
func (t *RBTree) deleteOneChild(n node) error {
	if DEBUG_TREE {
		logger.Debugf("deleteOneChild: n %v", n)
	}
	var child node
	var err error
	if n.Right == LEAF {
		child, err = t.GetNode(n.Left)
		if err != nil {
			return err
		}
	} else {
		child, err = t.GetNode(n.Right)
		if err != nil {
			return err
		}
	}
	child, err = t.replaceNode(n, child)
	if n.Color == BLACK {
		if child.Color == RED {
			child.Color = BLACK
			err = t.SaveNode(child)
			if err != nil {
				return err
			}
		} else {
			t.deleteCase1(child)
		}
	}
	return t.DelNode(n)
}

//N is the new root
func (t *RBTree) deleteCase1(n node) error {
	if DEBUG_TREE {
		logger.Debugf("deleteCase1: n %v", n)
	}
	if DEBUG_PRINT_TREE {
		t.PrintTree("", n)
	}
	if len(n.Parent) == 0 {
		// n is the new root
		return t.SaveRoot(n)
	} else {
		return t.deleteCase2(n)
	}
	return nil
}

//S is red
func (t *RBTree) deleteCase2(n node) error {
	if DEBUG_TREE {
		logger.Debugf("deleteCase2: n %v", n)
	}
	if DEBUG_PRINT_TREE {
		t.PrintTree("", n)
	}
	s, err := t.Sibling(n)
	if err != nil {
		return err
	}

	if s.Color == RED {
		p, err := t.Parent(n)
		if err != nil {
			return err
		}
		p.Color = RED
		err = t.SaveNode(p)
		if err != nil {
			return err
		}
		s.Color = BLACK
		err = t.SaveNode(s)
		if err != nil {
			return err
		}

		if n.Key == p.Left {
			err := t.RotateLeft(p)
			if err != nil {
				return err
			}
		} else {
			err := t.RotateRight(p)
			if err != nil {
				return err
			}
		}
	}
	return t.deleteCase3(n)
}

// P, S, and S's children are black
func (t *RBTree) deleteCase3(n node) error {
	if DEBUG_TREE {
		logger.Debugf("deleteCase3: n %v", n)
	}
	if DEBUG_PRINT_TREE {
		t.PrintTree("", n)
	}
	s, err := t.Sibling(n)
	if err != nil {
		return err
	}
	p, err := t.Parent(n)
	if err != nil {
		return err
	}
	sl, err := t.GetNode(s.Left)
	if err != nil {
		return err
	}
	sr, err := t.GetNode(s.Right)
	if err != nil {
		return err
	}

	if (p.Color == BLACK) && (s.Color == BLACK) && (sl.Color == BLACK) && (sr.Color == BLACK) {
		s.Color = RED
		err := t.SaveNode(s)
		if err != nil {
			return err
		}
		return t.deleteCase1(p)
	} else {
		return t.deleteCase4(n)
	}
}

//S and S's children are black, but P is red.
func (t *RBTree) deleteCase4(n node) error {
	if DEBUG_TREE {
		logger.Debugf("deleteCase4: n %v", n)
	}
	if DEBUG_PRINT_TREE {
		t.PrintTree("", n)
	}
	s, err := t.Sibling(n)
	if err != nil {
		return err
	}
	p, err := t.Parent(n)
	if err != nil {
		return err
	}
	sl, err := t.GetNode(s.Left)
	if err != nil {
		return err
	}
	sr, err := t.GetNode(s.Right)
	if err != nil {
		return err
	}

	if (p.Color == RED) && (s.Color == BLACK) && (sl.Color == BLACK) && (sr.Color == BLACK) {
		s.Color = RED
		err = t.SaveNode(s)
		if err != nil {
			return err
		}
		p.Color = BLACK
		return t.SaveNode(p)
	} else {
		return t.deleteCase5(n)
	}
}

//S is black, S's left child is red, S's right child is black, and N is the left child of its parent.
func (t *RBTree) deleteCase5(n node) error {
	if DEBUG_TREE {
		logger.Debugf("deleteCase5: n %v", n)
	}
	if DEBUG_PRINT_TREE {
		t.PrintTree("", n)
	}
	s, err := t.Sibling(n)
	if err != nil {
		return err
	}
	p, err := t.Parent(n)
	if err != nil {
		return err
	}
	sl, err := t.GetNode(s.Left)
	if err != nil {
		return err
	}
	sr, err := t.GetNode(s.Right)
	if err != nil {
		return err
	}

	if s.Color == BLACK {
		if (n.Key == p.Left) && (sr.Color == BLACK) && (sl.Color == RED) {
			s.Color = RED
			err = t.SaveNode(s)
			if err != nil {
				return err
			}
			sl.Color = BLACK
			err = t.SaveNode(sl)
			if err != nil {
				return err
			}
			err = t.RotateRight(s)
			if err != nil {
				return err
			}
		} else if (n.Key == p.Right) && (sl.Color == BLACK) && (sr.Color == RED) {
			s.Color = RED
			err = t.SaveNode(s)
			if err != nil {
				return err
			}
			sr.Color = BLACK
			err = t.SaveNode(sr)
			if err != nil {
				return err
			}
			err = t.RotateLeft(s)
			if err != nil {
				return err
			}
		}
	}

	return t.deleteCase6(n)
}

// S is black, S's right child is red, and N is the left child of its parent P
func (t *RBTree) deleteCase6(n node) error {
	if DEBUG_TREE {
		logger.Debugf("deleteCase6: n %v", n)
	}
	if DEBUG_PRINT_TREE {
		t.PrintTree("", n)
	}
	s, err := t.Sibling(n)
	if err != nil {
		return err
	}
	p, err := t.Parent(n)
	if err != nil {
		return err
	}
	sl, err := t.GetNode(s.Left)
	if err != nil {
		return err
	}
	sr, err := t.GetNode(s.Right)
	if err != nil {
		return err
	}
	if s.Color == BLACK {
		if sr.Color == RED && p.Left == n.Key {
			s.Color = p.Color
			err = t.SaveNode(s)
			if err != nil {
				return err
			}
			p.Color = BLACK
			err = t.SaveNode(p)
			if err != nil {
				return err
			}
			sr.Color = BLACK
			err = t.SaveNode(sr)
			if err != nil {
				return err
			}
			err := t.RotateLeft(p)
			if err != nil {
				return err
			}
		} else if sl.Color == RED && p.Right == n.Key {
			s.Color = p.Color
			err = t.SaveNode(s)
			if err != nil {
				return err
			}
			p.Color = BLACK
			err = t.SaveNode(p)
			if err != nil {
				return err
			}
			sl.Color = BLACK
			err = t.SaveNode(sl)
			if err != nil {
				return err
			}
			err := t.RotateRight(p)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// get value of the key
func (t *RBTree) Get(key string) ([]byte, error) {
	n, err := t.GetNode(key)
	if err != nil {
		return nil, err
	}
	return n.Value, nil
}

// search smallest node n that n.key >= key
func (t *RBTree) Search(key string) (node, error) {
	r, err := t.GetRoot()
	if err != nil {
		return NULL, err
	}
	return t.searchRecurse(key, r, NULL)
}

// search next smallest node whose key is greater than current node n
func (t *RBTree) SearchNext(n node) (node, error) {
	if n.IsNull() {
		return t.Search("")
	}

	if len(n.Right) > 0 {
		right, err := t.GetNode(n.Right)
		if err != nil {
			return NULL, err
		}
		return t.searchRecurse(n.Key, right, NULL)
	} else if len(n.Parent) > 0 {
		return t.BiggerParent(n)
	} else {
		return NULL, nil
	}
}

//return the first parent whose key is greater than current node
func (t *RBTree) BiggerParent(n node) (node, error) {
	p, err := t.Parent(n)
	if err != nil {
		return NULL, err
	}
	for p.Key <= n.Key && !p.IsNull() {
		p, err = t.Parent(p)
		if err != nil {
			return NULL, err
		}
	}
	return p, nil
}

// search smallest node n that n.key >= key
// n is current node
// p is parent/ancester that p.key > key
func (t *RBTree) searchRecurse(key string, n node, p node) (node, error) {
	if n.IsNull() || n.Key == key {
		return n, nil
	}
	if key < n.Key {
		left, err := t.GetNode(n.Left)
		if err != nil {
			return NULL, err
		}
		if left.IsNull() {
			return n, nil
		} else {
			//current node becomes p; search left
			return t.searchRecurse(key, left, n)
		}
	} else {
		right, err := t.GetNode(n.Right)
		if err != nil {
			return NULL, err
		}
		if right.IsNull() {
			return p, nil
		} else {
			//keep current p; search right
			return t.searchRecurse(key, right, p)
		}
	}
}

func (t *RBTree) SaveToLedger() error {
	for k, d := range t.ToDelete {
		//logger.Debugf("Save to Ledger key:%v delete:%v", k, d)
		if d == true {
			err := t.Stub.DelState(k)
			logger.Debugf("Delete ledger key: %v %v", k, err)
			if err != nil {
				return errors.Wrapf(err, "Failed to delete %v from ledger", k)
			}
		} else {
			v, _ := t.Cache[k]
			err := t.Stub.PutState(k, v)
			logger.Debugf("Save Ledger key:%v %v", k, err)
			if err != nil {
				return errors.Wrapf(err, "Failed to put %v in ledger", k)
			}
		}
	}
	return nil
}

func (t *RBTree) PrintTree(filename string, currentNode interface{}) error {
	var f *os.File
	if len(filename) > 0 {
		var err error
		f, err = os.Create(filename)
		if err != nil {
			return err
		}
	}

	if DEBUG_TREE {
		logger.Debugf("-->printTree %v", currentNode)
	}

	cn := NULL
	if currentNode, ok := currentNode.(node); ok {
		cn = currentNode
	}
	if currentNode, ok := currentNode.(string); ok {
		cn = NULL
		cn.Key = currentNode
	}

	n, _ := t.GetRoot()
	if DEBUG_TREE {
		logger.Debugf("root  %v", n)
	}
	if n.Right != LEAF || n.Key == cn.Parent {
		r, _ := t.GetNode(n.Right)
		if r.IsNull() && n.Key == cn.Parent {
			r.Key = "NIL"
			r.Parent = n.Key
		}
		t.printTreeInternal(r, f, true, "", cn)
	}

	t.printNodeKey(n, f, true, "", cn.Key)

	if n.Left != LEAF || n.Key == cn.Parent {
		l, _ := t.GetNode(n.Left)
		if l.IsNull() && n.Key == cn.Parent {
			l.Key = "NIL"
			l.Parent = n.Key
		}
		t.printTreeInternal(l, f, false, "", cn)
	}

	return nil
}

func (t *RBTree) writeString(f *os.File, str string) {
	if f != nil {
		f.WriteString(str)
		f.Sync()
	}
	logger.Debug(str)
}

// print value of key of the node n. if n.key == k, it's current node
func (t *RBTree) printNodeKey(n node, f *os.File, isRoot bool, indent string, currentNodeKey string) error {
	key := indent
	if !isRoot {
		key = key + "----- "
	}

	key = key + n.Color + " " + n.Key
	if len(n.Key) == 0 {
		key = "B NIL"
	}

	if currentNodeKey == n.Key {
		key = "(" + key + ")"
	}
	t.writeString(f, key)
	return nil
}

func (t *RBTree) printTreeInternal(n node, f *os.File, isRight bool, indent string, currentNode node) error {
	if n.IsNull() {
		return nil
	}
	if len(currentNode.Key) == 0 {
		currentNode.Key = "NIL"
	}

	if n.Right != LEAF || n.Key == currentNode.Parent {
		ind := "        "
		if !isRight {
			ind = " |      "
		}
		r, _ := t.GetNode(n.Right)
		if r.IsNull() && n.Key == currentNode.Parent {
			r.Key = "NIL"
			r.Parent = n.Key
		}
		t.printTreeInternal(r, f, true, indent+ind, currentNode)
	}

	keyindent := indent
	if isRight {
		keyindent = keyindent + " /"
	} else {
		keyindent = keyindent + " \\"
	}

	t.printNodeKey(n, f, false, keyindent, currentNode.Key)

	if n.Left != LEAF || n.Key == currentNode.Parent {
		ind := " |      "
		if !isRight {
			ind = "        "
		}
		l, _ := t.GetNode(n.Left)
		if l.IsNull() && n.Key == currentNode.Parent {
			l.Key = "NIL"
			l.Parent = n.Key
		}
		t.printTreeInternal(l, f, false, indent+ind, currentNode)
	}
	return nil
}

type TreeIter struct {
	FirstKey    string //inclusive
	LastKey     string //exclusive
	NextNode    node
	CurrentNode node
	Tree        *RBTree
	Closed      bool
	Ascending   bool
}

func (tIter *TreeIter) HasNext() bool {
	return !tIter.NextNode.IsNull()
}

func (tIter *TreeIter) Next() (*queryresult.KV, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	if tIter.Closed {
		return nil, errors.New("Next() called after Closed()")
	}
	err := tIter.getNext()
	if err != nil {
		return nil, err
	}
	if tIter.CurrentNode.IsNull() {
		return nil, errors.New("Next() called when it does not HaveNext()")
	}
	v := tIter.CurrentNode.Value
	k := tIter.CurrentNode.Key
	return &queryresult.KV{Key: k, Value: v}, err
}

func (tIter *TreeIter) getNext() error {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	tIter.CurrentNode = tIter.NextNode
	if tIter.CurrentNode.IsNull() {
		return nil
	}
	n, err := tIter.Tree.SearchNext(tIter.CurrentNode)
	if n.Key < tIter.LastKey {
		tIter.NextNode = n
	} else {
		tIter.NextNode = NULL
	}
	return err
}

func (tIter *TreeIter) Close() error {
	tIter.CurrentNode = NULL
	tIter.NextNode = NULL
	tIter.Closed = true
	return nil
}

//startKey (inclusive) and endKey (exclusive)
func (t *RBTree) NewTreeIter(startKey string, endKey string, ascending bool) (*TreeIter, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	tr := TreeIter{}
	tr.FirstKey = startKey
	tr.LastKey = endKey
	tr.NextNode = NULL
	tr.CurrentNode = NULL
	tr.Closed = false
	tr.Tree = t
	tr.Ascending = ascending

	s, err := tr.Tree.Search(startKey)
	logger.Debugf("Start key: %v", startKey)
	logger.Debugf("End key: %v", endKey)
	logger.Debugf("First key: %v", s.Key)
	if s.Key >= endKey {
		tr.Closed = true
		return &tr, nil
	}
	if err != nil {
		tr.Closed = true
		return nil, err
	}
	tr.NextNode = s
	return &tr, err
}

func (t *RBTree) GetKeyByRange(startKey, endKey string) (shim.StateQueryIteratorInterface, error) {
	defer utils.ExitFnLogger(logger, utils.EnterFnLogger(logger))
	if len(startKey) == 0 {
		startKey = global.EMPTY_KEY_SUBSTITUDE
	}
	if err := t.validateSimpleKeys(startKey, endKey); err != nil {
		return nil, err
	}

	return t.NewTreeIter(startKey, endKey, true)
}

func (t *RBTree) createRangeKeysForPartialCompositeKey(objectType string, attributes []string) (string, string, error) {
	partialCompositeKey, err := t.Stub.CreateCompositeKey(objectType, attributes)
	if err != nil {
		return "", "", err
	}
	startKey := partialCompositeKey
	endKey := partialCompositeKey + string(global.MAX_UNICODE_RUNE_VALUE)

	return startKey, endKey, nil
}
func (t *RBTree) validateSimpleKeys(simpleKeys ...string) error {
	for _, key := range simpleKeys {
		if len(key) > 0 && key[0] == global.COMPOSITE_KEY_NAMESPACE[0] {
			return errors.Errorf(`first character of the key [%s] contains a null character which is not allowed`, key)
		}
	}
	return nil
}

func (t *RBTree) GetKeyByPartialCompositeKey(objectType string, keys []string) (shim.StateQueryIteratorInterface, error) {
	startKey, endKey, _ := t.createRangeKeysForPartialCompositeKey(objectType, keys)
	return t.NewTreeIter(startKey, endKey, true)
}

func NewRBTree(stub cached_stub.CachedStubInterface, treeName string) *RBTree {
	tr := RBTree{}
	tr.Stub = stub
	tr.Cache = make(map[string][]byte)
	tr.ToDelete = make(map[string]bool)
	tr.TreeName = treeName
	tr.Prefix = PREFIX + treeName + "_"
	tr.GetRoot()
	return &tr
}

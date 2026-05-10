// Package layout implements a binary-tree layout engine for the terminal UI.
// Supports split, mount, unmount, resize, swap, and fullscreen operations
// with delta-op generation and canonical hashing for sync.
package layout

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
)

// LayoutTree represents a node in the layout hierarchy.
type LayoutTree struct {
	ID        string       `json:"id"`
	Type      string       `json:"type"`       // "split" or "leaf"
	Direction string       `json:"direction"`  // "h" or "v"
	Content   string       `json:"content"`    // "xterm", "monaco", "welcome"
	Path      string       `json:"path"`       // file path for monaco
	PTYID     string       `json:"pty_id"`     // PTY session ID for xterm
	Children  []LayoutTree `json:"children"`
	Size      float64      `json:"size"`       // 0.0-1.0 relative to parent
	focused   bool         // internal focus state
}

// Op represents a layout mutation operation.
type Op struct {
	Op        string  `json:"op"`          // "split", "mount", "unmount", "resize", "swap", "fullscreen"
	TargetID  string  `json:"target_id"`
	Direction string  `json:"direction,omitempty"`
	Content   string  `json:"content,omitempty"`
	PTYID     string  `json:"pty_id,omitempty"`
	Size      float64 `json:"size,omitempty"`
}

// focusTracker maintains a single focused leaf across the tree.
var (
	focusMu sync.RWMutex
	focusID string
)

// NewRoot creates a root leaf node with the given content type.
func NewRoot(content string) *LayoutTree {
	return &LayoutTree{
		ID:      "root",
		Type:    "leaf",
		Content: content,
		Size:    1.0,
	}
}

// Apply executes a layout operation and returns delta operations for the client.
func (t *LayoutTree) Apply(op Op) ([]Op, error) {
	node := t.Find(op.TargetID)
	if node == nil && op.Op != "mount" {
		return nil, fmt.Errorf("layout: target %q not found", op.TargetID)
	}

	switch op.Op {
	case "split":
		return t.applySplit(op)
	case "mount":
		return t.applyMount(op)
	case "unmount":
		return t.applyUnmount(op)
	case "resize":
		return t.applyResize(op)
	case "swap":
		return t.applySwap(op)
	case "fullscreen":
		return t.applyFullscreen(op)
	default:
		return nil, fmt.Errorf("layout: unknown op %q", op.Op)
	}
}

// applySplit converts a leaf into a split with two children (original + new).
func (t *LayoutTree) applySplit(op Op) ([]Op, error) {
	node := t.Find(op.TargetID)
	if node == nil {
		return nil, fmt.Errorf("layout: target %q not found for split", op.TargetID)
	}
	if node.Type == "split" {
		return nil, fmt.Errorf("layout: cannot split a split node")
	}

	direction := op.Direction
	if direction == "" {
		direction = "h"
	}

	// Preserve original leaf's properties
	origContent := node.Content
	origPath := node.Path
	origPTYID := node.PTYID
	origFocused := node.focused

	// Convert to split
	node.Type = "split"
	node.Direction = direction
	node.Content = ""
	node.Path = ""
	node.PTYID = ""
	node.Children = []LayoutTree{
		{
			ID:      node.ID + "_left",
			Type:    "leaf",
			Content: origContent,
			Path:    origPath,
			PTYID:   origPTYID,
			Size:    0.5,
			focused: origFocused,
		},
		{
			ID:      node.ID + "_right",
			Type:    "leaf",
			Content: op.Content,
			PTYID:   op.PTYID,
			Size:    0.5,
		},
	}

	return []Op{
		{Op: "split", TargetID: node.ID, Direction: direction, Content: op.Content, PTYID: op.PTYID},
	}, nil
}

// applyMount adds a new leaf as a sibling of the target.
func (t *LayoutTree) applyMount(op Op) ([]Op, error) {
	if op.TargetID == "" {
		// Mount at root level — split root
		splitOp := Op{Op: "split", TargetID: t.ID, Direction: "h", Content: op.Content, PTYID: op.PTYID}
		return t.applySplit(splitOp)
	}

	node := t.Find(op.TargetID)
	if node == nil {
		return nil, fmt.Errorf("layout: target %q not found for mount", op.TargetID)
	}

	// Find parent and determine position
	parent, idx := t.findParent(t, op.TargetID)
	if parent == nil {
		// Target is root — split root
		splitOp := Op{Op: "split", TargetID: t.ID, Direction: "v", Content: op.Content, PTYID: op.PTYID}
		return t.applySplit(splitOp)
	}

	// Create new leaf
	newLeaf := LayoutTree{
		ID:      "mount_" + node.ID,
		Type:    "leaf",
		Content: op.Content,
		PTYID:   op.PTYID,
		Size:    0.5,
	}

	// Convert parent to split if it's a leaf
	if parent.Type == "leaf" {
		dir := "h"
		if parent.Direction != "" {
			dir = parent.Direction
		}
		parent.Children = []LayoutTree{
			{ID: parent.ID + "_left", Type: "leaf", Content: parent.Content, Path: parent.Path, PTYID: parent.PTYID, Size: 0.5},
			{ID: parent.ID + "_right", Type: "leaf", Content: op.Content, PTYID: op.PTYID, Size: 0.5},
		}
		parent.Type = "split"
		parent.Direction = dir
		parent.Content = ""
		parent.Path = ""
		parent.PTYID = ""
		return []Op{
			{Op: "mount", TargetID: op.TargetID, Content: op.Content, PTYID: op.PTYID},
		}, nil
	}

	// Parent is already a split — insert new leaf after target's position
	newSize := 1.0 / float64(len(parent.Children)+1)
	for i := range parent.Children {
		parent.Children[i].Size = newSize
	}
	parent.Children = append(parent.Children[:idx+1], append([]LayoutTree{newLeaf}, parent.Children[idx+1:]...)...)
	newLeaf.Size = newSize

	return []Op{
		{Op: "mount", TargetID: op.TargetID, Content: op.Content, PTYID: op.PTYID},
	}, nil
}

// findParent returns the parent node and the index of the child with the given ID.
func (t *LayoutTree) findParent(current *LayoutTree, id string) (*LayoutTree, int) {
	for i, child := range current.Children {
		if child.ID == id {
			return current, i
		}
		if child.Type == "split" {
			// Need pointer to child in slice
			p, idx := t.findParentInSlice(&current.Children[i], id)
			if p != nil {
				return p, idx
			}
		}
	}
	return nil, -1
}

func (t *LayoutTree) findParentInSlice(node *LayoutTree, id string) (*LayoutTree, int) {
	for i, child := range node.Children {
		if child.ID == id {
			return node, i
		}
		if child.Type == "split" {
			p, idx := t.findParentInSlice(&node.Children[i], id)
			if p != nil {
				return p, idx
			}
		}
	}
	return nil, -1
}

// applyUnmount removes a leaf and merges parent if only one child remains.
func (t *LayoutTree) applyUnmount(op Op) ([]Op, error) {
	if op.TargetID == t.ID {
		return nil, fmt.Errorf("layout: cannot unmount root")
	}

	parent, idx := t.findParent(t, op.TargetID)
	if parent == nil {
		return nil, fmt.Errorf("layout: target %q not found for unmount", op.TargetID)
	}

	// Remove the child
	parent.Children = append(parent.Children[:idx], parent.Children[idx+1:]...)

	// If parent now has only one child, merge it up
	if len(parent.Children) == 1 {
		only := parent.Children[0]
		parent.Type = only.Type
		parent.Direction = only.Direction
		parent.Content = only.Content
		parent.Path = only.Path
		parent.PTYID = only.PTYID
		parent.Children = only.Children
		parent.Size = only.Size
	} else {
		// Recalculate sizes
		newSize := 1.0 / float64(len(parent.Children))
		for i := range parent.Children {
			parent.Children[i].Size = newSize
		}
	}

	return []Op{
		{Op: "unmount", TargetID: op.TargetID},
	}, nil
}

// applyResize adjusts the Size field on the target node.
func (t *LayoutTree) applyResize(op Op) ([]Op, error) {
	node := t.Find(op.TargetID)
	if node == nil {
		return nil, fmt.Errorf("layout: target %q not found for resize", op.TargetID)
	}

	parent, _ := t.findParent(t, op.TargetID)
	if parent != nil {
		parentSize := 1.0
		if parentNode := t.findParentOfNode(t, op.TargetID); parentNode != nil {
			parentSize = parentNode.Size
		}

		// Set target size and redistribute among siblings
		targetSize := op.Size
		if targetSize <= 0 {
			targetSize = 0.1
		}
		if targetSize > 1.0 {
			targetSize = 1.0
		}

		remaining := parentSize - targetSize
		if remaining < 0 {
			remaining = 0
		}
		siblingCount := 0
		for _, child := range parent.Children {
			if child.ID != op.TargetID {
				siblingCount++
			}
		}
		siblingSize := remaining / float64(siblingCount)
		if siblingSize < 0 {
			siblingSize = 0
		}

		for i := range parent.Children {
			if parent.Children[i].ID == op.TargetID {
				parent.Children[i].Size = targetSize
			} else {
				parent.Children[i].Size = siblingSize
			}
		}
	}

	node.Size = op.Size
	return []Op{
		{Op: "resize", TargetID: op.TargetID, Size: op.Size},
	}, nil
}

func (t *LayoutTree) findParentOfNode(current *LayoutTree, id string) *LayoutTree {
	for _, child := range current.Children {
		if child.ID == id {
			return current
		}
		if child.Type == "split" {
			if p := t.findParentOfNode(&current.Children[0], id); p != nil {
				return p
			}
		}
	}
	return nil
}

// applySwap toggles the Direction on a split node (h↔v).
func (t *LayoutTree) applySwap(op Op) ([]Op, error) {
	node := t.Find(op.TargetID)
	if node == nil {
		return nil, fmt.Errorf("layout: target %q not found for swap", op.TargetID)
	}
	if node.Type != "split" {
		return nil, fmt.Errorf("layout: cannot swap a leaf node")
	}

	if node.Direction == "h" {
		node.Direction = "v"
	} else {
		node.Direction = "h"
	}

	return []Op{
		{Op: "swap", TargetID: op.TargetID, Direction: node.Direction},
	}, nil
}

// applyFullscreen hides all siblings, showing only the target.
// For now, this is a logical operation — the client handles rendering.
func (t *LayoutTree) applyFullscreen(op Op) ([]Op, error) {
	node := t.Find(op.TargetID)
	if node == nil {
		return nil, fmt.Errorf("layout: target %q not found for fullscreen", op.TargetID)
	}

	return []Op{
		{Op: "fullscreen", TargetID: op.TargetID},
	}, nil
}

// Hash returns a SHA-256 hex digest of the canonical JSON representation.
func (t *LayoutTree) Hash() string {
	data := t.ToJSON()
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// Find returns the node with the given ID, or nil if not found.
func (t *LayoutTree) Find(id string) *LayoutTree {
	if t.ID == id {
		return t
	}
	for i := range t.Children {
		if found := t.Children[i].Find(id); found != nil {
			return found
		}
	}
	return nil
}

// FocusLeaf returns the ID of the currently focused leaf node.
func (t *LayoutTree) FocusLeaf() string {
	focusMu.RLock()
	defer focusMu.RUnlock()

	if focusID != "" {
		node := t.Find(focusID)
		if node != nil && node.Type == "leaf" {
			return focusID
		}
	}

	// Fallback: find first leaf
	return t.firstLeaf()
}

func (t *LayoutTree) firstLeaf() string {
	if t.Type == "leaf" {
		return t.ID
	}
	for i := range t.Children {
		if id := t.Children[i].firstLeaf(); id != "" {
			return id
		}
	}
	return ""
}

// SetFocus sets the focused leaf node by ID.
func (t *LayoutTree) SetFocus(id string) {
	focusMu.Lock()
	defer focusMu.Unlock()

	// Clear previous focus
	if focusID != "" {
		prev := t.Find(focusID)
		if prev != nil {
			prev.focused = false
		}
	}

	focusID = id
	node := t.Find(id)
	if node != nil {
		node.focused = true
	}
}

// ToJSON returns canonical JSON for the tree (sorted keys, no extra whitespace).
func (t *LayoutTree) ToJSON() []byte {
	data, _ := json.Marshal(t)
	return data
}

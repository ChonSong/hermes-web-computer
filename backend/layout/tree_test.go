package layout

import (
	"encoding/json"
	"testing"
)

func TestNewRoot(t *testing.T) {
	root := NewRoot("xterm")
	if root.ID != "root" {
		t.Errorf("expected ID 'root', got %q", root.ID)
	}
	if root.Type != "leaf" {
		t.Errorf("expected Type 'leaf', got %q", root.Type)
	}
	if root.Content != "xterm" {
		t.Errorf("expected Content 'xterm', got %q", root.Content)
	}
	if root.Size != 1.0 {
		t.Errorf("expected Size 1.0, got %f", root.Size)
	}
}

func TestFind(t *testing.T) {
	root := NewRoot("welcome")

	// Find root
	node := root.Find("root")
	if node == nil {
		t.Fatal("expected to find root node")
	}
	if node.ID != "root" {
		t.Errorf("expected root ID, got %q", node.ID)
	}

	// Find non-existent
	node = root.Find("nonexistent")
	if node != nil {
		t.Error("expected nil for non-existent node")
	}
}

func TestApplySplit(t *testing.T) {
	root := NewRoot("xterm")
	root.ID = "root"

	// Split root horizontally
	ops, err := root.Apply(Op{Op: "split", TargetID: "root", Direction: "h", Content: "monaco"})
	if err != nil {
		t.Fatalf("split failed: %v", err)
	}
	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(ops))
	}
	if ops[0].Op != "split" {
		t.Errorf("expected op 'split', got %q", ops[0].Op)
	}

	// Verify structure
	if root.Type != "split" {
		t.Errorf("expected root type 'split', got %q", root.Type)
	}
	if root.Direction != "h" {
		t.Errorf("expected direction 'h', got %q", root.Direction)
	}
	if len(root.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(root.Children))
	}

	// Check left child (original)
	left := root.Children[0]
	if left.ID != "root_left" {
		t.Errorf("expected left child ID 'root_left', got %q", left.ID)
	}
	if left.Content != "xterm" {
		t.Errorf("expected left content 'xterm', got %q", left.Content)
	}
	if left.Size != 0.5 {
		t.Errorf("expected left size 0.5, got %f", left.Size)
	}

	// Check right child (new)
	right := root.Children[1]
	if right.ID != "root_right" {
		t.Errorf("expected right child ID 'root_right', got %q", right.ID)
	}
	if right.Content != "monaco" {
		t.Errorf("expected right content 'monaco', got %q", right.Content)
	}
	if right.Size != 0.5 {
		t.Errorf("expected right size 0.5, got %f", right.Size)
	}

	// Cannot split a split node
	_, err = root.Apply(Op{Op: "split", TargetID: "root", Direction: "v", Content: "browser"})
	if err == nil {
		t.Error("expected error when splitting a split node")
	}
}

func TestApplySplitVertical(t *testing.T) {
	root := NewRoot("xterm")
	root.ID = "root"

	ops, err := root.Apply(Op{Op: "split", TargetID: "root", Direction: "v", Content: "browser"})
	if err != nil {
		t.Fatalf("split failed: %v", err)
	}
	if root.Direction != "v" {
		t.Errorf("expected direction 'v', got %q", root.Direction)
	}
	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(ops))
	}
}

func TestApplyMount(t *testing.T) {
	root := NewRoot("xterm")
	root.ID = "root"

	// Mount at root level - internally splits root
	ops, err := root.Apply(Op{Op: "mount", TargetID: "root", Content: "monaco"})
	if err != nil {
		t.Fatalf("mount failed: %v", err)
	}
	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(ops))
	}
	// Mount at root level returns "split" since it converts root to split
	if ops[0].Op != "split" {
		t.Errorf("expected op 'split' (mount at root converts to split), got %q", ops[0].Op)
	}

	// Root should now be a split
	if root.Type != "split" {
		t.Errorf("expected type 'split', got %q", root.Type)
	}
	if len(root.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(root.Children))
	}
}

func TestApplyMountIntoSplit(t *testing.T) {
	root := NewRoot("xterm")
	root.ID = "root"

	// First split root
	_, err := root.Apply(Op{Op: "split", TargetID: "root", Direction: "h", Content: "monaco"})
	if err != nil {
		t.Fatalf("split failed: %v", err)
	}

	// Mount into left child — mounts AFTER root_left in parent's children
	ops, err := root.Apply(Op{Op: "mount", TargetID: "root_left", Content: "browser"})
	if err != nil {
		t.Fatalf("mount failed: %v", err)
	}
	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(ops))
	}

	// left child remains a leaf; parent (root) gets new child inserted after root_left
	left := root.Find("root_left")
	if left.Type != "leaf" {
		t.Errorf("expected left child to remain leaf, got %q", left.Type)
	}
	// root now has 3 children: [root_left, mount_root_left, root_right]
	if len(root.Children) != 3 {
		t.Errorf("expected root to have 3 children, got %d", len(root.Children))
	}
}

func TestApplyUnmount(t *testing.T) {
	root := NewRoot("xterm")
	root.ID = "root"

	// Split root
	_, err := root.Apply(Op{Op: "split", TargetID: "root", Direction: "h", Content: "monaco"})
	if err != nil {
		t.Fatalf("split failed: %v", err)
	}

	// Unmount right child
	ops, err := root.Apply(Op{Op: "unmount", TargetID: "root_right"})
	if err != nil {
		t.Fatalf("unmount failed: %v", err)
	}
	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(ops))
	}
	if ops[0].Op != "unmount" {
		t.Errorf("expected op 'unmount', got %q", ops[0].Op)
	}

	// Should merge back to single leaf
	if root.Type != "leaf" {
		t.Errorf("expected root type 'leaf', got %q", root.Type)
	}
	if root.Content != "xterm" {
		t.Errorf("expected content 'xterm', got %q", root.Content)
	}
	if len(root.Children) != 0 {
		t.Errorf("expected 0 children, got %d", len(root.Children))
	}

	// Cannot unmount root
	_, err = root.Apply(Op{Op: "unmount", TargetID: "root"})
	if err == nil {
		t.Error("expected error when unmounting root")
	}
}

func TestApplyUnmountPreservesSibling(t *testing.T) {
	root := NewRoot("xterm")
	root.ID = "root"

	// Create structure via mount
	_, _ = root.Apply(Op{Op: "split", TargetID: "root", Direction: "h", Content: "monaco"})
	_, _ = root.Apply(Op{Op: "mount", TargetID: "root_right", Content: "browser"})
	_, _ = root.Apply(Op{Op: "mount", TargetID: "root_right", Content: "welcome"})

	// After 2 mounts, root has 4 children (mount inserts after target)
	// Due to the way mount works, we get: [root_left, root_right, mount_root_right, mount_root_right]
	if len(root.Children) != 4 {
		t.Fatalf("expected 4 children, got %d", len(root.Children))
	}

	// Unmount one of the mount_* nodes
	_, err := root.Apply(Op{Op: "unmount", TargetID: "mount_root_right"})
	if err != nil {
		t.Fatalf("unmount failed: %v", err)
	}

	// Should have 3 children now, with recalculated sizes
	if len(root.Children) != 3 {
		t.Errorf("expected 3 children after unmount, got %d", len(root.Children))
	}
}

func TestApplyResize(t *testing.T) {
	root := NewRoot("xterm")
	root.ID = "root"

	// Split root
	_, err := root.Apply(Op{Op: "split", TargetID: "root", Direction: "h", Content: "monaco"})
	if err != nil {
		t.Fatalf("split failed: %v", err)
	}

	// Resize left child
	ops, err := root.Apply(Op{Op: "resize", TargetID: "root_left", Size: 0.7})
	if err != nil {
		t.Fatalf("resize failed: %v", err)
	}
	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(ops))
	}
	if ops[0].Op != "resize" {
		t.Errorf("expected op 'resize', got %q", ops[0].Op)
	}
	if ops[0].Size != 0.7 {
		t.Errorf("expected size 0.7 in op, got %f", ops[0].Size)
	}

	// Check sizes
	left := root.Find("root_left")
	right := root.Find("root_right")
	if left.Size != 0.7 {
		t.Errorf("expected left size 0.7, got %f", left.Size)
	}
	// Right should get remaining space
	if right.Size < 0.29 || right.Size > 0.31 {
		t.Errorf("expected right size ~0.3, got %f", right.Size)
	}

	// Resize clamps values
	_, err = root.Apply(Op{Op: "resize", TargetID: "root_left", Size: 2.0})
	if err != nil {
		t.Fatalf("resize with large value failed: %v", err)
	}
	_, err = root.Apply(Op{Op: "resize", TargetID: "root_left", Size: -0.5})
	if err != nil {
		t.Fatalf("resize with negative value failed: %v", err)
	}
}

func TestApplyResizeNonExistent(t *testing.T) {
	root := NewRoot("xterm")
	root.ID = "root"

	_, err := root.Apply(Op{Op: "resize", TargetID: "nonexistent", Size: 0.5})
	if err == nil {
		t.Error("expected error for resize non-existent node")
	}
}

func TestApplySwap(t *testing.T) {
	root := NewRoot("xterm")
	root.ID = "root"

	// Split root
	_, err := root.Apply(Op{Op: "split", TargetID: "root", Direction: "h", Content: "monaco"})
	if err != nil {
		t.Fatalf("split failed: %v", err)
	}

	// Swap direction
	ops, err := root.Apply(Op{Op: "swap", TargetID: "root"})
	if err != nil {
		t.Fatalf("swap failed: %v", err)
	}
	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(ops))
	}
	if ops[0].Op != "swap" {
		t.Errorf("expected op 'swap', got %q", ops[0].Op)
	}
	if root.Direction != "v" {
		t.Errorf("expected direction 'v' after swap, got %q", root.Direction)
	}

	// Swap back
	_, err = root.Apply(Op{Op: "swap", TargetID: "root"})
	if err != nil {
		t.Fatalf("swap failed: %v", err)
	}
	if root.Direction != "h" {
		t.Errorf("expected direction 'h' after second swap, got %q", root.Direction)
	}

	// Cannot swap leaf
	_, err = root.Apply(Op{Op: "swap", TargetID: "root_left"})
	if err == nil {
		t.Error("expected error when swapping leaf node")
	}
}

func TestApplyFullscreen(t *testing.T) {
	root := NewRoot("xterm")
	root.ID = "root"

	// Split root
	_, err := root.Apply(Op{Op: "split", TargetID: "root", Direction: "h", Content: "monaco"})
	if err != nil {
		t.Fatalf("split failed: %v", err)
	}

	// Fullscreen left child
	ops, err := root.Apply(Op{Op: "fullscreen", TargetID: "root_left"})
	if err != nil {
		t.Fatalf("fullscreen failed: %v", err)
	}
	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(ops))
	}
	if ops[0].Op != "fullscreen" {
		t.Errorf("expected op 'fullscreen', got %q", ops[0].Op)
	}
	if ops[0].TargetID != "root_left" {
		t.Errorf("expected target 'root_left', got %q", ops[0].TargetID)
	}

	// Fullscreen non-existent node
	_, err = root.Apply(Op{Op: "fullscreen", TargetID: "nonexistent"})
	if err == nil {
		t.Error("expected error for fullscreen non-existent node")
	}
}

func TestApplyUnknownOp(t *testing.T) {
	root := NewRoot("xterm")
	root.ID = "root"

	_, err := root.Apply(Op{Op: "unknown"})
	if err == nil {
		t.Error("expected error for unknown op")
	}
}

func TestFocusLeaf(t *testing.T) {
	root := NewRoot("xterm")
	root.ID = "root"

	// Initially returns first leaf
	id := root.FocusLeaf()
	if id != "root" {
		t.Errorf("expected initial focus 'root', got %q", id)
	}

	// Split and check focus
	_, _ = root.Apply(Op{Op: "split", TargetID: "root", Direction: "h", Content: "monaco"})

	// Focus on right child
	root.SetFocus("root_right")
	id = root.FocusLeaf()
	if id != "root_right" {
		t.Errorf("expected focus 'root_right', got %q", id)
	}

	// Focus on left child
	root.SetFocus("root_left")
	id = root.FocusLeaf()
	if id != "root_left" {
		t.Errorf("expected focus 'root_left', got %q", id)
	}
}

func TestHash(t *testing.T) {
	root1 := NewRoot("xterm")
	root1.ID = "root"

	root2 := NewRoot("xterm")
	root2.ID = "root"

	// Same structure should produce same hash
	h1 := root1.Hash()
	h2 := root2.Hash()
	if h1 != h2 {
		t.Errorf("expected same hash for identical trees, got %s vs %s", h1, h2)
	}

	// Different content should produce different hash
	root2.Content = "monaco"
	h2 = root2.Hash()
	if h1 == h2 {
		t.Error("expected different hash for different content")
	}
}

func TestToJSON(t *testing.T) {
	root := NewRoot("xterm")
	root.ID = "root"

	data := root.ToJSON()
	if len(data) == 0 {
		t.Error("expected non-empty JSON")
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	err := json.Unmarshal(data, &parsed)
	if err != nil {
		t.Errorf("invalid JSON: %v", err)
	}

	// Check fields
	if parsed["id"] != "root" {
		t.Errorf("expected id 'root', got %v", parsed["id"])
	}
	if parsed["type"] != "leaf" {
		t.Errorf("expected type 'leaf', got %v", parsed["type"])
	}
	if parsed["content"] != "xterm" {
		t.Errorf("expected content 'xterm', got %v", parsed["content"])
	}
}

func TestToJSONAfterSplit(t *testing.T) {
	root := NewRoot("xterm")
	root.ID = "root"

	_, _ = root.Apply(Op{Op: "split", TargetID: "root", Direction: "h", Content: "monaco"})

	data := root.ToJSON()
	var parsed map[string]interface{}
	err := json.Unmarshal(data, &parsed)
	if err != nil {
		t.Errorf("invalid JSON: %v", err)
	}

	if parsed["type"] != "split" {
		t.Errorf("expected type 'split', got %v", parsed["type"])
	}

	children, ok := parsed["children"].([]interface{})
	if !ok {
		t.Fatal("expected children array")
	}
	if len(children) != 2 {
		t.Errorf("expected 2 children, got %d", len(children))
	}
}

func TestApplyMountEmptyTarget(t *testing.T) {
	root := NewRoot("xterm")
	root.ID = "root"

	// Mount with empty target should split root
	ops, err := root.Apply(Op{Op: "mount", TargetID: "", Content: "monaco"})
	if err != nil {
		t.Fatalf("mount failed: %v", err)
	}
	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(ops))
	}
	if root.Type != "split" {
		t.Errorf("expected root type 'split', got %q", root.Type)
	}
}

func TestUnmountLastChild(t *testing.T) {
	root := NewRoot("xterm")
	root.ID = "root"

	// Create 2-way split
	_, _ = root.Apply(Op{Op: "split", TargetID: "root", Direction: "h", Content: "monaco"})

	// Unmount one child
	_, err := root.Apply(Op{Op: "unmount", TargetID: "root_left"})
	if err != nil {
		t.Fatalf("unmount failed: %v", err)
	}

	// Should still have one child (root_right becomes merged into root)
	// The root becomes the remaining child
	if root.Type != "leaf" {
		t.Errorf("expected root to be leaf after unmount last child, got %q", root.Type)
	}
}

func TestDeepNesting(t *testing.T) {
	root := NewRoot("xterm")
	root.ID = "root"

	// Create deep nesting
	_, _ = root.Apply(Op{Op: "split", TargetID: "root", Direction: "h", Content: "monaco"})
	_, _ = root.Apply(Op{Op: "split", TargetID: "root_left", Direction: "v", Content: "browser"})
	_, _ = root.Apply(Op{Op: "split", TargetID: "root_left_left", Direction: "h", Content: "welcome"})

	// Find deepest node
	deep := root.Find("root_left_left_left")
	if deep == nil {
		t.Error("expected to find deeply nested node")
	}

	// Verify structure
	left := root.Find("root_left")
	if left.Type != "split" {
		t.Errorf("expected root_left to be split, got %q", left.Type)
	}
	if left.Direction != "v" {
		t.Errorf("expected root_left direction 'v', got %q", left.Direction)
	}
}

func TestApplySplitDefaultDirection(t *testing.T) {
	root := NewRoot("xterm")
	root.ID = "root"

	// Split without specifying direction
	_, err := root.Apply(Op{Op: "split", TargetID: "root", Content: "monaco"})
	if err != nil {
		t.Fatalf("split failed: %v", err)
	}

	// Should default to horizontal
	if root.Direction != "h" {
		t.Errorf("expected default direction 'h', got %q", root.Direction)
	}
}
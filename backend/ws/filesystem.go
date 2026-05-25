package ws

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// allowedRoot is the base directory for all filesystem operations.
// Defaults to the repo root. Override via HERMES_HWC_ROOT env var for testing.
// Lazily initialized so TestMain can set HERMES_HWC_ROOT before first use.
var allowedRoot string

func getAllowedRoot() string {
	if root := os.Getenv("HERMES_HWC_ROOT"); root != "" {
		return root
	}
	// Fallback to cwd/repo root — don't hardcode /opt/data/
	cwd, _ := os.Getwd()
	if cwd != "" {
		return cwd
	}
	return "/home/hermeswebui/.hermes/hermes-web-computer"
}

func allowedRootLazy() string {
	if allowedRoot == "" {
		allowedRoot = getAllowedRoot()
	}
	return allowedRoot
}

// sanitizePath ensures the requested path doesn't escape the allowed root.
func sanitizePath(reqPath string) (string, error) {
	if reqPath == "" || reqPath == "/" {
		return allowedRootLazy(), nil
	}
	clean := filepath.Clean(reqPath)
	// Remove leading slash and join with allowed root
	clean = strings.TrimPrefix(clean, "/")
	if clean == "" {
		return allowedRootLazy(), nil
	}
	clean = filepath.Join(allowedRootLazy(), clean)
	resolved, err := filepath.EvalSymlinks(clean)
	if err != nil {
		if !strings.HasPrefix(filepath.Clean(clean), filepath.Clean(allowedRootLazy())) {
			return "", fmt.Errorf("path escapes allowed root")
		}
		return clean, nil
	}
	if !strings.HasPrefix(resolved, filepath.Clean(allowedRootLazy())) {
		return "", fmt.Errorf("path escapes allowed root")
	}
	return resolved, nil
}

type fsListEntry struct {
	Name    string `json:"name"`
	Type    string `json:"type"` // "file" | "dir" | "symlink"
	Size    int64  `json:"size"`
	ModTime string `json:"mod_time"`
}

func (m *Multiplexer) handleFSList(sess *Session, params json.RawMessage) {
	var p struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		sess.Send(Event{Protocol: "ui", Event: "fs.error", Data: mustMarshal(map[string]string{"message": err.Error()})})
		return
	}

	cleanPath, err := sanitizePath(p.Path)
	if err != nil {
		sess.Send(Event{Protocol: "ui", Event: "fs.error", Data: mustMarshal(map[string]interface{}{"message": err.Error(), "path": p.Path})})
		return
	}

	entries, err := os.ReadDir(cleanPath)
	if err != nil {
		sess.Send(Event{Protocol: "ui", Event: "fs.error", Data: mustMarshal(map[string]interface{}{"message": err.Error(), "path": cleanPath})})
		return
	}

	result := make([]fsListEntry, 0, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		entryType := "file"
		if e.IsDir() {
			entryType = "dir"
		}
		if info.Mode()&os.ModeSymlink != 0 {
			entryType = "symlink"
		}
		result = append(result, fsListEntry{
			Name:    e.Name(),
			Type:    entryType,
			Size:    info.Size(),
			ModTime: info.ModTime().Format(time.RFC3339),
		})
	}

	sess.Send(Event{Protocol: "ui", Event: "fs.list.response", Data: mustMarshal(map[string]interface{}{"path": cleanPath, "entries": result})})
}

func (m *Multiplexer) handleFSRead(sess *Session, params json.RawMessage) {
	var p struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		sess.Send(Event{Protocol: "ui", Event: "fs.error", Data: mustMarshal(map[string]string{"message": err.Error()})})
		return
	}

	cleanPath, err := sanitizePath(p.Path)
	if err != nil {
		sess.Send(Event{Protocol: "ui", Event: "fs.error", Data: mustMarshal(map[string]interface{}{"message": err.Error(), "path": p.Path})})
		return
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		sess.Send(Event{Protocol: "ui", Event: "fs.error", Data: mustMarshal(map[string]interface{}{"message": err.Error(), "path": cleanPath})})
		return
	}

	isBinary := false
	for _, b := range data {
		if b < 32 && b != 9 && b != 10 && b != 13 {
			isBinary = true
			break
		}
	}

	if isBinary {
		encoded := base64.StdEncoding.EncodeToString(data)
		sess.Send(Event{Protocol: "ui", Event: "fs.read.response", Data: mustMarshal(map[string]interface{}{
			"path":     cleanPath,
			"content":  encoded,
			"encoding": "base64",
			"size":     len(data),
		})})
	} else {
		sess.Send(Event{Protocol: "ui", Event: "fs.read.response", Data: mustMarshal(map[string]interface{}{
			"path":     cleanPath,
			"content":  string(data),
			"encoding": "utf8",
			"size":     len(data),
		})})
	}
}

func (m *Multiplexer) handleFSWrite(sess *Session, params json.RawMessage) {
	var p struct {
		Path     string `json:"path"`
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		sess.Send(Event{Protocol: "ui", Event: "fs.error", Data: mustMarshal(map[string]string{"message": err.Error()})})
		return
	}

	cleanPath, err := sanitizePath(p.Path)
	if err != nil {
		sess.Send(Event{Protocol: "ui", Event: "fs.error", Data: mustMarshal(map[string]interface{}{"message": err.Error(), "path": p.Path})})
		return
	}

	var data []byte
	if p.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(p.Content)
		if err != nil {
			sess.Send(Event{Protocol: "ui", Event: "fs.error", Data: mustMarshal(map[string]string{"message": "base64 decode error: " + err.Error()})})
			return
		}
		data = decoded
	} else {
		data = []byte(p.Content)
	}

	if err := os.MkdirAll(filepath.Dir(cleanPath), 0755); err != nil {
		sess.Send(Event{Protocol: "ui", Event: "fs.error", Data: mustMarshal(map[string]string{"message": err.Error()})})
		return
	}

	if err := os.WriteFile(cleanPath, data, 0644); err != nil {
		sess.Send(Event{Protocol: "ui", Event: "fs.error", Data: mustMarshal(map[string]string{"message": err.Error()})})
		return
	}

	sess.Send(Event{Protocol: "ui", Event: "fs.write.response", Data: mustMarshal(map[string]interface{}{
		"path":         cleanPath,
		"bytes_written": len(data),
	})})
}

func (m *Multiplexer) handleFSStat(sess *Session, params json.RawMessage) {
	var p struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		sess.Send(Event{Protocol: "ui", Event: "fs.error", Data: mustMarshal(map[string]string{"message": err.Error()})})
		return
	}

	cleanPath, err := sanitizePath(p.Path)
	if err != nil {
		sess.Send(Event{Protocol: "ui", Event: "fs.error", Data: mustMarshal(map[string]interface{}{"message": err.Error(), "path": p.Path})})
		return
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			sess.Send(Event{Protocol: "ui", Event: "fs.stat.response", Data: mustMarshal(map[string]interface{}{
				"path":   cleanPath,
				"exists": false,
			})})
			return
		}
		sess.Send(Event{Protocol: "ui", Event: "fs.error", Data: mustMarshal(map[string]interface{}{"message": err.Error(), "path": cleanPath})})
		return
	}

	fileType := "file"
	if info.IsDir() {
		fileType = "dir"
	}

	sess.Send(Event{Protocol: "ui", Event: "fs.stat.response", Data: mustMarshal(map[string]interface{}{
		"path":    cleanPath,
		"exists":  true,
		"type":    fileType,
		"size":    info.Size(),
		"mod_time": info.ModTime().Format(time.RFC3339),
		"is_dir":  info.IsDir(),
	})})
}

func (m *Multiplexer) handleFSRename(sess *Session, params json.RawMessage) {
	var p struct {
		OldPath string `json:"old_path"`
		NewPath string `json:"new_path"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		sess.Send(Event{Protocol: "ui", Event: "fs.error", Data: mustMarshal(map[string]string{"message": err.Error()})})
		return
	}

	cleanOld, err := sanitizePath(p.OldPath)
	if err != nil {
		sess.Send(Event{Protocol: "ui", Event: "fs.error", Data: mustMarshal(map[string]interface{}{"message": err.Error(), "path": p.OldPath})})
		return
	}

	cleanNew, err := sanitizePath(p.NewPath)
	if err != nil {
		sess.Send(Event{Protocol: "ui", Event: "fs.error", Data: mustMarshal(map[string]interface{}{"message": err.Error(), "path": p.NewPath})})
		return
	}

	if err := os.Rename(cleanOld, cleanNew); err != nil {
		sess.Send(Event{Protocol: "ui", Event: "fs.error", Data: mustMarshal(map[string]string{"message": err.Error()})})
		return
	}

	sess.Send(Event{Protocol: "ui", Event: "fs.rename.success", Data: mustMarshal(map[string]interface{}{
		"old_path": cleanOld,
		"new_path": cleanNew,
	})})
}

func (m *Multiplexer) handleFSDelete(sess *Session, params json.RawMessage) {
	var p struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		sess.Send(Event{Protocol: "ui", Event: "fs.error", Data: mustMarshal(map[string]string{"message": err.Error()})})
		return
	}

	cleanPath, err := sanitizePath(p.Path)
	if err != nil {
		sess.Send(Event{Protocol: "ui", Event: "fs.error", Data: mustMarshal(map[string]interface{}{"message": err.Error(), "path": p.Path})})
		return
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			sess.Send(Event{Protocol: "ui", Event: "fs.delete.error", Data: mustMarshal(map[string]interface{}{"message": "path does not exist", "path": cleanPath})})
			return
		}
		sess.Send(Event{Protocol: "ui", Event: "fs.error", Data: mustMarshal(map[string]interface{}{"message": err.Error(), "path": cleanPath})})
		return
	}

	if info.IsDir() {
		if err := os.RemoveAll(cleanPath); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "fs.delete.error", Data: mustMarshal(map[string]string{"message": err.Error()})})
			return
		}
	} else {
		if err := os.Remove(cleanPath); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "fs.delete.error", Data: mustMarshal(map[string]string{"message": err.Error()})})
			return
		}
	}

	sess.Send(Event{Protocol: "ui", Event: "fs.delete.success", Data: mustMarshal(map[string]interface{}{
		"path": cleanPath,
	})})
}

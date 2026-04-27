// Copyright 2025 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/alibaba/opensandbox/execd/pkg/log"
	"github.com/gin-gonic/gin"
)

type BrowserController struct {
	ctx      *gin.Context
	mu       sync.RWMutex
	browsers map[string]*BrowserSession
}

type BrowserSession struct {
	PID       int
	Port      int
	DataDir   string
	CreatedAt time.Time
}

func NewBrowserController(ctx *gin.Context) *BrowserController {
	return &BrowserController{
		ctx:      ctx,
		browsers: make(map[string]*BrowserSession),
	}
}

// isOverlayFSEnabled checks if OverlayFS snapshots are enabled
func isOverlayFSEnabled() bool {
	return os.Getenv("ENABLE_OVERLAYFS_SNAPSHOTS") == "true"
}

// CreateBrowser creates a new browser session
func (c *BrowserController) CreateBrowser() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Launch browser
	cmd := exec.Command("/opt/opensandbox/browser-launch.sh", "/tmp/browser-"+strconv.FormatInt(time.Now().UnixNano(), 10))
	if err := cmd.Start(); err != nil {
		log.Error("Failed to start browser: %v", err)
		c.ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Wait for CDP port to be ready (simplified implementation, needs polling in production)
	time.Sleep(2 * time.Second)

	// Assume port 9222 (needs to be parsed from browser output in production)
	port := 9222
	session := &BrowserSession{
		PID:       cmd.Process.Pid,
		Port:      port,
		DataDir:   "/tmp/browser-" + strconv.FormatInt(time.Now().UnixNano(), 10),
		CreatedAt: time.Now(),
	}

	sessionID := strconv.Itoa(port)
	c.browsers[sessionID] = session

	log.Info("Browser session created: id=%s, pid=%d, port=%d", sessionID, session.PID, session.Port)

	c.ctx.JSON(200, gin.H{
		"sessionId": sessionID,
		"cdpPort":   port,
	})
}

// KillBrowser kills a browser session
func (c *BrowserController) KillBrowser() {
	sessionID := c.ctx.Param("sessionId")

	c.mu.Lock()
	defer c.mu.Unlock()

	session, exists := c.browsers[sessionID]
	if !exists {
		c.ctx.JSON(404, gin.H{"error": "Session not found"})
		return
	}

	// Kill process
	if err := exec.Command("kill", strconv.Itoa(session.PID)).Run(); err != nil {
		log.Error("Failed to kill browser process: %v", err)
	}

	// Clean up data directory
	if err := exec.Command("rm", "-rf", session.DataDir).Run(); err != nil {
		log.Error("Failed to clean up browser data directory: %v", err)
	}

	delete(c.browsers, sessionID)

	log.Info("Browser session killed: id=%s, pid=%d", sessionID, session.PID)

	c.ctx.JSON(200, gin.H{"success": true})
}

// ListBrowsers lists all active browser sessions
func (c *BrowserController) ListBrowsers() {
	c.mu.RLock()
	defer c.mu.RUnlock()

	sessions := make([]gin.H, 0, len(c.browsers))
	for id, session := range c.browsers {
		sessions = append(sessions, gin.H{
			"sessionId": id,
			"pid":       session.PID,
			"port":      session.Port,
			"createdAt": session.CreatedAt,
		})
	}

	c.ctx.JSON(200, gin.H{"sessions": sessions})
}

// CreateSnapshot creates a snapshot of the current browser state (OverlayFS only)
func (c *BrowserController) CreateSnapshot() {
	if !isOverlayFSEnabled() {
		c.ctx.JSON(501, gin.H{"error": "OverlayFS snapshots are not enabled"})
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := c.ctx.ShouldBindJSON(&req); err != nil {
		c.ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	overlayBaseDir := os.Getenv("OVERLAY_BASE_DIR")
	if overlayBaseDir == "" {
		overlayBaseDir = "/var/lib/overlay"
	}

	snapshotsDir := filepath.Join(overlayBaseDir, "snapshots")
	snapshotPath := filepath.Join(snapshotsDir, req.Name)
	upperDir := filepath.Join(overlayBaseDir, "upper")

	// Create snapshots directory
	if err := os.MkdirAll(snapshotsDir, 0755); err != nil {
		log.Error("Failed to create snapshots directory: %v", err)
		c.ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Hardlink copy upper layer to snapshot (zero-copy)
	if err := filepath.Walk(upperDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(upperDir, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(snapshotPath, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Hardlink for zero-copy
		return os.Link(path, dstPath)
	}); err != nil {
		log.Error("Failed to create snapshot: %v", err)
		c.ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	log.Info("Snapshot created: name=%s, path=%s", req.Name, snapshotPath)

	c.ctx.JSON(200, gin.H{
		"success": true,
		"name":    req.Name,
		"path":    snapshotPath,
	})
}

// RollbackSnapshot rolls back to a snapshot (OverlayFS only)
func (c *BrowserController) RollbackSnapshot() {
	if !isOverlayFSEnabled() {
		c.ctx.JSON(501, gin.H{"error": "OverlayFS snapshots are not enabled"})
		return
	}

	snapshotName := c.ctx.Param("name")

	overlayBaseDir := os.Getenv("OVERLAY_BASE_DIR")
	if overlayBaseDir == "" {
		overlayBaseDir = "/var/lib/overlay"
	}

	snapshotPath := filepath.Join(overlayBaseDir, "snapshots", snapshotName)
	upperDir := filepath.Join(overlayBaseDir, "upper")
	mergeDir := filepath.Join(overlayBaseDir, "merged")

	// Check if snapshot exists
	if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
		c.ctx.JSON(404, gin.H{"error": "Snapshot not found"})
		return
	}

	// Unmount OverlayFS
	if err := exec.Command("umount", mergeDir).Run(); err != nil {
		log.Error("Failed to unmount OverlayFS: %v", err)
		c.ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Remove current upper layer
	if err := os.RemoveAll(upperDir); err != nil {
		log.Error("Failed to remove upper layer: %v", err)
		c.ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Hardlink copy snapshot to upper layer
	if err := filepath.Walk(snapshotPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(snapshotPath, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(upperDir, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return os.Link(path, dstPath)
	}); err != nil {
		log.Error("Failed to restore snapshot: %v", err)
		c.ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Remount OverlayFS
	if err := exec.Command("mount", "-t", "overlay", "overlay",
		"-o", fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s",
			filepath.Join(overlayBaseDir, "lower"),
			upperDir,
			filepath.Join(overlayBaseDir, "work")),
		mergeDir).Run(); err != nil {
		log.Error("Failed to remount OverlayFS: %v", err)
		c.ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	log.Info("Snapshot rolled back: name=%s", snapshotName)

	c.ctx.JSON(200, gin.H{"success": true})
}

// ListSnapshots lists all available snapshots (OverlayFS only)
func (c *BrowserController) ListSnapshots() {
	if !isOverlayFSEnabled() {
		c.ctx.JSON(501, gin.H{"error": "OverlayFS snapshots are not enabled"})
		return
	}

	overlayBaseDir := os.Getenv("OVERLAY_BASE_DIR")
	if overlayBaseDir == "" {
		overlayBaseDir = "/var/lib/overlay"
	}

	snapshotsDir := filepath.Join(overlayBaseDir, "snapshots")

	entries, err := os.ReadDir(snapshotsDir)
	if err != nil {
		if os.IsNotExist(err) {
			c.ctx.JSON(200, gin.H{"snapshots": []string{}})
			return
		}
		log.Error("Failed to list snapshots: %v", err)
		c.ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	snapshots := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			snapshots = append(snapshots, entry.Name())
		}
	}

	c.ctx.JSON(200, gin.H{"snapshots": snapshots})
}

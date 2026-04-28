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
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alibaba/opensandbox/execd/pkg/log"
	"github.com/gin-gonic/gin"
)

type BrowserController struct {
	ctx *gin.Context
}

type BrowserSession struct {
	PID       int       `json:"pid"`
	Port      int       `json:"port"`
	DataDir   string    `json:"dataDir"`
	CreatedAt time.Time `json:"createdAt"`
}

type browserSessionStore struct {
	mu       sync.RWMutex
	browsers map[string]*BrowserSession
}

func NewBrowserController(ctx *gin.Context) *BrowserController {
	return &BrowserController{ctx: ctx}
}

var globalBrowserSessions = &browserSessionStore{
	browsers: make(map[string]*BrowserSession),
}

func (s *browserSessionStore) set(id string, session *BrowserSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.browsers[id] = session
}

func (s *browserSessionStore) get(id string) (*BrowserSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.browsers[id]
	return session, ok
}

func (s *browserSessionStore) delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.browsers, id)
}

func (s *browserSessionStore) list() []gin.H {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := make([]gin.H, 0, len(s.browsers))
	for id, session := range s.browsers {
		sessions = append(sessions, gin.H{
			"sessionId": id,
			"pid":       session.PID,
			"port":      session.Port,
			"dataDir":   session.DataDir,
			"createdAt": session.CreatedAt,
		})
	}

	slices.SortFunc(sessions, func(a, b gin.H) int {
		left, _ := a["createdAt"].(time.Time)
		right, _ := b["createdAt"].(time.Time)
		if left.Before(right) {
			return -1
		}
		if left.After(right) {
			return 1
		}
		return 0
	})

	return sessions
}

// isOverlayFSEnabled checks if OverlayFS snapshots are enabled
func isOverlayFSEnabled() bool {
	return os.Getenv("ENABLE_OVERLAYFS_SNAPSHOTS") == "true"
}

func overlaySnapshotsAvailable() bool {
	return false
}

func launchBrowser(dataDir string) (*BrowserSession, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "/opt/opensandbox/browser-launch.sh", dataDir)
	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("browser launch timed out: %s", outputStr)
	}
	if err != nil {
		if outputStr != "" {
			return nil, fmt.Errorf("browser launch failed: %w: %s", err, outputStr)
		}
		return nil, fmt.Errorf("browser launch failed: %w", err)
	}

	port, pid, actualDataDir, err := parseBrowserLaunchOutput(outputStr)
	if err != nil {
		return nil, err
	}

	return &BrowserSession{
		PID:       pid,
		Port:      port,
		DataDir:   actualDataDir,
		CreatedAt: time.Now(),
	}, nil
}

func parseBrowserLaunchOutput(output string) (int, int, string, error) {
	var port, pid int
	dataDir := ""

	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "CDP_PORT:"):
			value := strings.TrimSpace(strings.TrimPrefix(trimmed, "CDP_PORT:"))
			parsedPort, err := strconv.Atoi(value)
			if err != nil {
				return 0, 0, "", fmt.Errorf("failed to parse CDP port %q: %w", value, err)
			}
			port = parsedPort
		case strings.HasPrefix(trimmed, "BROWSER_PID:"):
			value := strings.TrimSpace(strings.TrimPrefix(trimmed, "BROWSER_PID:"))
			parsedPID, err := strconv.Atoi(value)
			if err != nil {
				return 0, 0, "", fmt.Errorf("failed to parse browser pid %q: %w", value, err)
			}
			pid = parsedPID
		case strings.HasPrefix(trimmed, "USER_DATA_DIR:"):
			dataDir = strings.TrimSpace(strings.TrimPrefix(trimmed, "USER_DATA_DIR:"))
		}
	}

	if port == 0 {
		return 0, 0, "", fmt.Errorf("failed to find CDP_PORT in output: %s", output)
	}
	if pid == 0 {
		return 0, 0, "", fmt.Errorf("failed to find BROWSER_PID in output: %s", output)
	}
	if dataDir == "" {
		return 0, 0, "", fmt.Errorf("failed to find USER_DATA_DIR in output: %s", output)
	}

	return port, pid, dataDir, nil
}

// CreateBrowser creates a new browser session
func (c *BrowserController) CreateBrowser() {
	dataDir := "/tmp/browser-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	session, err := launchBrowser(dataDir)
	if err != nil {
		log.Error("Failed to start browser: %v", err)
		c.ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	sessionID := strconv.Itoa(session.Port)
	globalBrowserSessions.set(sessionID, session)

	log.Info("Browser session created: id=%s, pid=%d, port=%d, dataDir=%s", sessionID, session.PID, session.Port, session.DataDir)

	c.ctx.JSON(200, gin.H{
		"sessionId": sessionID,
		"cdpPort":   session.Port,
		"dataDir":   session.DataDir,
	})
}

// KillBrowser kills a browser session
func (c *BrowserController) KillBrowser() {
	sessionID := c.ctx.Param("sessionId")

	session, exists := globalBrowserSessions.get(sessionID)
	if !exists {
		c.ctx.JSON(404, gin.H{"error": "Session not found"})
		return
	}

	// Kill process
	if err := exec.Command("kill", strconv.Itoa(session.PID)).Run(); err != nil {
		log.Error("Failed to kill browser process: %v", err)
	}

	// Clean up data directory
	if err := os.RemoveAll(session.DataDir); err != nil {
		log.Error("Failed to clean up browser data directory: %v", err)
	}

	globalBrowserSessions.delete(sessionID)

	log.Info("Browser session killed: id=%s, pid=%d", sessionID, session.PID)

	c.ctx.JSON(200, gin.H{"success": true})
}

// ListBrowsers lists all active browser sessions
func (c *BrowserController) ListBrowsers() {
	c.ctx.JSON(200, gin.H{"sessions": globalBrowserSessions.list()})
}

// CreateSnapshot creates a snapshot of the current browser state (OverlayFS only)
func (c *BrowserController) CreateSnapshot() {
	if !overlaySnapshotsAvailable() {
		c.ctx.JSON(501, gin.H{"error": "OverlayFS snapshot backend is unavailable in the current container runtime"})
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
	if !overlaySnapshotsAvailable() {
		c.ctx.JSON(501, gin.H{"error": "OverlayFS snapshot backend is unavailable in the current container runtime"})
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
	if !overlaySnapshotsAvailable() {
		c.ctx.JSON(501, gin.H{"error": "OverlayFS snapshot backend is unavailable in the current container runtime"})
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

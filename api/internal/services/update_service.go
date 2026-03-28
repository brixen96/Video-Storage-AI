package services

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/brixen96/video-storage-ai/internal/config"
)

const (
	githubOwner = "brixen96"
	githubRepo  = "video-storage-ai"
	githubAPI   = "https://api.github.com/repos/brixen96/video-storage-ai/releases/latest"
)

// UpdateInfo holds version comparison data returned to the frontend.
type UpdateInfo struct {
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version"`
	UpdateAvailable bool   `json:"update_available"`
	DownloadURL     string `json:"download_url,omitempty"`
	ReleaseNotes    string `json:"release_notes,omitempty"`
	PublishedAt     string `json:"published_at,omitempty"`
}

type githubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	PublishedAt string        `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// UpdateService handles checking for and applying updates.
type UpdateService struct{}

func NewUpdateService() *UpdateService { return &UpdateService{} }

// CheckForUpdate queries the GitHub releases API and returns version comparison info.
func (s *UpdateService) CheckForUpdate() (*UpdateInfo, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", githubAPI, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "VideoStorageAI/"+config.Version)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to reach GitHub: %w", err)
	}
	defer resp.Body.Close()

	// 404 means no releases published yet — not an error
	if resp.StatusCode == http.StatusNotFound {
		cur := strings.TrimPrefix(config.Version, "v")
		return &UpdateInfo{CurrentVersion: cur, LatestVersion: cur, UpdateAvailable: false}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release JSON: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	currentVersion := strings.TrimPrefix(config.Version, "v")

	info := &UpdateInfo{
		CurrentVersion:  currentVersion,
		LatestVersion:   latestVersion,
		UpdateAvailable: latestVersion != currentVersion && currentVersion != "dev",
		ReleaseNotes:    release.Body,
		PublishedAt:     release.PublishedAt,
	}

	// Prefer a Windows ZIP asset; fall back to any ZIP.
	for _, asset := range release.Assets {
		lower := strings.ToLower(asset.Name)
		if strings.HasSuffix(lower, ".zip") && strings.Contains(lower, "windows") {
			info.DownloadURL = asset.BrowserDownloadURL
			break
		}
	}
	if info.DownloadURL == "" {
		for _, asset := range release.Assets {
			if strings.HasSuffix(strings.ToLower(asset.Name), ".zip") {
				info.DownloadURL = asset.BrowserDownloadURL
				break
			}
		}
	}

	return info, nil
}

// InstallUpdate downloads the release ZIP, extracts it to a staging folder, writes
// an updater.bat next to the executables, and spawns it detached so it can replace
// the binaries after this process exits.
func (s *UpdateService) InstallUpdate(downloadURL string) error {
	if downloadURL == "" {
		return fmt.Errorf("no download URL provided")
	}

	updateDir := filepath.Join(config.DataDir(), "updates")
	stagingDir := filepath.Join(updateDir, "staging")

	// Fresh staging directory every time.
	_ = os.RemoveAll(stagingDir)
	if err := os.MkdirAll(stagingDir, 0755); err != nil {
		return fmt.Errorf("failed to create staging dir: %w", err)
	}

	// Download the ZIP to a temp file in the update dir.
	zipPath := filepath.Join(updateDir, "release.zip")
	log.Printf("[Updater] Downloading %s", downloadURL)
	if err := downloadFile(zipPath, downloadURL); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	log.Printf("[Updater] Download complete, extracting...")

	// Extract to staging.
	if err := extractZip(zipPath, stagingDir); err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}
	_ = os.Remove(zipPath)

	// Determine where this binary lives — that's where we'll replace files.
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)

	// Write updater.bat alongside the executables.
	updaterPath := filepath.Join(exeDir, "updater.bat")
	script := buildUpdaterScript(stagingDir, exeDir)
	if err := os.WriteFile(updaterPath, []byte(script), 0644); err != nil {
		return fmt.Errorf("failed to write updater.bat: %w", err)
	}

	// Spawn the updater detached so it survives after this process stops.
	log.Printf("[Updater] Spawning updater.bat — service will stop now")
	cmd := exec.Command("cmd", "/c", "start", "/b", "", updaterPath)
	cmd.Dir = exeDir
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to spawn updater.bat: %w", err)
	}

	return nil
}

// buildUpdaterScript generates a .bat that stops services, replaces files, then restarts.
func buildUpdaterScript(stagingDir, targetDir string) string {
	return fmt.Sprintf(`@echo off
echo [Updater] Waiting for services to stop...
timeout /t 4 /nobreak > nul

echo [Updater] Stopping services (if installed)...
sc stop VideoStorageAI-API 2>nul
sc stop VideoStorageAI-Web 2>nul
timeout /t 3 /nobreak > nul

echo [Updater] Replacing API binary...
if exist "%s\VideoStorageAI-API.exe" (
    copy /Y "%s\VideoStorageAI-API.exe" "%s\VideoStorageAI-API.exe" >nul
)

echo [Updater] Replacing Web binary...
if exist "%s\VideoStorageAI-Web.exe" (
    copy /Y "%s\VideoStorageAI-Web.exe" "%s\VideoStorageAI-Web.exe" >nul
)

echo [Updater] Updating frontend assets...
if exist "%s\dist" (
    xcopy /E /I /Y "%s\dist" "%s\dist\" >nul
)

echo [Updater] Restarting services...
sc start VideoStorageAI-API 2>nul
sc start VideoStorageAI-Web 2>nul

echo [Updater] Cleaning up staging...
rmdir /s /q "%s"

echo [Updater] Update complete!
`,
		// API binary
		stagingDir, stagingDir, targetDir,
		// Web binary
		stagingDir, stagingDir, targetDir,
		// dist
		stagingDir, stagingDir, targetDir,
		// cleanup
		stagingDir,
	)
}

// downloadFile streams a URL response into a local file.
func downloadFile(dest, url string) error {
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d downloading %s", resp.StatusCode, url)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

// extractZip extracts all entries from src ZIP into dest directory.
func extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	cleanDest := filepath.Clean(dest) + string(os.PathSeparator)

	for _, f := range r.File {
		fPath := filepath.Join(dest, filepath.FromSlash(f.Name))

		// Zip-slip guard
		if !strings.HasPrefix(filepath.Clean(fPath)+string(os.PathSeparator), cleanDest) {
			return fmt.Errorf("illegal path in ZIP: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fPath, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fPath), 0755); err != nil {
			return err
		}

		out, err := os.OpenFile(fPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			out.Close()
			return err
		}

		_, copyErr := io.Copy(out, rc)
		out.Close()
		rc.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}

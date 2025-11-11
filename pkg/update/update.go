package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/mod/semver"
)

const (
	githubAPIURL    = "https://api.github.com/repos/ionut-t/perp/releases/latest"
	updateCheckFile = ".update_check.json"
)

// release represents a GitHub release
type release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	PublishedAt time.Time `json:"published_at"`
	ReleaseURL  string    `json:"html_url"`
	Body        string    `json:"body"`
}

// updateCheck represents the last update check information
type updateCheck struct {
	LastChecked      time.Time `json:"last_checked"`
	LatestVersion    string    `json:"latest_version"`
	CurrentVersion   string    `json:"current_version"`
	LatestVersionURL string    `json:"latest_version_url,omitempty"`
	Dismissed        bool      `json:"dismissed"`
}

type LatestReleaseInfo struct {
	TagName    string
	ReleaseURL string
	HasUpdate  bool
	Dismissed  bool
}

// Checker handles checking for updates
type Checker struct {
	currentVersion      string
	storageDir          string
	httpClient          *http.Client
	lastCheck           *updateCheck
	updateCheckInterval time.Duration
}

// New creates a new update checker
func New(currentVersion, storageDir string, hours float64) *Checker {
	c := &Checker{
		currentVersion:      currentVersion,
		storageDir:          storageDir,
		updateCheckInterval: time.Duration(hours) * time.Hour,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	c.lastCheck = c.getLastCheck()

	return c
}

// CheckForUpdate checks if a new version is available
func (c *Checker) CheckForUpdate() (*LatestReleaseInfo, error) {
	shouldCheck := c.shouldCheckForUpdate()

	if !shouldCheck {
		if c.lastCheck != nil && c.lastCheck.Dismissed {
			return nil, nil
		}

		return &LatestReleaseInfo{
			TagName:    c.lastCheck.LatestVersion,
			ReleaseURL: c.lastCheck.LatestVersionURL,
			HasUpdate:  c.compareVersions(c.lastCheck.LatestVersion),
			Dismissed:  c.lastCheck.Dismissed,
		}, nil
	}

	release, err := c.getLatestRelease()
	if err != nil {
		return nil, err
	}

	_ = c.saveUpdateCheck(release)

	return &LatestReleaseInfo{
		TagName:    release.TagName,
		ReleaseURL: release.ReleaseURL,
		HasUpdate:  c.compareVersions(release.TagName),
		Dismissed:  false,
	}, nil
}

func (c *Checker) DismissUpdate() error {
	if c.lastCheck == nil {
		return nil
	}

	c.lastCheck.Dismissed = true

	checkPath := filepath.Join(c.storageDir, updateCheckFile)
	data, err := json.MarshalIndent(c.lastCheck, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(checkPath, data, 0o644); err != nil {
		return err
	}

	return nil
}

func (c *Checker) IsUpdateDismissed() bool {
	if c.lastCheck == nil {
		return false
	}

	return c.lastCheck.Dismissed
}

func (c *Checker) getLastCheck() *updateCheck {
	checkPath := filepath.Join(c.storageDir, updateCheckFile)

	data, err := os.ReadFile(checkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return nil
	}

	var lastCheck updateCheck
	if err := json.Unmarshal(data, &lastCheck); err != nil {
		return nil
	}

	return &lastCheck
}

// shouldCheckForUpdate determines if we should check for updates based on the last check time
func (c *Checker) shouldCheckForUpdate() bool {
	if c.lastCheck == nil {
		return true
	}

	return time.Since(c.lastCheck.LastChecked) > c.updateCheckInterval
}

// getLatestRelease fetches the latest release information from GitHub
func (c *Checker) getLatestRelease() (*release, error) {
	req, err := http.NewRequest("GET", githubAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// GitHub recommends including User-Agent
	req.Header.Set("User-Agent", "perp-update-checker")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release info: %w", err)
	}

	//nolint:errcheck
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var release release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode release info: %w", err)
	}

	return &release, nil
}

// saveUpdateCheck saves the update check information
func (c *Checker) saveUpdateCheck(release *release) error {
	checkPath := filepath.Join(c.storageDir, updateCheckFile)

	check := updateCheck{
		LastChecked:      time.Now(),
		LatestVersion:    release.TagName,
		CurrentVersion:   c.currentVersion,
		LatestVersionURL: release.ReleaseURL,
		Dismissed:        false,
	}

	data, err := json.MarshalIndent(check, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(c.storageDir, 0o755); err != nil {
		return err
	}

	return os.WriteFile(checkPath, data, 0o644)
}

// compareVersions compares the current version with the latest version
func (c *Checker) compareVersions(latestVersion string) bool {
	if c.currentVersion == "dev" {
		return false
	}

	return semver.Compare(c.currentVersion, latestVersion) == -1
}

// GetLastCheck returns information about the last update check
func (c *Checker) GetLastCheck() (*updateCheck, error) {
	checkPath := filepath.Join(c.storageDir, updateCheckFile)

	data, err := os.ReadFile(checkPath)
	if err != nil {
		return nil, err
	}

	var lastCheck updateCheck
	if err := json.Unmarshal(data, &lastCheck); err != nil {
		return nil, err
	}

	return &lastCheck, nil
}

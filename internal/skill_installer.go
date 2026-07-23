package dub

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultSkillSourceURL = "https://raw.githubusercontent.com/Continuum-AI-Corp/orcadub-plugin/main/skills/dub-video/SKILL.md"
	maxSkillDocumentSize  = 1 << 20
)

var (
	skillLinkFile   = os.Link
	skillRenameFile = os.Rename
)

type skillInstallScope string

const (
	skillInstallProject skillInstallScope = "project"
	skillInstallGlobal  skillInstallScope = "global"
)

type skillInstallStatus string

const (
	skillInstallStatusInstalled skillInstallStatus = "installed"
	skillInstallStatusUpdated   skillInstallStatus = "updated"
	skillInstallStatusUnchanged skillInstallStatus = "unchanged"
	skillInstallStatusConflict  skillInstallStatus = "conflict"
	skillInstallStatusError     skillInstallStatus = "error"
)

type skillPlatform struct {
	ID                   string
	Name                 string
	ProjectRoot          string
	GlobalRoot           string
	DetectionPaths       []string
	GlobalDetectionPaths []string
	Executables          []string
}

type skillInstallTarget struct {
	PlatformIDs   []string `json:"platforms"`
	PlatformNames []string `json:"platform_names"`
	Path          string   `json:"path"`
}

type skillInstallResult struct {
	Platforms     []string           `json:"platforms"`
	PlatformNames []string           `json:"platform_names"`
	Path          string             `json:"path"`
	Status        skillInstallStatus `json:"status"`
	Error         string             `json:"error,omitempty"`
}

type skillInstallReport struct {
	Source  string               `json:"source"`
	Scope   skillInstallScope    `json:"scope"`
	Results []skillInstallResult `json:"results"`
}

type skillInstaller struct {
	client    *http.Client
	sourceURL string
}

// skillPlatforms mirrors Comet's platform catalog and Skill roots. A platform
// without a special global root uses the same relative root under the user's
// home directory.
var skillPlatforms = []skillPlatform{
	{
		ID:                   "claude",
		Name:                 "Claude Code",
		ProjectRoot:          ".claude",
		GlobalRoot:           ".claude",
		GlobalDetectionPaths: []string{".claude"},
		Executables:          []string{"claude"},
	},
	{ID: "cursor", Name: "Cursor", ProjectRoot: ".cursor", GlobalRoot: ".cursor"},
	{
		ID:                   "codex",
		Name:                 "Codex",
		ProjectRoot:          ".agents",
		GlobalRoot:           ".agents",
		DetectionPaths:       []string{".codex"},
		GlobalDetectionPaths: []string{".codex"},
		Executables:          []string{"codex"},
	},
	{
		ID:          "opencode",
		Name:        "OpenCode",
		ProjectRoot: ".opencode",
		GlobalRoot:  ".config/opencode",
	},
	{ID: "windsurf", Name: "Windsurf", ProjectRoot: ".windsurf", GlobalRoot: ".windsurf"},
	{ID: "cline", Name: "Cline", ProjectRoot: ".cline", GlobalRoot: ".cline"},
	{ID: "roocode", Name: "RooCode", ProjectRoot: ".roo", GlobalRoot: ".roo"},
	{ID: "continue", Name: "Continue", ProjectRoot: ".continue", GlobalRoot: ".continue"},
	{
		ID:          "github-copilot",
		Name:        "GitHub Copilot",
		ProjectRoot: ".github",
		GlobalRoot:  ".github",
		DetectionPaths: []string{
			".github/copilot-instructions.md",
			".github/instructions",
			".github/prompts",
			".github/skills",
		},
	},
	{ID: "gemini", Name: "Gemini CLI", ProjectRoot: ".gemini", GlobalRoot: ".gemini"},
	{ID: "amazon-q", Name: "Amazon Q Developer", ProjectRoot: ".amazonq", GlobalRoot: ".amazonq"},
	{ID: "qwen", Name: "Qwen Code", ProjectRoot: ".qwen", GlobalRoot: ".qwen"},
	{ID: "kilocode", Name: "Kilo Code", ProjectRoot: ".kilocode", GlobalRoot: ".kilocode"},
	{ID: "auggie", Name: "Auggie (Augment CLI)", ProjectRoot: ".augment", GlobalRoot: ".augment"},
	{ID: "kimicode", Name: "Kimi Code", ProjectRoot: ".kimi-code", GlobalRoot: ".kimi-code"},
	{ID: "kiro", Name: "Kiro", ProjectRoot: ".kiro", GlobalRoot: ".kiro"},
	{ID: "lingma", Name: "Lingma", ProjectRoot: ".lingma", GlobalRoot: ".lingma"},
	{ID: "junie", Name: "Junie", ProjectRoot: ".junie", GlobalRoot: ".junie"},
	{ID: "codebuddy", Name: "CodeBuddy Code", ProjectRoot: ".codebuddy", GlobalRoot: ".codebuddy"},
	{ID: "costrict", Name: "CoStrict", ProjectRoot: ".cospec", GlobalRoot: ".cospec"},
	{ID: "crush", Name: "Crush", ProjectRoot: ".crush", GlobalRoot: ".crush"},
	{ID: "factory", Name: "Factory Droid", ProjectRoot: ".factory", GlobalRoot: ".factory"},
	{ID: "iflow", Name: "iFlow", ProjectRoot: ".iflow", GlobalRoot: ".iflow"},
	{ID: "pi", Name: "Pi", ProjectRoot: ".pi", GlobalRoot: ".pi/agent"},
	{ID: "qoder", Name: "Qoder", ProjectRoot: ".qoder", GlobalRoot: ".qoder"},
	{
		ID:          "antigravity",
		Name:        "Antigravity",
		ProjectRoot: ".agents",
		GlobalRoot:  ".gemini/antigravity",
	},
	{
		ID:          "antigravity2",
		Name:        "Antigravity 2.0",
		ProjectRoot: ".agents",
		GlobalRoot:  ".gemini/config",
	},
	{ID: "bob", Name: "Bob Shell", ProjectRoot: ".bob", GlobalRoot: ".bob"},
	{ID: "forgecode", Name: "ForgeCode", ProjectRoot: ".forge", GlobalRoot: ".forge"},
	{ID: "trae", Name: "Trae", ProjectRoot: ".trae", GlobalRoot: ".trae"},
	{ID: "trae-cn", Name: "Trae CN", ProjectRoot: ".trae-cn", GlobalRoot: ".trae-cn"},
	{ID: "zcode", Name: "ZCode", ProjectRoot: ".zcode", GlobalRoot: ".zcode"},
	{
		ID:          "mimocode",
		Name:        "MimoCode",
		ProjectRoot: ".mimocode",
		GlobalRoot:  ".config/mimocode",
	},
}

func findSkillPlatform(id string) (skillPlatform, bool) {
	for _, platform := range skillPlatforms {
		if platform.ID == id {
			return platform, true
		}
	}
	return skillPlatform{}, false
}

func detectSkillPlatforms(
	projectDir string,
	homeDir string,
	lookPath func(string) (string, error),
) []string {
	detected := make([]string, 0)
platformLoop:
	for _, platform := range skillPlatforms {
		projectPaths := platform.DetectionPaths
		if len(projectPaths) == 0 {
			projectPaths = []string{platform.ProjectRoot}
		}
		if projectDir != "" {
			for _, marker := range projectPaths {
				if _, err := os.Stat(filepath.Join(projectDir, filepath.FromSlash(marker))); err == nil {
					detected = append(detected, platform.ID)
					continue platformLoop
				}
			}
		}
		if homeDir != "" {
			for _, marker := range platform.GlobalDetectionPaths {
				if _, err := os.Stat(filepath.Join(homeDir, filepath.FromSlash(marker))); err == nil {
					detected = append(detected, platform.ID)
					continue platformLoop
				}
			}
		}
		if lookPath != nil {
			for _, executable := range platform.Executables {
				if _, err := lookPath(executable); err == nil {
					detected = append(detected, platform.ID)
					continue platformLoop
				}
			}
		}
	}
	return detected
}

func resolveSkillTargets(
	platformIDs []string,
	scope skillInstallScope,
	projectDir string,
	homeDir string,
) ([]skillInstallTarget, error) {
	if scope != skillInstallProject && scope != skillInstallGlobal {
		return nil, fmt.Errorf("unknown install scope %q (use project or global)", scope)
	}

	targets := make([]skillInstallTarget, 0, len(platformIDs))
	targetIndexes := make(map[string]int, len(platformIDs))
	for _, id := range platformIDs {
		platform, ok := findSkillPlatform(id)
		if !ok {
			return nil, fmt.Errorf("unknown platform %q", id)
		}

		baseDir := projectDir
		root := platform.ProjectRoot
		if scope == skillInstallGlobal {
			baseDir = homeDir
			root = platform.GlobalRoot
		}
		if !filepath.IsAbs(baseDir) {
			return nil, fmt.Errorf("%s install base must be an absolute path: %q", scope, baseDir)
		}
		destination := filepath.Clean(filepath.Join(
			baseDir,
			filepath.FromSlash(root),
			"skills",
			"orcadub",
			"SKILL.md",
		))
		if index, exists := targetIndexes[destination]; exists {
			targets[index].PlatformIDs = append(targets[index].PlatformIDs, platform.ID)
			targets[index].PlatformNames = append(targets[index].PlatformNames, platform.Name)
			continue
		}

		targetIndexes[destination] = len(targets)
		targets = append(targets, skillInstallTarget{
			PlatformIDs:   []string{platform.ID},
			PlatformNames: []string{platform.Name},
			Path:          destination,
		})
	}
	return targets, nil
}

func newSkillInstaller() skillInstaller {
	return skillInstaller{
		client:    &http.Client{Timeout: 30 * time.Second},
		sourceURL: defaultSkillSourceURL,
	}
}

func (installer skillInstaller) install(
	ctx context.Context,
	platformIDs []string,
	scope skillInstallScope,
	projectDir string,
	homeDir string,
	force bool,
) (skillInstallReport, error) {
	targets, err := resolveSkillTargets(platformIDs, scope, projectDir, homeDir)
	if err != nil {
		return skillInstallReport{}, err
	}
	if len(targets) == 0 {
		return skillInstallReport{}, fmt.Errorf("no platforms selected")
	}

	document, err := installer.downloadSkill(ctx)
	if err != nil {
		return skillInstallReport{}, err
	}

	report := skillInstallReport{
		Source:  installer.sourceURL,
		Scope:   scope,
		Results: make([]skillInstallResult, 0, len(targets)),
	}
	for _, target := range targets {
		status, installErr := installSkillDocument(target.Path, document, force)
		result := skillInstallResult{
			Platforms:     append([]string(nil), target.PlatformIDs...),
			PlatformNames: append([]string(nil), target.PlatformNames...),
			Path:          target.Path,
			Status:        status,
		}
		if installErr != nil {
			result.Status = skillInstallStatusError
			result.Error = installErr.Error()
		}
		report.Results = append(report.Results, result)
	}
	return report, nil
}

func (installer skillInstaller) downloadSkill(ctx context.Context) ([]byte, error) {
	if installer.sourceURL == "" {
		return nil, fmt.Errorf("skill source URL is empty")
	}
	client := installer.client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		installer.sourceURL,
		http.NoBody,
	)
	if err != nil {
		return nil, fmt.Errorf("build Skill download request: %w", err)
	}
	request.Header.Set("User-Agent", "orcadub-skill-installer")
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("download OrcaDub Skill: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"download OrcaDub Skill: HTTP %d from %s",
			response.StatusCode,
			installer.sourceURL,
		)
	}
	document, err := io.ReadAll(io.LimitReader(response.Body, maxSkillDocumentSize+1))
	if err != nil {
		return nil, fmt.Errorf("read OrcaDub Skill: %w", err)
	}
	if len(document) > maxSkillDocumentSize {
		return nil, fmt.Errorf("downloaded OrcaDub Skill exceeds %d bytes", maxSkillDocumentSize)
	}
	if err := validateSkillDocument(document); err != nil {
		return nil, err
	}
	return document, nil
}

func validateSkillDocument(document []byte) error {
	scanner := bufio.NewScanner(bytes.NewReader(document))
	if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != "---" {
		return fmt.Errorf("downloaded document is not a Skill: missing YAML frontmatter")
	}

	foundName := false
	foundEnd := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "---" {
			foundEnd = true
			break
		}
		if line == "name: dub-video" {
			foundName = true
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("validate downloaded Skill: %w", err)
	}
	if !foundEnd {
		return fmt.Errorf("downloaded document is not a Skill: unterminated YAML frontmatter")
	}
	if !foundName {
		return fmt.Errorf("downloaded Skill has no exact `name: dub-video` frontmatter")
	}
	return nil
}

//nolint:gocyclo // explicit overwrite-safe file states are clearer kept in one operation.
func installSkillDocument(
	destination string,
	document []byte,
	force bool,
) (skillInstallStatus, error) {
	existing, err := os.ReadFile(destination)
	exists := err == nil
	if err != nil && !os.IsNotExist(err) {
		return skillInstallStatusError, fmt.Errorf("read existing Skill %s: %w", destination, err)
	}
	if exists {
		if bytes.Equal(existing, document) {
			return skillInstallStatusUnchanged, nil
		}
		if !force {
			return skillInstallStatusConflict, nil
		}
	}

	destinationDir := filepath.Dir(destination)
	if err := os.MkdirAll(destinationDir, 0o755); err != nil {
		return skillInstallStatusError, fmt.Errorf("create Skill directory %s: %w", destinationDir, err)
	}
	tempFile, err := os.CreateTemp(destinationDir, ".SKILL.md-*")
	if err != nil {
		return skillInstallStatusError, fmt.Errorf("create temporary Skill file: %w", err)
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = os.Remove(tempPath)
	}()

	if _, err := tempFile.Write(document); err != nil {
		_ = tempFile.Close()
		return skillInstallStatusError, fmt.Errorf("write temporary Skill file: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return skillInstallStatusError, fmt.Errorf("sync temporary Skill file: %w", err)
	}
	if err := tempFile.Chmod(0o644); err != nil {
		_ = tempFile.Close()
		return skillInstallStatusError, fmt.Errorf("set Skill file permissions: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return skillInstallStatusError, fmt.Errorf("close temporary Skill file: %w", err)
	}

	if !exists {
		if err := skillLinkFile(tempPath, destination); err != nil {
			if os.IsExist(err) {
				concurrent, readErr := os.ReadFile(destination)
				if readErr != nil {
					return skillInstallStatusError, fmt.Errorf(
						"inspect concurrently installed Skill %s: %w",
						destination,
						readErr,
					)
				}
				if bytes.Equal(concurrent, document) {
					return skillInstallStatusUnchanged, nil
				}
				if !force {
					return skillInstallStatusConflict, nil
				}
			} else {
				return skillInstallStatusError, fmt.Errorf(
					"install new Skill at %s without overwriting: %w",
					destination,
					err,
				)
			}
		} else {
			return skillInstallStatusInstalled, nil
		}
	}
	if err := skillRenameFile(tempPath, destination); err != nil {
		return skillInstallStatusError, fmt.Errorf("install Skill at %s: %w", destination, err)
	}
	return skillInstallStatusUpdated, nil
}

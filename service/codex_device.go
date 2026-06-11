package service

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
)

// ansiEscape matches CSI escape sequences (e.g. color codes like `\033[94m`).
// Codex CLI wraps URL and short code with these for terminal highlighting.
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

const (
	codexCLIPathInPath     = "codex"
	codexMountedNVMRoot    = "/host-nvm/versions/node"
	codexAuthHome          = "/data/codex-home"
	codexAuthJSONRelative  = ".codex/auth.json"
	deviceAuthTimeout      = 15 * time.Minute
	deviceAuthStartupLimit = 20 * time.Second
)

// DeviceAuthOutput holds the parsed output from `codex login --device-auth`.
type DeviceAuthOutput struct {
	VerificationURL string `json:"verification_url"`
	UserCode        string `json:"user_code"`
	ExpiresIn       int    `json:"expires_in"`
}

type deviceAuthSession struct {
	StartedAt    time.Time
	UserCode     string
	URL          string
	cmd          *exec.Cmd
	cancel       context.CancelFunc
	resultChan   chan *DeviceAuthResult
	authJSONPath string
}

// DeviceAuthResult holds the result of a completed device auth.
type DeviceAuthResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	AccountID    string `json:"account_id"`
	Email        string `json:"email"`
	ExpiresAt    string `json:"expires_at"`
	LastRefresh  string `json:"last_refresh"`
}

var (
	deviceAuthSessions = map[string]*deviceAuthSession{}
	deviceAuthMu       sync.Mutex
)

// StartDeviceAuth starts `codex login --device-auth`, captures the OpenAI URL and
// short code, and keeps the subprocess alive in the background until completion.
func StartDeviceAuth(ctx context.Context) (*DeviceAuthOutput, *DeviceAuthSessionHandle, error) {
	ctx, cancel := context.WithTimeout(ctx, deviceAuthTimeout)
	authJSONPath, err := ensureCodexAuthHome()
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("codex device auth: prepare auth home: %w", err)
	}

	cmd, err := buildCodexDeviceAuthCommand(ctx, authJSONPath)
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("codex device auth: resolve CLI: %w", err)
	}

	reader, writer := io.Pipe()
	cmd.Stdout = writer
	cmd.Stderr = writer

	parsedCh := make(chan *DeviceAuthOutput, 1)
	parseErrCh := make(chan error, 1)
	waitErrCh := make(chan error, 1)

	go scanCodexDeviceAuthOutput(reader, parsedCh, parseErrCh)

	if err := cmd.Start(); err != nil {
		_ = writer.Close()
		cancel()
		return nil, nil, fmt.Errorf("codex device auth: start failed: %w", err)
	}

	go func() {
		err := cmd.Wait()
		_ = writer.Close()
		waitErrCh <- err
	}()

	var parsed *DeviceAuthOutput
	select {
	case parsed = <-parsedCh:
	case err := <-parseErrCh:
		cancel()
		return nil, nil, fmt.Errorf("codex device auth: parse output: %w", err)
	case err := <-waitErrCh:
		cancel()
		if err != nil {
			return nil, nil, fmt.Errorf("codex device auth: exited before code was available: %w", err)
		}
		return nil, nil, errors.New("codex device auth: exited before code was available")
	case <-time.After(deviceAuthStartupLimit):
		cancel()
		return nil, nil, errors.New("codex device auth: timed out waiting for authentication code")
	}

	sessionID := generateSessionID()
	session := &deviceAuthSession{
		StartedAt:    time.Now(),
		UserCode:     parsed.UserCode,
		URL:          parsed.VerificationURL,
		cmd:          cmd,
		cancel:       cancel,
		resultChan:   make(chan *DeviceAuthResult, 1),
		authJSONPath: authJSONPath,
	}

	deviceAuthMu.Lock()
	deviceAuthSessions[sessionID] = session
	deviceAuthMu.Unlock()

	go waitForDeviceAuthCompletion(sessionID, session, waitErrCh, cancel)

	logger.LogInfo(ctx, fmt.Sprintf("codex device auth started: session=%s code=%s", sessionID, parsed.UserCode))

	return parsed, &DeviceAuthSessionHandle{ID: sessionID}, nil
}

// DeviceAuthSessionHandle identifies an active device auth session.
type DeviceAuthSessionHandle struct {
	ID string
}

// PollDeviceAuth checks if the device auth session has completed.
// Returns nil, nil while still pending.
//
// Behaviour: on every call, we re-read the auth.json from disk. The codex
// CLI 0.137.0 writes the file once the user authorizes; the file watcher
// pattern in the original design relied on cmd.Wait() to wake us up, but
// in practice the CLI has been observed to keep polling for much longer
// than needed, leaving the user staring at "Waiting for authorization..."
// even though the credential is already on disk.
//
// To avoid that, we no longer trust cmd.Wait() as the source of truth —
// we trust the auth.json file. The channel is preserved for in-process
// handoff if cmd.Wait() does fire, but the file is the contract.
func PollDeviceAuth(sessionID string) (*DeviceAuthResult, error) {
	authJSONPath := filepath.Join(codexAuthHome, codexAuthJSONRelative)

	// Always check disk first — this is the canonical signal that the
	// user has finished authorizing.
	if result, err := readCodexAuthJSON(authJSONPath); err == nil {
		// Clean up the in-process session so future polls return cleanly.
		deviceAuthMu.Lock()
		delete(deviceAuthSessions, sessionID)
		deviceAuthMu.Unlock()
		return result, nil
	}

	// Fall back to the in-process channel for fast-path handoff when
	// cmd.Wait() does fire in time.
	deviceAuthMu.Lock()
	session, ok := deviceAuthSessions[sessionID]
	deviceAuthMu.Unlock()
	if !ok {
		// Session was already cleared (we may have just returned above).
		// Treat the next poll as a re-check; the disk check above will
		// return the credential when it appears.
		return nil, nil
	}

	select {
	case result, ok := <-session.resultChan:
		if !ok || result == nil {
			return nil, errors.New("device auth failed or timed out")
		}
		return result, nil
	default:
		return nil, nil
	}
}

func waitForDeviceAuthCompletion(sessionID string, session *deviceAuthSession, waitErrCh <-chan error, cancel context.CancelFunc) {
	defer cancel()
	defer func() {
		deviceAuthMu.Lock()
		delete(deviceAuthSessions, sessionID)
		deviceAuthMu.Unlock()
	}()

	err := <-waitErrCh
	if err != nil {
		logger.LogError(context.Background(), fmt.Sprintf("codex device auth: CLI exited with error: %v", err))
		session.resultChan <- nil
		close(session.resultChan)
		return
	}

	result, err := readCodexAuthJSON(session.authJSONPath)
	if err != nil {
		logger.LogError(context.Background(), fmt.Sprintf("codex device auth: read auth.json: %v", err))
		session.resultChan <- nil
		close(session.resultChan)
		return
	}

	session.resultChan <- result
	close(session.resultChan)
}

func ensureCodexAuthHome() (string, error) {
	if err := os.MkdirAll(filepath.Join(codexAuthHome, ".codex"), 0o700); err != nil {
		return "", err
	}
	return filepath.Join(codexAuthHome, codexAuthJSONRelative), nil
}

func buildCodexDeviceAuthCommand(ctx context.Context, authJSONPath string) (*exec.Cmd, error) {
	env := append(os.Environ(),
		"CI=true",
		"HOME="+codexAuthHome,
	)

	if path, err := exec.LookPath(codexCLIPathInPath); err == nil {
		cmd := exec.CommandContext(ctx, path, "login", "--device-auth")
		cmd.Env = env
		return cmd, nil
	}

	nodePath, codexScriptPath, err := resolveMountedCodexCLI()
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, nodePath, codexScriptPath, "login", "--device-auth")
	cmd.Env = env
	cmd.Dir = filepath.Dir(authJSONPath)
	return cmd, nil
}

func resolveMountedCodexCLI() (string, string, error) {
	nodeCandidates, err := filepath.Glob(filepath.Join(codexMountedNVMRoot, "*/bin/node"))
	if err != nil {
		return "", "", err
	}
	scriptCandidates, err := filepath.Glob(filepath.Join(codexMountedNVMRoot, "*/lib/node_modules/@openai/codex/bin/codex.js"))
	if err != nil {
		return "", "", err
	}
	if len(nodeCandidates) == 0 || len(scriptCandidates) == 0 {
		return "", "", errors.New("codex CLI not found in PATH or mounted /host-nvm")
	}
	sort.Strings(nodeCandidates)
	sort.Strings(scriptCandidates)
	return nodeCandidates[len(nodeCandidates)-1], scriptCandidates[len(scriptCandidates)-1], nil
}

func scanCodexDeviceAuthOutput(r io.Reader, parsedCh chan<- *DeviceAuthOutput, errCh chan<- error) {
	scanner := bufio.NewScanner(r)
	// Codex CLI emits lines like "   \033[94mAMX4-E7YV3\033[0m" with ANSI color
	// codes around the URL and short code. The default 64K buffer is fine, but
	// we need to strip the ANSI sequences so `looksLikeCodexDeviceCode` and the
	// URL prefix check see the raw token.
	stripANSI := func(line string) string {
		return strings.TrimSpace(ansiEscape.ReplaceAllString(line, ""))
	}
	var url, code string
	sent := false

	for scanner.Scan() {
		line := stripANSI(scanner.Text())
		if line == "" {
			continue
		}

		if url == "" && strings.Contains(line, "https://auth.openai.com/codex/device") {
			if strings.HasPrefix(line, "http") {
				url = line
			} else {
				// Whole line is e.g. "Open this link ... https://auth.openai.com/codex/device".
				// `stripANSI` already trimmed it; the URL is inside the sentence.
				url = "https://auth.openai.com/codex/device"
			}
		}

		if code == "" && looksLikeCodexDeviceCode(line) {
			code = line
		}

		if !sent && code != "" {
			if url == "" {
				url = "https://auth.openai.com/codex/device"
			}
			parsedCh <- &DeviceAuthOutput{
				VerificationURL: url,
				UserCode:        code,
				ExpiresIn:       900,
			}
			sent = true
		}
	}

	if err := scanner.Err(); err != nil {
		if !sent {
			errCh <- err
		}
		return
	}

	if !sent {
		errCh <- errors.New("could not parse authentication code from codex CLI output")
	}
}

func looksLikeCodexDeviceCode(value string) bool {
	if strings.HasPrefix(value, "http") || len(value) < 8 || len(value) > 32 || !strings.Contains(value, "-") {
		return false
	}
	for _, r := range value {
		switch {
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-':
		default:
			return false
		}
	}
	return true
}

func readCodexAuthJSON(path string) (*DeviceAuthResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read auth.json: %w", err)
	}

	var raw map[string]interface{}
	if err := common.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse auth.json: %w", err)
	}

	// Codex 0.137.0 nests tokens under "tokens": { access_token, ... }.
	// Older / hand-crafted auth.json files keep them at the top level.
	// We accept both shapes.
	tokens, _ := raw["tokens"].(map[string]interface{})
	if tokens == nil {
		tokens = raw
	}

	accessToken, _ := tokens["access_token"].(string)
	if accessToken == "" {
		if v, ok := raw["access_token"].(string); ok {
			accessToken = v
		}
	}
	refreshToken, _ := tokens["refresh_token"].(string)
	if refreshToken == "" {
		if v, ok := raw["refresh_token"].(string); ok {
			refreshToken = v
		}
	}
	email, _ := raw["email"].(string)
	if email == "" {
		if v, ok := tokens["email"].(string); ok {
			email = v
		}
	}
	accountID := extractAccountID(raw)
	if accountID == "" {
		accountID = extractAccountID(tokens)
	}
	if accessToken == "" {
		return nil, errors.New("auth.json missing access_token")
	}

	now := time.Now()
	expiresAt := ""
	if exp, ok := raw["expires_at"].(string); ok && exp != "" {
		expiresAt = exp
	} else if expIn, ok := raw["expires_in"].(float64); ok && expIn > 0 {
		expiresAt = now.Add(time.Duration(expIn) * time.Second).Format(time.RFC3339)
	} else {
		expiresAt = now.Add(1 * time.Hour).Format(time.RFC3339)
	}

	return &DeviceAuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		AccountID:    accountID,
		Email:        email,
		ExpiresAt:    expiresAt,
		LastRefresh:  now.Format(time.RFC3339),
	}, nil
}

func extractAccountID(raw map[string]interface{}) string {
	if v, ok := raw["account_id"].(string); ok && v != "" {
		return v
	}
	if v, ok := raw["chatgpt_account_id"].(string); ok && v != "" {
		return v
	}
	if auth, ok := raw["auth"].(map[string]interface{}); ok {
		if v, ok := auth["account_id"].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

func generateSessionID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// ParseAuthJSON parses a raw auth.json string and returns DeviceAuthResult.
func ParseAuthJSON(raw string) (*DeviceAuthResult, error) {
	var data map[string]interface{}
	if err := common.Unmarshal([]byte(raw), &data); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	accessToken, _ := data["access_token"].(string)
	refreshToken, _ := data["refresh_token"].(string)
	email, _ := data["email"].(string)
	accountID := extractAccountID(data)
	if accessToken == "" {
		return nil, errors.New("auth.json missing access_token")
	}

	now := time.Now()
	expiresAt := ""
	if exp, ok := data["expires_at"].(string); ok && exp != "" {
		expiresAt = exp
	} else if expIn, ok := data["expires_in"].(float64); ok && expIn > 0 {
		expiresAt = now.Add(time.Duration(expIn) * time.Second).Format(time.RFC3339)
	} else {
		expiresAt = now.Add(1 * time.Hour).Format(time.RFC3339)
	}

	return &DeviceAuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		AccountID:    accountID,
		Email:        email,
		ExpiresAt:    expiresAt,
		LastRefresh:  now.Format(time.RFC3339),
	}, nil
}

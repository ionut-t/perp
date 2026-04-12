// Package lsp provides a minimal LSP client for communicating with the
// postgres-language-server binary over stdio.
package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/ionut-t/goeditor/core"
	"github.com/ionut-t/perp/pkg/server"
)

const (
	virtualDocURI = "file:///tmp/perp-scratch.sql"
	schemaURI     = "https://pg-language-server.com/0.22.1/schema.json"
)

// jsonrpcMsg is the base structure for LSP JSON-RPC 2.0 messages.
type jsonrpcMsg struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Client is a minimal LSP client that communicates with a postgres-language-server
// subprocess over stdio.
type Client struct {
	stdin      io.WriteCloser
	stdout     *bufio.Reader
	cmd        *exec.Cmd // the running LSP subprocess; used to terminate it on Close
	cfgDir     string    // temp directory holding the config file; cleaned up on Close
	binaryPath string

	mu      sync.Mutex
	reqID   atomic.Int64
	pending map[int64]chan *jsonrpcMsg

	// DocVersion is incremented on each DidChange call.
	DocVersion int
}

// New spawns the postgres-language-server binary, writes a temporary config file
// with the given DB credentials, and completes the LSP initialization handshake.
func New(binaryPath string, db server.Server) (*Client, error) {
	cfgDir, err := writeTempConfig(db)
	if err != nil {
		return nil, fmt.Errorf("lsp: write config: %w", err)
	}

	cmd := exec.Command(binaryPath, "lsp-proxy", "--config-path", cfgDir)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		_ = os.RemoveAll(cfgDir)
		return nil, fmt.Errorf("lsp: stdin pipe: %w", err)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		_ = os.RemoveAll(cfgDir)
		return nil, fmt.Errorf("lsp: stdout pipe: %w", err)
	}

	// Discard stderr so it doesn't pollute the TUI.
	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		_ = os.RemoveAll(cfgDir)
		return nil, fmt.Errorf("lsp: start binary: %w", err)
	}

	client := &Client{
		stdin:      stdin,
		stdout:     bufio.NewReader(stdoutPipe),
		cmd:        cmd,
		cfgDir:     cfgDir,
		binaryPath: binaryPath,
		pending:    make(map[int64]chan *jsonrpcMsg),
	}

	// Start reading responses in the background.
	go client.readLoop()

	// Perform the LSP initialize handshake.
	if err := client.initialize(); err != nil {
		client.Close()
		return nil, fmt.Errorf("lsp: initialize: %w", err)
	}

	return client, nil
}

// DidOpen notifies the server that a document has been opened.
func (c *Client) DidOpen(content string) error {
	return c.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri":        virtualDocURI,
			"languageId": "sql",
			"version":    c.DocVersion,
			"text":       content,
		},
	})
}

// DidChange notifies the server of a document content change.
func (c *Client) DidChange(content string) error {
	c.DocVersion++
	return c.notify("textDocument/didChange", map[string]any{
		"textDocument": map[string]any{
			"uri":     virtualDocURI,
			"version": c.DocVersion,
		},
		"contentChanges": []map[string]any{
			{"text": content},
		},
	})
}

// Completion requests completions at the given zero-indexed line and character position.
func (c *Client) Completion(ctx context.Context, line, char int) ([]core.Completion, error) {
	result, err := c.call(ctx, "textDocument/completion", map[string]any{
		"textDocument": map[string]any{"uri": virtualDocURI},
		"position": map[string]any{
			"line":      line,
			"character": char,
		},
		"context": map[string]any{
			"triggerKind": 1, // Invoked
		},
	})
	if err != nil {
		return nil, err
	}

	// The result can be CompletionList or CompletionItem[].
	// Try CompletionList first.
	var list struct {
		Items []lspCompletionItem `json:"items"`
	}
	if err := json.Unmarshal(result, &list); err == nil && list.Items != nil {
		return convertItems(list.Items), nil
	}

	// Try bare array.
	var items []lspCompletionItem
	if err := json.Unmarshal(result, &items); err == nil {
		return convertItems(items), nil
	}

	return nil, nil
}

// Close stops the LSP subprocess and releases resources.
func (c *Client) Close() {
	_ = c.stdin.Close()

	// stop gracefully shuts down all processes spawned by the LSP binary.
	_ = exec.Command(c.binaryPath, "stop").Run()

	// Kill the main process as a fallback and reap it in the background.
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
		go c.cmd.Wait() //nolint:errcheck // reap to avoid zombie
	}

	if c.cfgDir != "" {
		_ = os.RemoveAll(c.cfgDir)
	}
}

// writeTempConfig creates a temporary directory with a postgres-language-server.jsonc
// config file containing the given DB credentials.
func writeTempConfig(db server.Server) (string, error) {
	dir, err := os.MkdirTemp("", "perp-lsp-*")
	if err != nil {
		return "", err
	}

	type dbSection struct {
		Host              string `json:"host"`
		Port              int    `json:"port"`
		Username          string `json:"username"`
		Password          string `json:"password"`
		Database          string `json:"database"`
		ConnTimeoutSecs   int    `json:"connTimeoutSecs"`
		DisableConnection bool   `json:"disableConnection"`
	}

	cfg := struct {
		Schema string    `json:"$schema"`
		DB     dbSection `json:"db"`
	}{
		Schema: schemaURI,
		DB: dbSection{
			Host:            db.Address,
			Port:            db.Port,
			Username:        db.Username,
			Password:        db.Password,
			Database:        db.Database,
			ConnTimeoutSecs: 10,
		},
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}

	cfgPath := filepath.Join(dir, "postgres-language-server.jsonc")
	if err := os.WriteFile(cfgPath, data, 0o600); err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}

	return dir, nil
}

func (c *Client) initialize() error {
	ctx := context.Background()

	rootURI := "file://" + c.cfgDir

	_, err := c.call(ctx, "initialize", map[string]any{
		"processId": nil,
		"clientInfo": map[string]any{
			"name":    "perp",
			"version": "1.0.0",
		},
		"rootUri": rootURI,
		"workspaceFolders": []map[string]any{
			{"uri": rootURI, "name": "perp"},
		},
		"capabilities": map[string]any{
			"textDocument": map[string]any{
				"completion": map[string]any{
					"completionItem": map[string]any{
						"documentationFormat": []string{"plaintext", "markdown"},
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}

	return c.notify("initialized", map[string]any{})
}

// call sends a JSON-RPC request and waits for the response.
func (c *Client) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id := c.reqID.Add(1)

	rawParams, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	ch := make(chan *jsonrpcMsg, 1)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()

	msg := jsonrpcMsg{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  method,
		Params:  rawParams,
	}

	if err := c.sendMsg(msg); err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, err
	}

	select {
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, ctx.Err()
	case resp := <-ch:
		if resp.Error != nil {
			return nil, fmt.Errorf("lsp error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		return resp.Result, nil
	}
}

// notify sends a JSON-RPC notification (no response expected).
func (c *Client) notify(method string, params any) error {
	rawParams, err := json.Marshal(params)
	if err != nil {
		return err
	}

	return c.sendMsg(jsonrpcMsg{
		JSONRPC: "2.0",
		Method:  method,
		Params:  rawParams,
	})
}

// sendMsg serialises a message and writes it with LSP framing headers.
func (c *Client) sendMsg(msg jsonrpcMsg) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, err := io.WriteString(c.stdin, header); err != nil {
		return err
	}
	_, err = c.stdin.Write(body)
	return err
}

// readLoop reads LSP messages from stdout and dispatches them.
func (c *Client) readLoop() {
	for {
		msg, err := c.readMsg()
		if err != nil {
			// Server closed or crashed — drain pending channels.
			c.mu.Lock()
			for id, ch := range c.pending {
				ch <- &jsonrpcMsg{Error: &jsonrpcError{Code: -1, Message: err.Error()}}
				delete(c.pending, id)
			}
			c.mu.Unlock()
			return
		}

		if msg.ID != nil {
			// Response to a previous call.
			c.mu.Lock()
			ch, ok := c.pending[*msg.ID]
			if ok {
				delete(c.pending, *msg.ID)
			}
			c.mu.Unlock()

			if ok {
				ch <- msg
			}
		}
	}
}

// readMsg reads one LSP message from stdout using Content-Length framing.
func (c *Client) readMsg() (*jsonrpcMsg, error) {
	var contentLength int

	// Read headers until blank line.
	for {
		line, err := c.stdout.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("lsp: read header: %w", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if v, ok := strings.CutPrefix(line, "Content-Length:"); ok {
			v = strings.TrimSpace(v)
			contentLength, err = strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("lsp: invalid Content-Length: %w", err)
			}
		}
	}

	if contentLength == 0 {
		return nil, fmt.Errorf("lsp: missing Content-Length")
	}

	body := make([]byte, contentLength)
	if _, err := io.ReadFull(c.stdout, body); err != nil {
		return nil, fmt.Errorf("lsp: read body: %w", err)
	}

	var msg jsonrpcMsg
	if err := json.Unmarshal(body, &msg); err != nil {
		return nil, fmt.Errorf("lsp: unmarshal: %w", err)
	}

	return &msg, nil
}

func convertItems(items []lspCompletionItem) []core.Completion {
	completions := make([]core.Completion, len(items))
	for i, item := range items {
		completions[i] = toCompletion(item)
	}
	return completions
}

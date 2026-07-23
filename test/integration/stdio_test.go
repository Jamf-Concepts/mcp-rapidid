//go:build integration

// Copyright 2026, Jamf Software LLC

package integration

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"
)

type mcpStdioRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpStdioResponse struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      int                    `json:"id"`
	Method  string                 `json:"method"`
	Result  json.RawMessage        `json:"result"`
	Params  json.RawMessage        `json:"params,omitempty"`
	Error   *mcpStdioResponseError `json:"error,omitempty"`
}

type mcpStdioResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type mcpLogNotification struct {
	Level  string `json:"level"`
	Logger string `json:"logger"`
	Data   any    `json:"data"`
}

type stdoutResult struct {
	Line []byte
	Err  error
}

type mcpResult struct {
	Content json.RawMessage `json:"content"`
	IsError bool            `json:"isError"`
}

func send(stdin io.WriteCloser, req mcpStdioRequest) error {
	line, err := json.Marshal(req)
	if err != nil {
		return err
	}

	_, err = stdin.Write(append(line, '\n'))
	if err != nil {
		return err
	}

	return nil
}

func recv(stdout *bufio.Reader, timeout time.Duration) (mcpStdioResponse, error) {
	ch := make(chan stdoutResult, 1)

	go func() {
		line, err := stdout.ReadBytes('\n')
		ch <- stdoutResult{Line: line, Err: err}
	}()

	select {
	case res := <-ch:
		if res.Err != nil {
			return mcpStdioResponse{}, res.Err
		}

		var jsonRPCResponse mcpStdioResponse
		fmt.Println(string(res.Line))
		err := json.Unmarshal(res.Line, &jsonRPCResponse)
		if err != nil {
			return mcpStdioResponse{}, err
		}

		return jsonRPCResponse, nil
	case <-time.After(timeout):
		fmt.Printf("timed out after %s waiting for response\n", timeout)
		return mcpStdioResponse{}, nil
	}
}

func recvResponse(stdout *bufio.Reader, wantID int, timeout time.Duration) (mcpStdioResponse, []mcpLogNotification, error) {
	deadline := time.Now().Add(timeout)
	var logs []mcpLogNotification

	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return mcpStdioResponse{}, nil, fmt.Errorf("timed out waiting for response id %d", wantID)
		}

		res, err := recv(stdout, remaining)
		if err != nil {
			return mcpStdioResponse{}, nil, err
		}

		if res.Method == "notifications/message" {
			var logMsg mcpLogNotification
			err = json.Unmarshal(res.Params, &logMsg)
			if err != nil {
				fmt.Printf("failed to devode log notification: %s", err)
				continue
			}

			logs = append(logs, logMsg)
			continue
		}

		if res.ID == wantID {
			return res, logs, nil
		}

		fmt.Printf("skipping non-matching message (id=%d): %+v\n", res.ID, res)
	}
}

func ptr[T any](v T) *T {
	return &v
}

func setupMCPSession(t *testing.T, envOverrides map[string]string) (io.WriteCloser, *bufio.Reader, int) {
	t.Helper()

	bin := os.Getenv("RI_MCP_BIN")
	if bin == "" {
		t.Fatal("RI_MCP_BIN environment variable not present")
	}

	env := map[string]string{
		"RI_HOST":      os.Getenv("RI_HOST"),
		"RI_USER":      os.Getenv("RI_USER"),
		"RI_PASSWORD":  os.Getenv("RI_PASSWORD"),
		"RI_LOG_LEVEL": os.Getenv("RI_LOG_LEVEL"),
	}
	for k, v := range envOverrides {
		env[k] = v
	}

	var cmdEnv []string
	for k, v := range env {
		cmdEnv = append(cmdEnv, k+"="+v)
	}

	cmd := exec.Command(bin)
	cmd.Env = cmdEnv
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}

	err = cmd.Start()

	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		stdin.Close()
		cmd.Process.Kill()
		cmd.Wait()
	})

	reader := bufio.NewReader(stdout)

	send(stdin, mcpStdioRequest{JSONRPC: "2.0", ID: ptr(1), Method: "initialize",
		Params: json.RawMessage(`{"protocolVersion":"2024-11-05","clientInfo":{"name":"integration-test","version":"0.0.0"},"capabilities":{}}`)})
	if _, _, err := recvResponse(reader, 1, 3*time.Second); err != nil {
		t.Fatalf("initialize failed: %v", err)
	}

	send(stdin, mcpStdioRequest{JSONRPC: "2.0", Method: "notifications/initialized", Params: json.RawMessage(`{}`)})

	send(stdin, mcpStdioRequest{JSONRPC: "2.0", ID: ptr(2), Method: "logging/setLevel",
		Params: json.RawMessage(`{"level":"debug"}`)})
	if _, _, err := recvResponse(reader, 2, 3*time.Second); err != nil {
		t.Fatalf("logging/setLevel failed: %v", err)
	}

	send(stdin, mcpStdioRequest{JSONRPC: "2.0", ID: ptr(3), Method: "tools/list",
		Params: json.RawMessage(`{}`)})
	if _, _, err := recvResponse(reader, 3, 3*time.Second); err != nil {
		t.Fatalf("tools/list failed: %v", err)
	}

	return stdin, reader, 4
}

type toolCallTest struct {
	name    string
	tool    string
	args    json.RawMessage
	wantErr bool
}

func runToolTests(t *testing.T, stdin io.WriteCloser, reader *bufio.Reader, nextID int, tests []toolCallTest) {
	t.Helper()

	for i, tt := range tests {
		id := nextID + i
		t.Run(tt.name, func(t *testing.T) {
			err := send(stdin, mcpStdioRequest{
				JSONRPC: "2.0",
				ID:      ptr(id),
				Method:  "tools/call",
				Params:  json.RawMessage(`{"name":"` + tt.tool + `","arguments":` + string(tt.args) + `}`),
			})
			if err != nil {
				t.Fatal(err)
			}

			res, logs, err := recvResponse(reader, id, 3*time.Second)
			if err != nil {
				t.Fatal(err)
			}

			for _, l := range logs {
				t.Logf("Level: %s, Logger: %s, Data: %+v", l.Level, l.Logger, l.Data)
			}

			if res.Result != nil {
				var result mcpResult
				if err = json.Unmarshal(res.Result, &result); err != nil {
					t.Fatal(err)
				}
				if result.IsError && !tt.wantErr {
					t.Fatalf("tool returned unexpected error: %s", result.Content)
				}
				if !result.IsError && tt.wantErr {
					t.Fatalf("expected tool error, got success: %s", result.Content)
				}
			}
		})
	}
}

func TestToolCallsUserPassword(t *testing.T) {
	stdin, reader, nextID := setupMCPSession(t, nil)

	runToolTests(t, stdin, reader, nextID, []toolCallTest{
		{"get-connect-projects", "get-connect-projects", json.RawMessage(`{}`), false},
		{"search-users", "search-users", json.RawMessage(`{"criteria":"ramon"}`), false},
		{"get-my-delegations", "get-my-delegations", json.RawMessage(`{}`), false},
	})
}

func TestToolCallsAuthFailure(t *testing.T) {
	stdin, reader, nextID := setupMCPSession(t, map[string]string{
		"RI_USER":     "invalid-user",
		"RI_PASSWORD": "invalid-password",
	})

	runToolTests(t, stdin, reader, nextID, []toolCallTest{
		{"get-connect-projects", "get-connect-projects", json.RawMessage(`{}`), true},
	})
}

func TestToolCallsServiceIdentity(t *testing.T) {
	stdin, reader, nextID := setupMCPSession(t, map[string]string{
		"RI_USER":                        "",
		"RI_PASSWORD":                    "",
		"RI_SERVICE_IDENTITY_SECRET_KEY": os.Getenv("RI_SERVICE_IDENTITY_SECRET_KEY"),
	})

	runToolTests(t, stdin, reader, nextID, []toolCallTest{
		{"get-connect-projects", "get-connect-projects", json.RawMessage(`{}`), false},
	})
}

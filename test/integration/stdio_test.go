//go:build integration

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

		// Not our response — likely a log notification (no id) or a
		// stale response. Log it and keep reading within the same deadline.
		fmt.Printf("skipping non-matching message (id=%d): %+v\n", res.ID, res)
	}
}

func ptr[T any](v T) *T {
	return &v
}

func TestToolCalls(t *testing.T) {
	bin := os.Getenv("RI_MCP_BIN")
	if bin == "" {
		t.Fatal("RI_MCP_BIN environment variable not present")
	}

	cmd := exec.Command(bin)
	cmd.Env = []string{
		"RI_HOST=" + os.Getenv("RI_HOST"),
		"RI_USER=" + os.Getenv("RI_USER"),
		"RI_PASSWORD=" + os.Getenv("RI_PASSWORD"),
		"RI_LOG_LEVEL=" + os.Getenv("RI_LOG_LEVEL"),
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}

	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		stdin.Close()
		cmd.Process.Kill()
		cmd.Wait()
	})

	tests := []struct {
		name string
		req  mcpStdioRequest
	}{
		{
			name: "init",
			req: mcpStdioRequest{
				JSONRPC: "2.0",
				ID:      ptr(1),
				Method:  "initialize",
				Params:  json.RawMessage(`{ "protocolVersion": "2024-11-05", "clientInfo": {"name": "integration-test", "version": "0.0.0"}, "capabilities": {}}`),
			},
		},
		{
			name: "notification-init",
			req: mcpStdioRequest{
				JSONRPC: "2.0",
				Method:  "notifications/initialized",
				Params:  json.RawMessage(`{}`),
			},
		},
		{
			name: "set-log-level",
			req: mcpStdioRequest{
				JSONRPC: "2.0",
				ID:      ptr(2),
				Method:  "logging/setLevel",
				Params:  json.RawMessage(`{"level":"debug"}`),
			},
		},
		{
			name: "list-tools",
			req: mcpStdioRequest{
				JSONRPC: "2.0",
				ID:      ptr(3),
				Method:  "tools/list",
				Params:  json.RawMessage(`{}`),
			},
		},
		{
			name: "call-tool-get-connect-projects",
			req: mcpStdioRequest{
				JSONRPC: "2.0",
				ID:      ptr(4),
				Method:  "tools/call",
				Params:  json.RawMessage(`{"name":"get-connect-projects","arguments":{}}`),
			},
		},
		{
			name: "call-tool-search-users",
			req: mcpStdioRequest{
				JSONRPC: "2.0",
				ID:      ptr(5),
				Method:  "tools/call",
				Params:  json.RawMessage(`{"name":"search-users","arguments":{"criteria": "ramon"}}`),
			},
		},
		{
			name: "call-tool-get-my-delegations",
			req: mcpStdioRequest{
				JSONRPC: "2.0",
				ID:      ptr(6),
				Method:  "tools/call",
				Params:  json.RawMessage(`{"name":"get-my-delegations","arguments":{}}`),
			},
		},
	}

	reader := bufio.NewReader(stdout)

	for i, test := range tests {
		err := send(stdin, test.req)
		if err != nil {
			t.Fatal(err)
		}

		if test.req.ID != nil {
			res, logs, err := recvResponse(reader, *test.req.ID, time.Second*3)
			if err != nil {
				t.Fatal(err)
			}

			for _, l := range logs {
				t.Logf("Test: %d, Level: %s, Logger: %s, Data: %+v", i+1, l.Level, l.Logger, l.Data)
			}

			if res.Result != nil {
				var resultPayload mcpResult
				err = json.Unmarshal(res.Result, &resultPayload)
				if err != nil {
					t.Fatal(err)
				}

				if resultPayload.IsError {
					t.Logf("received error response %s", resultPayload.Content)
					t.Fatal()
				}
			}

			t.Logf("Test %d: ID: %d, result: %s", i+1, res.ID, res.Result)
		} else {
			t.Logf("Test %s does not have a response", test.name)
		}
	}
}

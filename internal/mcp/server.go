package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
)

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolCallResult struct {
	Content []toolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Serve runs the MCP JSON-RPC 2.0 server over stdio until stdin closes.
// repoPath is the default repository directory for all tool calls.
func Serve(repoPath string) error {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024) // 4 MB for large diffs
	enc := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			enc.Encode(rpcResponse{ //nolint:errcheck
				JSONRPC: "2.0",
				Error:   &rpcError{Code: -32700, Message: "parse error"},
			})
			continue
		}

		// Notifications have no ID — don't respond.
		if req.ID == nil || string(req.ID) == "null" {
			continue
		}

		result, rpcErr := dispatch(context.Background(), req, repoPath)
		resp := rpcResponse{JSONRPC: "2.0", ID: req.ID}
		if rpcErr != nil {
			resp.Error = rpcErr
		} else {
			resp.Result = result
		}
		enc.Encode(resp) //nolint:errcheck
	}
	return scanner.Err()
}

func dispatch(ctx context.Context, req rpcRequest, repoPath string) (any, *rpcError) {
	switch req.Method {
	case "initialize":
		return map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "git-next", "version": "1.0"},
		}, nil

	case "tools/list":
		list := make([]map[string]any, 0, len(Registry))
		for _, t := range Registry {
			list = append(list, map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"inputSchema": json.RawMessage(t.InputSchema),
			})
		}
		return map[string]any{"tools": list}, nil

	case "tools/call":
		var p struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return nil, &rpcError{Code: -32602, Message: "invalid params"}
		}
		var found *Tool
		for i := range Registry {
			if Registry[i].Name == p.Name {
				found = &Registry[i]
				break
			}
		}
		if found == nil {
			return nil, &rpcError{Code: -32601, Message: fmt.Sprintf("unknown tool: %s", p.Name)}
		}

		// repo_path argument overrides server default.
		resolvedPath := repoPath
		if rp, ok := p.Arguments["repo_path"].(string); ok && rp != "" {
			resolvedPath = rp
		}

		result, err := found.Handler(ctx, p.Arguments, resolvedPath)
		if err != nil {
			text, _ := json.Marshal(map[string]string{"error": err.Error()})
			return toolCallResult{
				Content: []toolContent{{Type: "text", Text: string(text)}},
				IsError: true,
			}, nil
		}
		text, _ := json.Marshal(result)
		return toolCallResult{
			Content: []toolContent{{Type: "text", Text: string(text)}},
		}, nil

	default:
		return nil, &rpcError{Code: -32601, Message: fmt.Sprintf("method not found: %s", req.Method)}
	}
}

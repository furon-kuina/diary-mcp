package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type MCPRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	ID      int    `json:"id"`
	Params  any    `json:"params,omitempty"`
}

type MCPResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Result  any           `json:"result,omitempty"`
	Error   ResponseError `json:"error,omitempty,omitzero"`
}

type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func main() {
	logFile, err := os.Create("/tmp/diary-mcp.log")
	if err != nil {
		panic(err)
	}
	defer logFile.Close()
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		received := sc.Bytes()
		fmt.Fprintln(logFile, "[DEBUG] received: ", string(received))
		var msg MCPRequest
		err := json.Unmarshal(received, &msg)
		if err != nil {
			fmt.Fprintf(logFile, "[ERROR] invalid message: %v\n", err)
			continue
		}
		response := MCPResponse{
			JSONRPC: "2.0",
			ID:      msg.ID,
		}
		switch msg.Method {
		case "initialize":
			response.Result = map[string]any{
				"capabilities": struct{}{},
				"serverInfo": map[string]string{
					"name":    "diary-mcp",
					"version": "0.0.1",
				},
				"protocolVersion": "2024-11-05",
			}
		case "notifications/initialized":
			fmt.Fprintf(logFile, "[DEBUG] initialized!\n")
		case "tools/list":
			response.Result = map[string]any{
				"tools": []map[string]any{
					{"name": "adder",
						"description": "Adds two numbers",
						"inputSchema": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"num1": map[string]any{
									"type": "number",
								},
								"num2": map[string]any{
									"type": "number",
								},
							},
							"required": []string{"num1", "num2"},
						},
					},
				},
				"nextCursor": "next-page-cursor",
			}
		case "tools/call":
			if msg.Params == nil {
				response.Error = ResponseError{
					Code:    -32600,
					Message: "params required",
				}
			}
			params, ok := msg.Params.(map[string]any)
			if !ok {
				response.Error = ResponseError{
					Code:    -32600,
					Message: "invalid request",
				}
				break
			}
			if _, ok := params["name"]; !ok {
				response.Error = ResponseError{
					Code:    -32600,
					Message: "no method name",
				}
				break
			}
			if params["name"] != "add" {
				response.Error = ResponseError{
					Code:    -32601,
					Message: "only add method is supported",
				}
				break
			}
			arguments, ok := params["arguments"].(map[string]any)
			if !ok {
				response.Error = ResponseError{
					Code:    -32602,
					Message: "arguments required",
				}
				break
			}
			num1, ok1 := arguments["num1"].(float64)
			num2, ok2 := arguments["num2"].(float64)
			if !ok1 || !ok2 {
				response.Error = ResponseError{
					Code:    -32602,
					Message: "num1 and num2 is required",
				}
				break
			}
			sum := num1 + num2
			response.Result = map[string]any{"content": []map[string]any{
				{
					"type": "text",
					"text": fmt.Sprintf("%f", sum),
				},
			}}
		default:
			fmt.Fprintf(logFile, "[ERROR] unknown method: %s\n", msg.Method)
			continue
		}
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			fmt.Fprintf(logFile, "[ERROR] failed to marshal response: %v\n", err)
			continue
		}
		fmt.Fprintf(logFile, "[DEBUG] sending response: %s\n", string(jsonResponse))
		if _, err := fmt.Fprintf(os.Stdout, "%s\n", jsonResponse); err != nil {
			return
		}
	}
}

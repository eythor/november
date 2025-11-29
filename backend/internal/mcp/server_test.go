package mcp

import (
	"encoding/json"
	"testing"
)

func TestHandleInitialize(t *testing.T) {
	server := &Server{}
	
	request := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "initialize",
		ID:      1,
	}
	
	reqBytes, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}
	
	response, err := server.HandleMessage(reqBytes)
	if err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}
	
	if response.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC 2.0, got %s", response.JSONRPC)
	}
	
	if response.ID.(float64) != 1 {
		t.Errorf("Expected ID 1, got %v", response.ID)
	}
	
	result, ok := response.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}
	
	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("Expected protocol version 2024-11-05, got %v", result["protocolVersion"])
	}
}

func TestHandleToolsList(t *testing.T) {
	server := &Server{}
	
	request := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/list",
		ID:      2,
	}
	
	reqBytes, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}
	
	response, err := server.HandleMessage(reqBytes)
	if err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}
	
	result, ok := response.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}
	
	tools, ok := result["tools"].([]map[string]interface{})
	if !ok {
		t.Fatal("Tools is not an array of maps")
	}
	
	expectedTools := []string{
		"lookup_patient",
		"schedule_appointment", 
		"cancel_appointment",
		"get_medical_history",
		"get_medication_info",
		"answer_health_question",
	}
	
	if len(tools) != len(expectedTools) {
		t.Errorf("Expected %d tools, got %d", len(expectedTools), len(tools))
	}
	
	for i, tool := range tools {
		if tool["name"] != expectedTools[i] {
			t.Errorf("Expected tool %s at position %d, got %s", expectedTools[i], i, tool["name"])
		}
	}
}

func TestHandleUnknownMethod(t *testing.T) {
	server := &Server{}
	
	request := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "unknown_method",
		ID:      3,
	}
	
	reqBytes, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}
	
	response, err := server.HandleMessage(reqBytes)
	if err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}
	
	if response.Error == nil {
		t.Fatal("Expected error for unknown method")
	}
	
	if response.Error.Code != -32601 {
		t.Errorf("Expected error code -32601, got %d", response.Error.Code)
	}
}
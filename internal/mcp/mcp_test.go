package mcp

import (
	"encoding/json"
	"os"
	"testing"

	"wiki/internal/config"
	"wiki/internal/database"
	"wiki/internal/repository"
)

func setupTestRepo(t *testing.T) (*repository.WikiRepository, func()) {
	dbFile := "test_mcp_wiki.db"
	os.Remove(dbFile)

	cfg := &config.Config{DBDriver: "sqlite", DBPath: dbFile}
	db, err := database.InitDB(cfg)
	if err != nil {
		t.Fatalf("Failed to init test db: %v", err)
	}

	repo := repository.NewWikiRepository(db)
	cleanup := func() {
		os.Remove(dbFile)
	}

	return repo, cleanup
}

func TestMCPInitialize(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	server := NewServer(repo)
	initReq := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
	}
	rawReq, _ := json.Marshal(initReq)

	resBytes, err := server.HandleRequest(rawReq, "test-user", true)
	if err != nil {
		t.Fatalf("Failed to handle initialize request: %v", err)
	}

	var res Response
	json.Unmarshal(resBytes, &res)

	if res.Error != nil {
		t.Fatalf("Initialize returned error: %v", res.Error)
	}

	rawRes, _ := json.Marshal(res.Result)
	var initRes InitializeResult
	json.Unmarshal(rawRes, &initRes)

	if initRes.ServerInfo.Name != "Pharatropic Wiki MCP Server" {
		t.Fatalf("Expected server name 'Pharatropic Wiki MCP Server', got '%s'", initRes.ServerInfo.Name)
	}
}

func TestMCPListTools(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	server := NewServer(repo)
	req := Request{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
	}
	rawReq, _ := json.Marshal(req)

	resBytes, _ := server.HandleRequest(rawReq, "test-user", true)
	var res Response
	json.Unmarshal(resBytes, &res)

	rawRes, _ := json.Marshal(res.Result)
	var listRes ListToolsResult
	json.Unmarshal(rawRes, &listRes)

	if len(listRes.Tools) == 0 {
		t.Fatalf("Expected tools in tools/list, got 0")
	}

	foundCreate := false
	for _, tool := range listRes.Tools {
		if tool.Name == "create_page" {
			foundCreate = true
			break
		}
	}

	if !foundCreate {
		t.Fatalf("Expected tool 'create_page' in tools list")
	}
}

func TestMCPCreateAndReadPageTool(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	server := NewServer(repo)

	// 1. Call create_page tool via MCP
	callParams := CallToolParams{
		Name: "create_page",
		Arguments: map[string]interface{}{
			"title":     "MCP Test Page",
			"content":   "# Created via MCP\nThis is an MCP test.",
			"summary":   "MCP Summary",
			"is_public": true,
			"tags":      []interface{}{"mcp", "test"},
		},
	}
	rawParams, _ := json.Marshal(callParams)

	req := Request{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params:  rawParams,
	}
	rawReq, _ := json.Marshal(req)

	resBytes, _ := server.HandleRequest(rawReq, "AI Assistant", true)
	var res Response
	json.Unmarshal(resBytes, &res)

	if res.Error != nil {
		t.Fatalf("tools/call create_page returned RPC error: %v", res.Error)
	}

	// 2. Call read_page tool via MCP
	readParams := CallToolParams{
		Name: "read_page",
		Arguments: map[string]interface{}{
			"slug": "mcp-test-page",
		},
	}
	rawReadParams, _ := json.Marshal(readParams)

	readReq := Request{
		JSONRPC: "2.0",
		ID:      4,
		Method:  "tools/call",
		Params:  rawReadParams,
	}
	rawReadReq, _ := json.Marshal(readReq)

	resReadBytes, _ := server.HandleRequest(rawReadReq, "AI Assistant", true)
	var readRes Response
	json.Unmarshal(resReadBytes, &readRes)

	if readRes.Error != nil {
		t.Fatalf("tools/call read_page returned RPC error: %v", readRes.Error)
	}
}

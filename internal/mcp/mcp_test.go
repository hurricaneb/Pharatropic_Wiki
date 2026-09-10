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

	resBytes, err := server.HandleRequest(rawReq, "test-user", true, false)
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

	resBytes, _ := server.HandleRequest(rawReq, "test-user", true, false)
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

	resBytes, _ := server.HandleRequest(rawReq, "AI Assistant", true, false)
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

	resReadBytes, _ := server.HandleRequest(rawReadReq, "AI Assistant", true, false)
	var readRes Response
	json.Unmarshal(resReadBytes, &readRes)

	if readRes.Error != nil {
		t.Fatalf("tools/call read_page returned RPC error: %v", readRes.Error)
	}
}

func TestMCPCreatePageWithParentSlug(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	server := NewServer(repo)

	createParent := func(name string) CallToolParams {
		return CallToolParams{
			Name: "create_page",
			Arguments: map[string]interface{}{
				"title":   name,
				"content": "root content",
			},
		}
	}
	callTool := func(id int, params CallToolParams) Response {
		rawParams, _ := json.Marshal(params)
		req := Request{JSONRPC: "2.0", ID: id, Method: "tools/call", Params: rawParams}
		rawReq, _ := json.Marshal(req)
		resBytes, _ := server.HandleRequest(rawReq, "AI Assistant", true, false)
		var res Response
		json.Unmarshal(resBytes, &res)
		return res
	}

	parentRes := callTool(10, createParent("MCP Parent Page"))
	if parentRes.Error != nil {
		t.Fatalf("create_page (parent) returned RPC error: %v", parentRes.Error)
	}

	childRes := callTool(11, CallToolParams{
		Name: "create_page",
		Arguments: map[string]interface{}{
			"title":       "MCP Child Page",
			"content":     "child content",
			"parent_slug": "mcp-parent-page",
		},
	})
	if childRes.Error != nil {
		t.Fatalf("create_page (child) returned RPC error: %v", childRes.Error)
	}

	rawChildRes, _ := json.Marshal(childRes.Result)
	var childToolRes CallToolResult
	json.Unmarshal(rawChildRes, &childToolRes)
	if childToolRes.IsError {
		t.Fatalf("Expected create_page with parent_slug to succeed, got tool error: %+v", childToolRes.Content)
	}

	page, err := repo.GetPageBySlug("mcp-child-page", true)
	if err != nil {
		t.Fatalf("Failed to fetch created child page: %v", err)
	}
	if page.ParentID == nil {
		t.Fatalf("Expected child page to have a ParentID set via MCP parent_slug")
	}

	// A second level of nesting must still be rejected through the MCP path
	grandchildRes := callTool(12, CallToolParams{
		Name: "create_page",
		Arguments: map[string]interface{}{
			"title":       "MCP Grandchild Page",
			"content":     "x",
			"parent_slug": "mcp-child-page",
		},
	})
	rawGrandchildRes, _ := json.Marshal(grandchildRes.Result)
	var grandchildToolRes CallToolResult
	json.Unmarshal(rawGrandchildRes, &grandchildToolRes)
	if !grandchildToolRes.IsError {
		t.Fatalf("Expected create_page to reject nesting a second level deep via MCP")
	}
}

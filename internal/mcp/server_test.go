package mcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"wiki/internal/models"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestHandleRequest_ParseError(t *testing.T) {
	repo := newIsolatedRepo(t)
	server := NewServer(repo)

	resBytes, err := server.HandleRequest([]byte("not json"), "tester", true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var res Response
	json.Unmarshal(resBytes, &res)
	if res.Error == nil || res.Error.Code != -32700 {
		t.Fatalf("Expected a parse error response, got %+v", res)
	}
}

func TestHandleRequest_Notification(t *testing.T) {
	repo := newIsolatedRepo(t)
	server := NewServer(repo)

	req := Request{JSONRPC: "2.0", Method: "notifications/initialized"}
	raw, _ := json.Marshal(req)

	resBytes, err := server.HandleRequest(raw, "tester", true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resBytes != nil {
		t.Fatalf("Expected no response for a notification, got %s", resBytes)
	}
}

func TestHandleRequest_Ping(t *testing.T) {
	repo := newIsolatedRepo(t)
	server := NewServer(repo)

	req := Request{JSONRPC: "2.0", ID: 1, Method: "ping"}
	raw, _ := json.Marshal(req)

	resBytes, _ := server.HandleRequest(raw, "tester", true, false)
	var res Response
	json.Unmarshal(resBytes, &res)
	if res.Error != nil {
		t.Fatalf("Expected no error for ping, got %+v", res.Error)
	}
}

func TestHandleRequest_UnknownMethod(t *testing.T) {
	repo := newIsolatedRepo(t)
	server := NewServer(repo)

	req := Request{JSONRPC: "2.0", ID: 1, Method: "does/not/exist"}
	raw, _ := json.Marshal(req)

	resBytes, _ := server.HandleRequest(raw, "tester", true, false)
	var res Response
	json.Unmarshal(resBytes, &res)
	if res.Error == nil || res.Error.Code != -32601 {
		t.Fatalf("Expected a method-not-found error, got %+v", res)
	}
}

func TestHandleRequest_ToolsCallInvalidParams(t *testing.T) {
	repo := newIsolatedRepo(t)
	server := NewServer(repo)

	req := Request{JSONRPC: "2.0", ID: 1, Method: "tools/call", Params: json.RawMessage(`"not an object"`)}
	raw, _ := json.Marshal(req)

	resBytes, _ := server.HandleRequest(raw, "tester", true, false)
	var res Response
	json.Unmarshal(resBytes, &res)
	if res.Error == nil || res.Error.Code != -32602 {
		t.Fatalf("Expected an invalid-params error, got %+v", res)
	}
}

func TestHandleRequest_ToolsCallRequiresAuthForWrites(t *testing.T) {
	repo := newIsolatedRepo(t)
	server := NewServer(repo)

	params := CallToolParams{Name: "create_page", Arguments: map[string]interface{}{"title": "X", "content": "y"}}
	rawParams, _ := json.Marshal(params)
	req := Request{JSONRPC: "2.0", ID: 1, Method: "tools/call", Params: rawParams}
	raw, _ := json.Marshal(req)

	resBytes, _ := server.HandleRequest(raw, "tester", false, false)
	var res Response
	json.Unmarshal(resBytes, &res)
	rawResult, _ := json.Marshal(res.Result)
	var toolRes CallToolResult
	json.Unmarshal(rawResult, &toolRes)
	if !toolRes.IsError {
		t.Fatal("Expected create_page to be rejected when not authenticated")
	}
}

func TestHandleRequest_ToolsCallBlockedByMustChangePassword(t *testing.T) {
	repo := newIsolatedRepo(t)
	server := NewServer(repo)

	params := CallToolParams{Name: "update_page", Arguments: map[string]interface{}{"slug": "x", "content": "y"}}
	rawParams, _ := json.Marshal(params)
	req := Request{JSONRPC: "2.0", ID: 1, Method: "tools/call", Params: rawParams}
	raw, _ := json.Marshal(req)

	resBytes, _ := server.HandleRequest(raw, "tester", true, true)
	var res Response
	json.Unmarshal(resBytes, &res)
	rawResult, _ := json.Marshal(res.Result)
	var toolRes CallToolResult
	json.Unmarshal(rawResult, &toolRes)
	if !toolRes.IsError {
		t.Fatal("Expected update_page to be blocked when the account must change its password")
	}
}

func TestHandleRequest_ToolsCallExecutionError(t *testing.T) {
	repo := newIsolatedRepo(t)
	server := NewServer(repo)

	// read_page with a valid slug string but the repo will report not-found
	// as a *tool* error (IsError), not an RPC error — exercise a case that
	// actually reaches ExecuteToolCall's success path with no error return.
	params := CallToolParams{Name: "list_tags"}
	rawParams, _ := json.Marshal(params)
	req := Request{JSONRPC: "2.0", ID: 1, Method: "tools/call", Params: rawParams}
	raw, _ := json.Marshal(req)

	resBytes, _ := server.HandleRequest(raw, "tester", true, false)
	var res Response
	json.Unmarshal(resBytes, &res)
	if res.Error != nil {
		t.Fatalf("Expected no RPC-level error, got %+v", res.Error)
	}
}

func TestHandleRequest_ResourcesList(t *testing.T) {
	repo := newIsolatedRepo(t)
	repo.CreatePage(&models.CreatePageRequest{Title: "Resurs", Content: "x"}, "tester")
	server := NewServer(repo)

	req := Request{JSONRPC: "2.0", ID: 1, Method: "resources/list"}
	raw, _ := json.Marshal(req)

	resBytes, _ := server.HandleRequest(raw, "tester", true, false)
	var res Response
	json.Unmarshal(resBytes, &res)
	if res.Error != nil {
		t.Fatalf("unexpected error: %+v", res.Error)
	}
}

func TestHandleRequest_ResourcesReadInvalidParams(t *testing.T) {
	repo := newIsolatedRepo(t)
	server := NewServer(repo)

	req := Request{JSONRPC: "2.0", ID: 1, Method: "resources/read", Params: json.RawMessage(`123`)}
	raw, _ := json.Marshal(req)

	resBytes, _ := server.HandleRequest(raw, "tester", true, false)
	var res Response
	json.Unmarshal(resBytes, &res)
	if res.Error == nil || res.Error.Code != -32602 {
		t.Fatalf("Expected an invalid-params error, got %+v", res)
	}
}

func TestHandleRequest_ResourcesReadError(t *testing.T) {
	repo := newIsolatedRepo(t)
	server := NewServer(repo)

	params := ReadResourceParams{URI: "wiki://pages/finns-inte"}
	rawParams, _ := json.Marshal(params)
	req := Request{JSONRPC: "2.0", ID: 1, Method: "resources/read", Params: rawParams}
	raw, _ := json.Marshal(req)

	resBytes, _ := server.HandleRequest(raw, "tester", true, false)
	var res Response
	json.Unmarshal(resBytes, &res)
	if res.Error == nil {
		t.Fatal("Expected an error reading a nonexistent resource")
	}
}

// --- GinHandler ---

func TestGinHandler_SSEStream(t *testing.T) {
	repo := newIsolatedRepo(t)
	server := NewServer(repo)

	router := gin.New()
	router.GET("/mcp", server.GinHandler())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/mcp", nil)
	router.ServeHTTP(w, req)

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("Expected an SSE content type, got %q", ct)
	}
}

func TestGinHandler_PostJSONRPC(t *testing.T) {
	repo := newIsolatedRepo(t)
	server := NewServer(repo)

	router := gin.New()
	router.POST("/mcp", server.GinHandler())

	body, _ := json.Marshal(Request{JSONRPC: "2.0", ID: 1, Method: "ping"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/mcp", bytes.NewReader(body))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGinHandler_PostNotificationReturnsNoContent(t *testing.T) {
	repo := newIsolatedRepo(t)
	server := NewServer(repo)

	router := gin.New()
	router.POST("/mcp", server.GinHandler())

	body, _ := json.Marshal(Request{JSONRPC: "2.0", Method: "notifications/initialized"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/mcp", bytes.NewReader(body))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("Expected 204, got %d", w.Code)
	}
}

type errReadCloser struct{}

func (errReadCloser) Read(p []byte) (int, error) { return 0, errors.New("simulated read failure") }
func (errReadCloser) Close() error               { return nil }

func TestGinHandler_BodyReadError(t *testing.T) {
	repo := newIsolatedRepo(t)
	server := NewServer(repo)

	router := gin.New()
	router.POST("/mcp", server.GinHandler())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/mcp", nil)
	req.Body = errReadCloser{}
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 for an unreadable body, got %d", w.Code)
	}
}

func TestGinHandler_DerivesAuthAndAuthorFromContext(t *testing.T) {
	repo := newIsolatedRepo(t)
	user, _ := repo.CreateUser("mcpuser", "mcpuser@example.com", "pw", "user")
	server := NewServer(repo)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", user)
		c.Set("username", user.Username)
		c.Next()
	})
	router.POST("/mcp", server.GinHandler())

	params := CallToolParams{Name: "create_page", Arguments: map[string]interface{}{"title": "Via HTTP MCP", "content": "x"}}
	rawParams, _ := json.Marshal(params)
	body, _ := json.Marshal(Request{JSONRPC: "2.0", ID: 1, Method: "tools/call", Params: rawParams})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/mcp", bytes.NewReader(body))
	router.ServeHTTP(w, req)

	var res Response
	json.Unmarshal(w.Body.Bytes(), &res)
	rawResult, _ := json.Marshal(res.Result)
	var toolRes CallToolResult
	json.Unmarshal(rawResult, &toolRes)
	if toolRes.IsError {
		t.Fatalf("Expected create_page to succeed for an authenticated, non-password-locked user, got: %+v", toolRes)
	}

	page, err := repo.GetPageBySlug("via-http-mcp", true)
	if err != nil {
		t.Fatalf("Expected page to be created: %v", err)
	}
	revs, _ := repo.GetRevisions(page.Slug)
	if len(revs) != 1 || revs[0].Author != "mcpuser" {
		t.Fatalf("Expected revision author to be the authenticated username, got %+v", revs)
	}
}

func TestGinHandler_BlocksWhenMustChangePassword(t *testing.T) {
	repo := newIsolatedRepo(t)
	user, _ := repo.CreateUser("lockeduser", "lockeduser@example.com", "pw", "user")
	user.MustChangePassword = true
	server := NewServer(repo)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", user)
		c.Set("username", user.Username)
		c.Next()
	})
	router.POST("/mcp", server.GinHandler())

	params := CallToolParams{Name: "create_page", Arguments: map[string]interface{}{"title": "Blocked", "content": "x"}}
	rawParams, _ := json.Marshal(params)
	body, _ := json.Marshal(Request{JSONRPC: "2.0", ID: 1, Method: "tools/call", Params: rawParams})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/mcp", bytes.NewReader(body))
	router.ServeHTTP(w, req)

	var res Response
	json.Unmarshal(w.Body.Bytes(), &res)
	rawResult, _ := json.Marshal(res.Result)
	var toolRes CallToolResult
	json.Unmarshal(rawResult, &toolRes)
	if !toolRes.IsError {
		t.Fatal("Expected create_page to be blocked for a user who must change their password")
	}
}

// --- ServeStdio ---

func TestServeStdio(t *testing.T) {
	repo := newIsolatedRepo(t)
	server := NewServer(repo)

	origStdin, origStdout := os.Stdin, os.Stdout
	t.Cleanup(func() { os.Stdin = origStdin; os.Stdout = origStdout })

	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdin pipe: %v", err)
	}
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	os.Stdin = inR
	os.Stdout = outW

	var outBuf bytes.Buffer
	outDone := make(chan struct{})
	go func() {
		io.Copy(&outBuf, outR)
		close(outDone)
	}()

	serveDone := make(chan error, 1)
	go func() {
		serveDone <- server.ServeStdio()
	}()

	// A blank line must be skipped, and a well-formed request must produce a response
	inW.Write([]byte("\n"))
	inW.Write([]byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n"))
	inW.Close() // EOF stops the ServeStdio loop

	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatalf("Expected ServeStdio to return nil on EOF, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for ServeStdio to return")
	}

	outW.Close()
	<-outDone

	if !bytes.Contains(outBuf.Bytes(), []byte(`"id":1`)) {
		t.Fatalf("Expected stdout to contain the ping response, got: %s", outBuf.String())
	}
}

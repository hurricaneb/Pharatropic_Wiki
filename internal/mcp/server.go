package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"wiki/internal/models"
	"wiki/internal/repository"

	"github.com/gin-gonic/gin"
)

type Server struct {
	repo *repository.WikiRepository
}

func NewServer(repo *repository.WikiRepository) *Server {
	return &Server{repo: repo}
}

func (s *Server) HandleRequest(rawReq []byte, author string, isAuthed bool, mustChangePassword bool) ([]byte, error) {
	var req Request
	if err := json.Unmarshal(rawReq, &req); err != nil {
		return json.Marshal(Response{
			JSONRPC: "2.0",
			Error:   &RPCError{Code: -32700, Message: "Parse error"},
		})
	}

	res := Response{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	switch req.Method {
	case "initialize":
		res.Result = InitializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities: ServerCapabilities{
				Tools:     &ToolsCapability{ListChanged: true},
				Resources: &ResourcesCapability{ListChanged: true},
			},
			ServerInfo: Implementation{
				Name:    "Pharatropic Wiki MCP Server",
				Version: "1.0.0",
			},
		}

	case "notifications/initialized", "initialized":
		return nil, nil // Notifications do not receive a response

	case "ping":
		res.Result = map[string]interface{}{}

	case "tools/list":
		res.Result = ListToolsResult{
			Tools: GetToolDefinitions(),
		}

	case "tools/call":
		var params CallToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			res.Error = &RPCError{Code: -32602, Message: "Invalid params"}
			break
		}

		// Enforce authentication check for write operations (create_page, update_page)
		if (params.Name == "create_page" || params.Name == "update_page") && !isAuthed {
			res.Result = CallToolResult{
				IsError: true,
				Content: []ToolContent{{
					Type: "text",
					Text: "Authentication required to create or update wiki pages. Please provide a valid API key via X-API-Key header.",
				}},
			}
			break
		}

		if (params.Name == "create_page" || params.Name == "update_page") && mustChangePassword {
			res.Result = CallToolResult{
				IsError: true,
				Content: []ToolContent{{
					Type: "text",
					Text: "This account must change its password before it can create or update wiki pages. Log in via the web UI to set a new password.",
				}},
			}
			break
		}

		toolRes, err := ExecuteToolCall(s.repo, params.Name, params.Arguments, author, isAuthed)
		if err != nil {
			res.Error = &RPCError{Code: -32603, Message: err.Error()}
			break
		}
		res.Result = toolRes

	case "resources/list":
		resources, err := GetResourceDefinitions(s.repo, isAuthed)
		if err != nil {
			res.Error = &RPCError{Code: -32603, Message: err.Error()}
			break
		}
		res.Result = ListResourcesResult{Resources: resources}

	case "resources/read":
		var params ReadResourceParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			res.Error = &RPCError{Code: -32602, Message: "Invalid params"}
			break
		}

		content, err := ReadResource(s.repo, params.URI, isAuthed)
		if err != nil {
			res.Error = &RPCError{Code: -32603, Message: err.Error()}
			break
		}
		res.Result = content

	default:
		res.Error = &RPCError{Code: -32601, Message: fmt.Sprintf("Method '%s' not found", req.Method)}
	}

	return json.Marshal(res)
}

// ServeStdio launches the MCP server reading stdin line-by-line and writing responses to stdout
func (s *Server) ServeStdio() error {
	log.Println("🤖 Starting Pharatropic Wiki MCP Server in Stdio mode...")
	reader := bufio.NewReader(os.Stdin)

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if len(bytesTrim(line)) == 0 {
			continue
		}

		respBytes, err := s.HandleRequest(line, "AI Assistant (Stdio MCP)", true, false)
		if err != nil {
			log.Printf("MCP Error: %v\n", err)
			continue
		}

		if respBytes != nil {
			os.Stdout.Write(respBytes)
			os.Stdout.Write([]byte("\n"))
		}
	}
}

func bytesTrim(b []byte) []byte {
	return []byte(string(b))
}

// GinHandler returns a Gin handler for HTTP & SSE MCP connections at /api/v1/mcp
func (s *Server) GinHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check authentication state from gin context
		author := c.GetString("username")
		if author == "" {
			author = "AI Assistant (HTTP MCP)"
		}
		userVal, existsUser := c.Get("user")
		_, existsUsername := c.Get("username")
		isAuthed := existsUser || existsUsername

		mustChangePassword := false
		if existsUser {
			if user, ok := userVal.(*models.User); ok {
				mustChangePassword = user.MustChangePassword
			}
		}

		// Support GET for SSE stream initialization
		if c.Request.Method == http.MethodGet {
			c.Header("Content-Type", "text/event-stream")
			c.Header("Cache-Control", "no-cache")
			c.Header("Connection", "keep-alive")
			c.Header("Access-Control-Allow-Origin", "*")

			c.SSEvent("endpoint", fmt.Sprintf("/api/v1/mcp?session_id=%d", c.Writer.Status()))
			c.Writer.Flush()
			return
		}

		// Handle POST for JSON-RPC 2.0 requests
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
			return
		}

		respBytes, err := s.HandleRequest(bodyBytes, author, isAuthed, mustChangePassword)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if respBytes == nil {
			c.Status(http.StatusNoContent)
			return
		}

		c.Data(http.StatusOK, "application/json", respBytes)
	}
}

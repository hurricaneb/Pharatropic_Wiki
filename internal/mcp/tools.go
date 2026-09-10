package mcp

import (
	"encoding/json"
	"fmt"
	"strings"

	"wiki/internal/models"
	"wiki/internal/repository"
)

func GetToolDefinitions() []Tool {
	return []Tool{
		{
			Name:        "list_pages",
			Description: "List all visible wiki pages in Pharatropic Wiki with titles, slugs, summaries, tags, view counts, and public/private status.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropertySchema{
					"search": {
						Type:        "string",
						Description: "Optional keyword to filter pages by title or content",
					},
					"tag": {
						Type:        "string",
						Description: "Optional tag name to filter pages by category",
					},
				},
			},
		},
		{
			Name:        "read_page",
			Description: "Read full markdown content, metadata, tags, and revision info for a specific wiki page by slug or title.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropertySchema{
					"slug": {
						Type:        "string",
						Description: "The URL slug or title of the wiki page to read (e.g. 'valkommen-till-wikin')",
					},
				},
				Required: []string{"slug"},
			},
		},
		{
			Name:        "search_pages",
			Description: "Perform full-text search across all wiki page titles and markdown contents.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropertySchema{
					"query": {
						Type:        "string",
						Description: "Search term or phrase to look for across the wiki",
					},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "create_page",
			Description: "Create a new wiki page in Pharatropic Wiki with Markdown content, title, tags, summary, and public/private status.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropertySchema{
					"title": {
						Type:        "string",
						Description: "Title of the new wiki page (e.g. 'Go API Architecture')",
					},
					"content": {
						Type:        "string",
						Description: "Full page content formatted in Markdown",
					},
					"summary": {
						Type:        "string",
						Description: "Short one-sentence summary of the page",
					},
					"is_public": {
						Type:        "boolean",
						Description: "Whether the page is publicly visible without login. Default is false (Private).",
					},
					"tags": {
						Type:        "array",
						Description: "List of tag names for category classification (e.g. ['api', 'go'])",
					},
					"parent_slug": {
						Type:        "string",
						Description: "Slug of an existing top-level page to create this page as a subpage of. Only one level of nesting is supported: the parent must itself be a top-level page. Omit to create a top-level page.",
					},
				},
				Required: []string{"title", "content"},
			},
		},
		{
			Name:        "update_page",
			Description: "Update the Markdown content or metadata of an existing wiki page and record a new revision.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropertySchema{
					"slug": {
						Type:        "string",
						Description: "Slug or title of the wiki page to update",
					},
					"content": {
						Type:        "string",
						Description: "New Markdown content for the page",
					},
					"title": {
						Type:        "string",
						Description: "Updated title for the page (optional)",
					},
					"summary": {
						Type:        "string",
						Description: "Updated summary (optional)",
					},
					"is_public": {
						Type:        "boolean",
						Description: "Updated public/private visibility status (optional)",
					},
					"comment": {
						Type:        "string",
						Description: "Reason or description for this edit revision (e.g. 'Added API section')",
					},
					"tags": {
						Type:        "array",
						Description: "Updated list of tags (optional)",
					},
					"parent_slug": {
						Type:        "string",
						Description: "Slug of an existing top-level page to move this page under. Pass an empty string to detach it back to a top-level page. Omit to leave the parent unchanged. The parent must itself be a top-level page (only one level of nesting is supported), and a page that already has subpages of its own cannot be given a parent.",
					},
				},
				Required: []string{"slug", "content"},
			},
		},
		{
			Name:        "get_backlinks",
			Description: "List all wiki pages that link to a specific wiki page (incoming [[WikiLinks]] or URL references).",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropertySchema{
					"slug": {
						Type:        "string",
						Description: "Slug or title of the target wiki page",
					},
				},
				Required: []string{"slug"},
			},
		},
		{
			Name:        "list_tags",
			Description: "List all existing tags and categories across Pharatropic Wiki.",
			InputSchema: InputSchema{
				Type: "object",
			},
		},
	}
}

func ExecuteToolCall(repo *repository.WikiRepository, name string, args map[string]interface{}, author string, isAuthed bool) (*CallToolResult, error) {
	if author == "" {
		author = "AI Assistant (MCP)"
	}

	switch name {
	case "list_pages":
		search, _ := args["search"].(string)
		tag, _ := args["tag"].(string)
		pages, err := repo.ListPages(search, tag, isAuthed)
		if err != nil {
			return &CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Error listing pages: %v", err)}}}, nil
		}

		jsonBytes, _ := json.MarshalIndent(pages, "", "  ")
		return &CallToolResult{Content: []ToolContent{{Type: "text", Text: string(jsonBytes)}}}, nil

	case "read_page":
		slugVal, _ := args["slug"].(string)
		if slugVal == "" {
			return &CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: "Parameter 'slug' is required"}}}, nil
		}

		page, err := repo.GetPageBySlug(slugVal, isAuthed)
		if err != nil {
			return &CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Error reading page: %v", err)}}}, nil
		}

		jsonBytes, _ := json.MarshalIndent(page, "", "  ")
		return &CallToolResult{Content: []ToolContent{{Type: "text", Text: string(jsonBytes)}}}, nil

	case "search_pages":
		query, _ := args["query"].(string)
		if query == "" {
			return &CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: "Parameter 'query' is required"}}}, nil
		}

		pages, err := repo.SearchPages(query, isAuthed)
		if err != nil {
			return &CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Error searching pages: %v", err)}}}, nil
		}

		jsonBytes, _ := json.MarshalIndent(pages, "", "  ")
		return &CallToolResult{Content: []ToolContent{{Type: "text", Text: string(jsonBytes)}}}, nil

	case "create_page":
		title, _ := args["title"].(string)
		content, _ := args["content"].(string)
		summary, _ := args["summary"].(string)
		
		if title == "" || content == "" {
			return &CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: "Parameters 'title' and 'content' are required"}}}, nil
		}

		var isPublic *bool
		if val, exists := args["is_public"].(bool); exists {
			isPublic = &val
		}

		var tags []string
		if rawTags, exists := args["tags"].([]interface{}); exists {
			for _, t := range rawTags {
				if str, ok := t.(string); ok && str != "" {
					tags = append(tags, str)
				}
			}
		}

		parentSlug, _ := args["parent_slug"].(string)

		req := models.CreatePageRequest{
			Title:      title,
			Content:    content,
			Summary:    summary,
			IsPublic:   isPublic,
			Tags:       tags,
			ParentSlug: parentSlug,
			Comment:    "Created via MCP Tool Call",
		}

		page, err := repo.CreatePage(&req, author)
		if err != nil {
			return &CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Error creating page: %v", err)}}}, nil
		}

		jsonBytes, _ := json.MarshalIndent(page, "", "  ")
		return &CallToolResult{Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Successfully created page '%s'!\n\n%s", page.Title, string(jsonBytes))}}}, nil

	case "update_page":
		slugVal, _ := args["slug"].(string)
		content, _ := args["content"].(string)
		title, _ := args["title"].(string)
		summary, _ := args["summary"].(string)
		comment, _ := args["comment"].(string)

		if slugVal == "" || content == "" {
			return &CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: "Parameters 'slug' and 'content' are required"}}}, nil
		}

		if comment == "" {
			comment = "Updated via MCP Tool Call"
		}

		var isPublic *bool
		if val, exists := args["is_public"].(bool); exists {
			isPublic = &val
		}

		var tags []string
		if rawTags, exists := args["tags"].([]interface{}); exists {
			for _, t := range rawTags {
				if str, ok := t.(string); ok && str != "" {
					tags = append(tags, str)
				}
			}
		}

		req := models.UpdatePageRequest{
			Title:    title,
			Content:  content,
			Summary:  summary,
			IsPublic: isPublic,
			Comment:  comment,
			Tags:     tags,
		}
		if val, exists := args["parent_slug"].(string); exists {
			req.ParentSlug = &val
		}

		page, err := repo.UpdatePage(slugVal, &req, author)
		if err != nil {
			return &CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Error updating page: %v", err)}}}, nil
		}

		jsonBytes, _ := json.MarshalIndent(page, "", "  ")
		return &CallToolResult{Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Successfully updated page '%s'!\n\n%s", page.Title, string(jsonBytes))}}}, nil

	case "get_backlinks":
		slugVal, _ := args["slug"].(string)
		if slugVal == "" {
			return &CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: "Parameter 'slug' is required"}}}, nil
		}

		backlinks, err := repo.GetBacklinks(slugVal, isAuthed)
		if err != nil {
			return &CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Error fetching backlinks: %v", err)}}}, nil
		}

		jsonBytes, _ := json.MarshalIndent(backlinks, "", "  ")
		return &CallToolResult{Content: []ToolContent{{Type: "text", Text: string(jsonBytes)}}}, nil

	case "list_tags":
		tags, err := repo.ListTags()
		if err != nil {
			return &CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Error listing tags: %v", err)}}}, nil
		}

		jsonBytes, _ := json.MarshalIndent(tags, "", "  ")
		return &CallToolResult{Content: []ToolContent{{Type: "text", Text: string(jsonBytes)}}}, nil

	default:
		return &CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Unknown tool '%s'", name)}}}, nil
	}
}

func GetResourceDefinitions(repo *repository.WikiRepository, isAuthed bool) ([]Resource, error) {
	pages, err := repo.ListPages("", "", isAuthed)
	if err != nil {
		return nil, err
	}

	var resources []Resource
	for _, p := range pages {
		resources = append(resources, Resource{
			URI:         fmt.Sprintf("wiki://pages/%s", p.Slug),
			Name:        p.Title,
			Description: p.Summary,
			MIMEType:    "text/markdown",
		})
	}

	return resources, nil
}

func ReadResource(repo *repository.WikiRepository, uri string, isAuthed bool) (*ReadResourceResult, error) {
	if !strings.HasPrefix(uri, "wiki://pages/") {
		return nil, fmt.Errorf("invalid resource URI '%s'", uri)
	}

	slugVal := strings.TrimPrefix(uri, "wiki://pages/")
	page, err := repo.GetPageBySlug(slugVal, isAuthed)
	if err != nil {
		return nil, err
	}

	return &ReadResourceResult{
		Contents: []ResourceContent{
			{
				URI:      uri,
				MIMEType: "text/markdown",
				Text:     page.Content,
			},
		},
	}, nil
}

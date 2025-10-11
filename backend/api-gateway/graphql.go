package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// =============================================================================
// GRAPHQL SCHEMA AND HANDLERS
// =============================================================================

// GraphQLSchema represents the GraphQL schema implementation
type GraphQLSchema struct {
	gateway *APIGateway
}

// NewGraphQLSchema creates a new GraphQL schema with resolvers
func NewGraphQLSchema(gateway *APIGateway) *GraphQLSchema {
	return &GraphQLSchema{
		gateway: gateway,
	}
}

// GraphQLRequest represents a GraphQL request
type GraphQLRequest struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables"`
	OperationName string                 `json:"operationName"`
}

// GraphQLResponse represents a GraphQL response
type GraphQLResponse struct {
	Data   interface{}    `json:"data,omitempty"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

// GraphQLError represents a GraphQL error
type GraphQLError struct {
	Message   string     `json:"message"`
	Path      []string   `json:"path,omitempty"`
	Locations []Location `json:"locations,omitempty"`
}

// Location represents error location in GraphQL query
type Location struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// GraphQLHandler handles GraphQL requests
func (gw *APIGateway) GraphQLHandler(c *gin.Context) {
	var req GraphQLRequest

	// Parse GraphQL request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, GraphQLResponse{
			Errors: []GraphQLError{{
				Message: "Invalid GraphQL request: " + err.Error(),
			}},
		})
		return
	}

	// Record metrics
	if gw.metrics != nil {
		operationType := extractOperationType(req.Query)
		gw.metrics.RecordGraphQLOperation(operationType, req.OperationName)
	}

	// Process GraphQL query
	response := gw.schema.ProcessQuery(c.Request.Context(), req)

	// Return response
	c.JSON(http.StatusOK, response)
}

// GraphQLPlaygroundHandler serves GraphQL Playground (development only)
func (gw *APIGateway) GraphQLPlaygroundHandler(c *gin.Context) {
	// Only allow playground in development mode
	if gin.Mode() != gin.DebugMode {
		c.JSON(http.StatusNotFound, gin.H{"error": "GraphQL Playground is only available in development mode"})
		return
	}

	html := `
<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<title>Nuclear AO3 GraphQL Playground</title>
	<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/static/css/index.css" />
	<link rel="shortcut icon" href="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/favicon.png" />
	<script src="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/static/js/middleware.js"></script>
</head>
<body>
	<div id="root">
		<style>
			body { background-color: rgb(23, 42, 58); font-family: Open Sans, sans-serif; height: 90vh; }
			#root { height: 100%; width: 100%; display: flex; align-items: center; justify-content: center; }
			.loading { font-size: 32px; font-weight: 200; color: rgba(255, 255, 255, .6); margin-left: 20px; }
			img { width: 78px; height: 78px; }
			.title { font-weight: 400; }
		</style>
		<img src="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/logo.png" alt="">
		<div class="loading"> Loading
			<span class="title">Nuclear AO3 GraphQL Playground</span>
		</div>
	</div>
	<script>
		window.addEventListener('load', function (event) {
			GraphQLPlayground.init(document.getElementById('root'), {
				endpoint: '/graphql',
				subscriptionEndpoint: 'ws://localhost:8080/graphql/ws'
			})
		})
	</script>
</body>
</html>`

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// GraphQLSubscriptionHandler handles WebSocket subscriptions
func (gw *APIGateway) GraphQLSubscriptionHandler(c *gin.Context) {
	// For now, return a placeholder - would need WebSocket implementation
	c.JSON(http.StatusNotImplemented, gin.H{
		"message":  "GraphQL subscriptions not yet implemented",
		"endpoint": "ws://localhost:8080/graphql/ws",
	})
}

// ProcessQuery processes a GraphQL query and returns response
func (schema *GraphQLSchema) ProcessQuery(ctx context.Context, req GraphQLRequest) GraphQLResponse {
	// This is a simplified implementation - in production would use graphql-go or similar

	// Extract operation type and field from query
	operationType := extractOperationType(req.Query)

	switch operationType {
	case "query":
		return schema.handleQuery(ctx, req)
	case "mutation":
		return schema.handleMutation(ctx, req)
	default:
		return GraphQLResponse{
			Errors: []GraphQLError{{
				Message: "Unsupported operation type: " + operationType,
			}},
		}
	}
}

// handleQuery processes GraphQL queries
func (schema *GraphQLSchema) handleQuery(ctx context.Context, req GraphQLRequest) GraphQLResponse {
	// Parse the query to understand what data is being requested
	query := strings.ToLower(req.Query)

	if strings.Contains(query, "works") {
		return schema.handleWorksQuery(ctx, req)
	} else if strings.Contains(query, "tags") {
		return schema.handleTagsQuery(ctx, req)
	} else if strings.Contains(query, "search") {
		return schema.handleSearchQuery(ctx, req)
	} else if strings.Contains(query, "user") {
		return schema.handleUserQuery(ctx, req)
	}

	// Default response for unrecognized queries
	return GraphQLResponse{
		Data: map[string]interface{}{
			"hello":   "Welcome to Nuclear AO3 GraphQL API",
			"version": "1.0.0",
		},
	}
}

// handleMutation processes GraphQL mutations
func (schema *GraphQLSchema) handleMutation(ctx context.Context, req GraphQLRequest) GraphQLResponse {
	// Parse the mutation to understand what operation is being requested
	query := strings.ToLower(req.Query)

	if strings.Contains(query, "auth") {
		return schema.handleAuthMutation(ctx, req)
	}

	return GraphQLResponse{
		Errors: []GraphQLError{{
			Message: "Mutation not implemented: " + req.OperationName,
		}},
	}
}

// handleWorksQuery handles work-related queries
func (schema *GraphQLSchema) handleWorksQuery(ctx context.Context, req GraphQLRequest) GraphQLResponse {
	// Proxy to work service
	response, err := schema.gateway.proxyToService("work", "GET", "/api/v1/works", nil)
	if err != nil {
		return GraphQLResponse{
			Errors: []GraphQLError{{
				Message: "Failed to fetch works: " + err.Error(),
			}},
		}
	}

	return GraphQLResponse{
		Data: map[string]interface{}{
			"works": response,
		},
	}
}

// handleTagsQuery handles tag-related queries
func (schema *GraphQLSchema) handleTagsQuery(ctx context.Context, req GraphQLRequest) GraphQLResponse {
	// Proxy to tag service
	response, err := schema.gateway.proxyToService("tag", "GET", "/api/v1/tags", nil)
	if err != nil {
		return GraphQLResponse{
			Errors: []GraphQLError{{
				Message: "Failed to fetch tags: " + err.Error(),
			}},
		}
	}

	return GraphQLResponse{
		Data: map[string]interface{}{
			"tags": response,
		},
	}
}

// handleSearchQuery handles search-related queries
func (schema *GraphQLSchema) handleSearchQuery(ctx context.Context, req GraphQLRequest) GraphQLResponse {
	// Proxy to search service
	response, err := schema.gateway.proxyToService("search", "GET", "/api/v1/search/works", nil)
	if err != nil {
		return GraphQLResponse{
			Errors: []GraphQLError{{
				Message: "Failed to perform search: " + err.Error(),
			}},
		}
	}

	return GraphQLResponse{
		Data: map[string]interface{}{
			"search": response,
		},
	}
}

// handleUserQuery handles user-related queries
func (schema *GraphQLSchema) handleUserQuery(ctx context.Context, req GraphQLRequest) GraphQLResponse {
	// Proxy to auth service
	response, err := schema.gateway.proxyToService("auth", "GET", "/api/v1/user/profile", nil)
	if err != nil {
		return GraphQLResponse{
			Errors: []GraphQLError{{
				Message: "Failed to fetch user: " + err.Error(),
			}},
		}
	}

	return GraphQLResponse{
		Data: map[string]interface{}{
			"user": response,
		},
	}
}

// handleAuthMutation handles authentication-related mutations
func (schema *GraphQLSchema) handleAuthMutation(ctx context.Context, req GraphQLRequest) GraphQLResponse {
	query := strings.ToLower(req.Query)

	if strings.Contains(query, "login") {
		return schema.handleLoginMutation(ctx, req)
	} else if strings.Contains(query, "register") {
		return schema.handleRegisterMutation(ctx, req)
	}

	return GraphQLResponse{
		Errors: []GraphQLError{{
			Message: "Unknown auth mutation",
		}},
	}
}

// handleLoginMutation handles login mutations
func (schema *GraphQLSchema) handleLoginMutation(ctx context.Context, req GraphQLRequest) GraphQLResponse {
	// Extract input from variables
	variables := req.Variables
	if variables == nil {
		return GraphQLResponse{
			Errors: []GraphQLError{{
				Message: "Variables required for login mutation",
			}},
		}
	}

	input, ok := variables["input"].(map[string]interface{})
	if !ok {
		return GraphQLResponse{
			Errors: []GraphQLError{{
				Message: "Input field required for login mutation",
			}},
		}
	}

	// Prepare login request body
	loginData := map[string]interface{}{
		"email":    input["email"],
		"password": input["password"],
	}

	// Convert to JSON for proxy request
	jsonData, err := json.Marshal(loginData)
	if err != nil {
		return GraphQLResponse{
			Errors: []GraphQLError{{
				Message: "Failed to marshal login data: " + err.Error(),
			}},
		}
	}

	// Proxy to auth service login endpoint
	response, err := schema.gateway.proxyToService("auth", "POST", "/api/v1/auth/login", strings.NewReader(string(jsonData)))
	if err != nil {
		return GraphQLResponse{
			Errors: []GraphQLError{{
				Message: "Login failed: " + err.Error(),
			}},
		}
	}

	// Extract token from response
	responseMap, ok := response.(map[string]interface{})
	if !ok {
		return GraphQLResponse{
			Errors: []GraphQLError{{
				Message: "Invalid response format from auth service",
			}},
		}
	}

	// Check for auth service errors
	if errorField, exists := responseMap["error"]; exists {
		return GraphQLResponse{
			Data: map[string]interface{}{
				"auth": map[string]interface{}{
					"login": map[string]interface{}{
						"token": nil,
						"errors": []map[string]interface{}{
							{
								"field":   "general",
								"message": fmt.Sprintf("%v", errorField),
							},
						},
					},
				},
			},
		}
	}

	// Return successful response
	return GraphQLResponse{
		Data: map[string]interface{}{
			"auth": map[string]interface{}{
				"login": map[string]interface{}{
					"token":  responseMap["access_token"],
					"errors": []interface{}{},
				},
			},
		},
	}
}

// handleRegisterMutation handles registration mutations
func (schema *GraphQLSchema) handleRegisterMutation(ctx context.Context, req GraphQLRequest) GraphQLResponse {
	// Extract input from variables
	variables := req.Variables
	if variables == nil {
		return GraphQLResponse{
			Errors: []GraphQLError{{
				Message: "Variables required for register mutation",
			}},
		}
	}

	input, ok := variables["input"].(map[string]interface{})
	if !ok {
		return GraphQLResponse{
			Errors: []GraphQLError{{
				Message: "Input field required for register mutation",
			}},
		}
	}

	// Prepare register request body
	registerData := map[string]interface{}{
		"email":    input["email"],
		"username": input["username"],
		"password": input["password"],
	}

	// Convert to JSON for proxy request
	jsonData, err := json.Marshal(registerData)
	if err != nil {
		return GraphQLResponse{
			Errors: []GraphQLError{{
				Message: "Failed to marshal register data: " + err.Error(),
			}},
		}
	}

	// Proxy to auth service register endpoint
	response, err := schema.gateway.proxyToService("auth", "POST", "/api/v1/auth/register", strings.NewReader(string(jsonData)))
	if err != nil {
		return GraphQLResponse{
			Errors: []GraphQLError{{
				Message: "Registration failed: " + err.Error(),
			}},
		}
	}

	// Extract response
	responseMap, ok := response.(map[string]interface{})
	if !ok {
		return GraphQLResponse{
			Errors: []GraphQLError{{
				Message: "Invalid response format from auth service",
			}},
		}
	}

	// Check for auth service errors
	if errorField, exists := responseMap["error"]; exists {
		return GraphQLResponse{
			Data: map[string]interface{}{
				"auth": map[string]interface{}{
					"register": map[string]interface{}{
						"token": nil,
						"errors": []map[string]interface{}{
							{
								"field":   "general",
								"message": fmt.Sprintf("%v", errorField),
							},
						},
					},
				},
			},
		}
	}

	// Return successful response
	return GraphQLResponse{
		Data: map[string]interface{}{
			"auth": map[string]interface{}{
				"register": map[string]interface{}{
					"token":  responseMap["access_token"],
					"errors": []interface{}{},
				},
			},
		},
	}
}

// proxyToService makes HTTP requests to microservices
func (gw *APIGateway) proxyToService(serviceName, method, path string, body io.Reader) (interface{}, error) {
	var serviceClient *ServiceClient

	switch serviceName {
	case "auth":
		serviceClient = gw.authService
	case "work":
		serviceClient = gw.workService
	case "tag":
		serviceClient = gw.tagService
	case "search":
		serviceClient = gw.searchService
	default:
		return nil, fmt.Errorf("unknown service: %s", serviceName)
	}

	url := serviceClient.BaseURL + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := serviceClient.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("service error %d: %s", resp.StatusCode, string(respBody))
	}

	var result interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// extractOperationType extracts the operation type from a GraphQL query
func extractOperationType(query string) string {
	query = strings.TrimSpace(strings.ToLower(query))

	if strings.HasPrefix(query, "mutation") {
		return "mutation"
	} else if strings.HasPrefix(query, "subscription") {
		return "subscription"
	}

	return "query" // Default to query
}

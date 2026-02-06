package workbench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"sort"
	"strings"
	"text/template"
	"time"
)

// TestGenerator generates Go test code from snapshots.
type TestGenerator struct {
	snapshots *SnapshotManager
	provider  string
}

// NewTestGenerator creates a new test generator.
func NewTestGenerator(snapshots *SnapshotManager, provider string) *TestGenerator {
	return &TestGenerator{
		snapshots: snapshots,
		provider:  provider,
	}
}

// GeneratedTest represents a generated test file.
type GeneratedTest struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
	Package  string `json:"package"`
}

// GeneratedFixture represents a generated fixture file.
type GeneratedFixture struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

// GenerateResult contains all generated files.
type GenerateResult struct {
	TestFile     GeneratedTest      `json:"test_file"`
	Fixtures     []GeneratedFixture `json:"fixtures"`
	Instructions string             `json:"instructions"`
}

// Generate generates test code and fixtures from all snapshots.
func (g *TestGenerator) Generate() (*GenerateResult, error) {
	snapshots := g.snapshots.List("", nil)
	if len(snapshots) == 0 {
		return nil, fmt.Errorf("no snapshots to generate tests from")
	}

	// Group snapshots by operation
	groups := g.snapshots.GroupByOperation()

	// Generate fixture files
	var fixtures []GeneratedFixture
	for _, group := range groups {
		for i, snap := range group.Snapshots {
			fixture := g.generateFixture(&snap, i)
			fixtures = append(fixtures, fixture)
		}
	}

	// Generate test file
	testFile, err := g.generateTestFile(groups)
	if err != nil {
		return nil, err
	}

	// Generate instructions
	instructions := g.generateInstructions()

	return &GenerateResult{
		TestFile:     testFile,
		Fixtures:     fixtures,
		Instructions: instructions,
	}, nil
}

func (g *TestGenerator) generateFixture(snap *Snapshot, index int) GeneratedFixture {
	// Create a clean response body for the fixture
	filename := fmt.Sprintf("%s_%d.json", sanitizeFilename(snap.Operation), index+1)
	
	// Try to pretty-print JSON body
	body := snap.Response.Body
	if json.Valid([]byte(body)) {
		var parsed interface{}
		if err := json.Unmarshal([]byte(body), &parsed); err == nil {
			if pretty, err := json.MarshalIndent(parsed, "", "  "); err == nil {
				body = string(pretty)
			}
		}
	}

	return GeneratedFixture{
		Filename: filename,
		Content:  body,
	}
}

func (g *TestGenerator) generateTestFile(groups []SnapshotGroup) (GeneratedTest, error) {
	data := testTemplateData{
		Package:     g.provider,
		Provider:    g.provider,
		ProviderCap: capitalizeFirst(g.provider),
		GeneratedAt: time.Now().Format(time.RFC3339),
		Groups:      make([]testGroupData, 0, len(groups)),
	}

	for _, group := range groups {
		groupData := testGroupData{
			Operation:    group.Operation,
			OperationCap: toGoName(group.Operation),
			Fixtures:     make([]testFixtureData, 0, len(group.Snapshots)),
		}

		for i, snap := range group.Snapshots {
			fixture := testFixtureData{
				Name:         snap.Name,
				Filename:     fmt.Sprintf("%s_%d.json", sanitizeFilename(snap.Operation), i+1),
				Method:       snap.Request.Method,
				URLPattern:   extractURLPattern(snap.Request.URL),
				StatusCode:   snap.Response.StatusCode,
				Description:  snap.Description,
			}
			groupData.Fixtures = append(groupData.Fixtures, fixture)
		}

		data.Groups = append(data.Groups, groupData)
	}

	// Sort groups for consistent output
	sort.Slice(data.Groups, func(i, j int) bool {
		return data.Groups[i].Operation < data.Groups[j].Operation
	})

	// Execute template
	tmpl, err := template.New("test").Parse(testFileTemplate)
	if err != nil {
		return GeneratedTest{}, fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return GeneratedTest{}, fmt.Errorf("failed to execute template: %w", err)
	}

	// Format the generated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// If formatting fails, return unformatted with a note
		return GeneratedTest{
			Filename: "client_test.go",
			Content:  "// WARNING: Code formatting failed, manual fixes needed\n" + buf.String(),
			Package:  g.provider,
		}, nil
	}

	return GeneratedTest{
		Filename: "client_test.go",
		Content:  string(formatted),
		Package:  g.provider,
	}, nil
}

func (g *TestGenerator) generateInstructions() string {
	connectorPath := fmt.Sprintf("internal/connectors/plugins/public/%s", g.provider)
	
	return fmt.Sprintf(`## Test Setup Instructions

1. Create the testdata directory:
   mkdir -p %s/testdata

2. Copy the fixture files to the testdata directory

3. Copy the test file to the connector directory:
   cp client_test.go %s/

4. Run the tests:
   go test -v ./%s/...

## What's Generated

- **Fixture files**: JSON responses captured from the real PSP API
- **Test file**: Go tests that mock HTTP responses and call your connector

## Customizing Tests

The generated tests use a mock HTTP server. You may need to:

1. Add authentication setup if your connector requires it
2. Adjust URL patterns if they don't match exactly
3. Add assertions for specific fields you care about

## Updating Fixtures

To update fixtures when the PSP API changes:

1. Run the workbench against the real API
2. Save new snapshots for changed endpoints
3. Regenerate tests
4. Review and commit the changes
`, connectorPath, connectorPath, connectorPath)
}

type testTemplateData struct {
	Package     string
	Provider    string
	ProviderCap string
	GeneratedAt string
	Groups      []testGroupData
}

type testGroupData struct {
	Operation    string
	OperationCap string
	Fixtures     []testFixtureData
}

type testFixtureData struct {
	Name        string
	Filename    string
	Method      string
	URLPattern  string
	StatusCode  int
	Description string
}

const testFileTemplate = `// Code generated by workbench. DO NOT EDIT.
// Generated at: {{.GeneratedAt}}

package {{.Package}}

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// loadFixture loads a test fixture from the testdata directory.
func loadFixture(t *testing.T, filename string) []byte {
	t.Helper()
	path := filepath.Join("testdata", filename)
	data, err := os.ReadFile(path)
	require.NoError(t, err, "failed to load fixture: %s", filename)
	return data
}

// setupMockServer creates a mock HTTP server that returns fixtures based on URL patterns.
func setupMockServer(t *testing.T, handlers map[string]http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for pattern, handler := range handlers {
			if matchURLPattern(r.URL.Path, pattern) {
				handler(w, r)
				return
			}
		}
		t.Logf("No handler for: %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	}))
}

// matchURLPattern checks if a URL path matches a pattern (simple wildcard matching).
func matchURLPattern(path, pattern string) bool {
	// Simple matching: exact match or prefix match with wildcard
	if pattern == path {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(path, strings.TrimSuffix(pattern, "*"))
	}
	return false
}

{{range .Groups}}
// Test{{$.ProviderCap}}{{.OperationCap}} tests the {{.Operation}} operation.
func Test{{$.ProviderCap}}{{.OperationCap}}(t *testing.T) {
	{{range .Fixtures}}
	t.Run("{{.Name}}", func(t *testing.T) {
		fixture := loadFixture(t, "{{.Filename}}")
		
		server := setupMockServer(t, map[string]http.HandlerFunc{
			"{{.URLPattern}}": func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "{{.Method}}", r.Method)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader({{.StatusCode}})
				w.Write(fixture)
			},
		})
		defer server.Close()

		// TODO: Initialize your connector client with server.URL
		// client := NewClient(server.URL, ...)
		
		// TODO: Call the connector method and assert results
		// result, err := client.{{$.OperationCap}}(...)
		// require.NoError(t, err)
		// require.NotNil(t, result)
		
		// Verify fixture is valid JSON
		var parsed interface{}
		err := json.Unmarshal(fixture, &parsed)
		require.NoError(t, err, "fixture should be valid JSON")
	})
	{{end}}
}
{{end}}
`

func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func toGoName(s string) string {
	// Convert operation name to Go-style name
	// e.g., "fetch_accounts" -> "FetchAccounts"
	parts := strings.Split(s, "_")
	for i, part := range parts {
		parts[i] = capitalizeFirst(part)
	}
	return strings.Join(parts, "")
}

func extractURLPattern(url string) string {
	// Extract path from URL for pattern matching
	// e.g., "https://api.stripe.com/v1/accounts" -> "/v1/accounts"
	
	// Find the path part
	idx := strings.Index(url, "://")
	if idx >= 0 {
		url = url[idx+3:]
	}
	idx = strings.Index(url, "/")
	if idx >= 0 {
		return url[idx:]
	}
	return "/"
}

// GenerateFixturesOnly generates only the fixture files (for updating).
func (g *TestGenerator) GenerateFixturesOnly() ([]GeneratedFixture, error) {
	snapshots := g.snapshots.List("", nil)
	if len(snapshots) == 0 {
		return nil, fmt.Errorf("no snapshots to generate fixtures from")
	}

	groups := g.snapshots.GroupByOperation()
	
	var fixtures []GeneratedFixture
	for _, group := range groups {
		for i, snap := range group.Snapshots {
			fixture := g.generateFixture(&snap, i)
			fixtures = append(fixtures, fixture)
		}
	}

	return fixtures, nil
}

// PreviewTest returns a preview of what would be generated.
func (g *TestGenerator) PreviewTest() (*GenerateResult, error) {
	return g.Generate()
}

package workbench

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
)

// Introspector provides code introspection capabilities for connectors.
type Introspector struct {
	provider    string
	connectorID models.ConnectorID
	basePath    string // Path to the connector's source directory
}

// NewIntrospector creates a new introspector for a connector.
func NewIntrospector(provider string, connectorID models.ConnectorID) *Introspector {
	// Try to find the connector source directory
	// Look in common locations relative to working directory
	// Check both new location (pkg/connectors/) and old location (internal/connectors/plugins/public/)
	possiblePaths := []string{
		// New location (pkg/connectors/)
		filepath.Join("pkg", "connectors", provider),
		filepath.Join("..", "pkg", "connectors", provider),
		filepath.Join("..", "..", "pkg", "connectors", provider),
		// Old location (internal/connectors/plugins/public/)
		filepath.Join("internal", "connectors", "plugins", "public", provider),
		filepath.Join("..", "internal", "connectors", "plugins", "public", provider),
		filepath.Join("..", "..", "internal", "connectors", "plugins", "public", provider),
	}

	var basePath string
	for _, p := range possiblePaths {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			basePath, _ = filepath.Abs(p)
			break
		}
	}

	return &Introspector{
		provider:    provider,
		connectorID: connectorID,
		basePath:    basePath,
	}
}

// FileInfo represents information about a source file.
type FileInfo struct {
	Name      string `json:"name"`
	Path      string `json:"path"`       // Relative path from connector root
	Size      int64  `json:"size"`
	IsDir     bool   `json:"is_dir"`
	Children  []FileInfo `json:"children,omitempty"`
}

// SourceFile represents the contents of a source file.
type SourceFile struct {
	Path     string   `json:"path"`
	Name     string   `json:"name"`
	Content  string   `json:"content"`
	Language string   `json:"language"`
	Lines    int      `json:"lines"`
	Symbols  []Symbol `json:"symbols,omitempty"`
}

// Symbol represents a code symbol (function, type, etc.)
type Symbol struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"` // "function", "type", "const", "var", "method"
	Line     int    `json:"line"`
	Doc      string `json:"doc,omitempty"`
	Signature string `json:"signature,omitempty"`
}

// ConnectorInfo provides overview information about a connector.
type ConnectorInfo struct {
	Provider     string            `json:"provider"`
	ConnectorID  string            `json:"connector_id"`
	Capabilities []CapabilityInfo  `json:"capabilities"`
	Config       []ConfigParam     `json:"config"`
	SourcePath   string            `json:"source_path,omitempty"`
	HasSource    bool              `json:"has_source"`
	Files        []FileInfo        `json:"files,omitempty"`
	Methods      []MethodInfo      `json:"methods,omitempty"`
}

// CapabilityInfo describes a connector capability.
type CapabilityInfo struct {
	Name        string `json:"name"`
	Supported   bool   `json:"supported"`
	Description string `json:"description,omitempty"`
}

// ConfigParam describes a configuration parameter.
type ConfigParam struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

// MethodInfo describes a plugin method.
type MethodInfo struct {
	Name        string `json:"name"`
	Implemented bool   `json:"implemented"`
	File        string `json:"file,omitempty"`
	Line        int    `json:"line,omitempty"`
}

// AllCapabilities returns all possible connector capabilities with descriptions.
var AllCapabilities = []CapabilityInfo{
	{Name: "FETCH_ACCOUNTS", Description: "Fetch accounts from the PSP"},
	{Name: "FETCH_BALANCES", Description: "Fetch account balances"},
	{Name: "FETCH_EXTERNAL_ACCOUNTS", Description: "Fetch external/beneficiary accounts"},
	{Name: "FETCH_PAYMENTS", Description: "Fetch payment transactions"},
	{Name: "FETCH_OTHERS", Description: "Fetch other data types"},
	{Name: "CREATE_TRANSFER", Description: "Create internal transfers"},
	{Name: "CREATE_PAYOUT", Description: "Create payouts to external accounts"},
	{Name: "CREATE_BANK_ACCOUNT", Description: "Create/link bank accounts"},
	{Name: "CREATE_WEBHOOKS", Description: "Register webhooks with PSP"},
	{Name: "TRANSLATE_WEBHOOK", Description: "Parse incoming webhooks"},
}

// GetInfo returns overview information about the connector.
func (i *Introspector) GetInfo() (*ConnectorInfo, error) {
	info := &ConnectorInfo{
		Provider:    i.provider,
		ConnectorID: i.connectorID.String(),
		HasSource:   i.basePath != "",
		SourcePath:  i.basePath,
	}

	// Get capabilities
	caps, err := registry.GetCapabilities(i.provider)
	if err == nil {
		capMap := make(map[string]bool)
		for _, c := range caps {
			capMap[c.String()] = true
		}
		for _, c := range AllCapabilities {
			c.Supported = capMap[c.Name]
			info.Capabilities = append(info.Capabilities, c)
		}
	}

	// Get config schema
	config, err := registry.GetConfig(i.provider)
	if err == nil {
		for name, param := range config {
			info.Config = append(info.Config, ConfigParam{
				Name:     name,
				Type:     string(param.DataType),
				Required: param.Required,
			})
		}
		// Sort by required first, then name
		sort.Slice(info.Config, func(i, j int) bool {
			if info.Config[i].Required != info.Config[j].Required {
				return info.Config[i].Required
			}
			return info.Config[i].Name < info.Config[j].Name
		})
	}

	// Get file tree if source is available
	if i.basePath != "" {
		files, err := i.GetFileTree()
		if err == nil {
			info.Files = files
		}

		// Try to find implemented methods
		methods := i.findImplementedMethods()
		info.Methods = methods
	}

	return info, nil
}

// GetFileTree returns the file tree for the connector source.
func (i *Introspector) GetFileTree() ([]FileInfo, error) {
	if i.basePath == "" {
		return nil, fmt.Errorf("source path not found")
	}

	return i.readDir("")
}

func (i *Introspector) readDir(relPath string) ([]FileInfo, error) {
	fullPath := filepath.Join(i.basePath, relPath)
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}

	var files []FileInfo
	for _, entry := range entries {
		name := entry.Name()
		
		// Skip hidden files and test files for cleaner view
		if strings.HasPrefix(name, ".") {
			continue
		}

		entryPath := filepath.Join(relPath, name)
		info, err := entry.Info()
		if err != nil {
			continue
		}

		fi := FileInfo{
			Name:  name,
			Path:  entryPath,
			Size:  info.Size(),
			IsDir: entry.IsDir(),
		}

		if entry.IsDir() {
			children, err := i.readDir(entryPath)
			if err == nil {
				fi.Children = children
			}
		}

		files = append(files, fi)
	}

	// Sort: directories first, then by name
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return files[i].Name < files[j].Name
	})

	return files, nil
}

// GetFile returns the contents of a source file.
func (i *Introspector) GetFile(relPath string) (*SourceFile, error) {
	if i.basePath == "" {
		return nil, fmt.Errorf("source path not found")
	}

	// Security: ensure path doesn't escape connector directory
	fullPath := filepath.Join(i.basePath, relPath)
	fullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(fullPath, i.basePath) {
		return nil, fmt.Errorf("invalid path")
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	sf := &SourceFile{
		Path:     relPath,
		Name:     filepath.Base(relPath),
		Content:  string(content),
		Lines:    strings.Count(string(content), "\n") + 1,
		Language: detectLanguage(relPath),
	}

	// Extract symbols for Go files
	if sf.Language == "go" {
		symbols := i.extractSymbols(fullPath, string(content))
		sf.Symbols = symbols
	}

	return sf, nil
}

func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".md":
		return "markdown"
	case ".sql":
		return "sql"
	default:
		return "text"
	}
}

func (i *Introspector) extractSymbols(path string, content string) []Symbol {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, content, parser.ParseComments)
	if err != nil {
		return nil
	}

	var symbols []Symbol

	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			sym := Symbol{
				Name: d.Name.Name,
				Kind: "function",
				Line: fset.Position(d.Pos()).Line,
			}
			if d.Recv != nil {
				sym.Kind = "method"
				// Get receiver type
				if len(d.Recv.List) > 0 {
					if star, ok := d.Recv.List[0].Type.(*ast.StarExpr); ok {
						if ident, ok := star.X.(*ast.Ident); ok {
							sym.Signature = fmt.Sprintf("(*%s) %s", ident.Name, d.Name.Name)
						}
					} else if ident, ok := d.Recv.List[0].Type.(*ast.Ident); ok {
						sym.Signature = fmt.Sprintf("(%s) %s", ident.Name, d.Name.Name)
					}
				}
			}
			if d.Doc != nil {
				sym.Doc = strings.TrimSpace(d.Doc.Text())
			}
			symbols = append(symbols, sym)

		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					sym := Symbol{
						Name: s.Name.Name,
						Kind: "type",
						Line: fset.Position(s.Pos()).Line,
					}
					if d.Doc != nil {
						sym.Doc = strings.TrimSpace(d.Doc.Text())
					}
					symbols = append(symbols, sym)

				case *ast.ValueSpec:
					kind := "var"
					if d.Tok == token.CONST {
						kind = "const"
					}
					for _, name := range s.Names {
						if name.Name == "_" {
							continue
						}
						symbols = append(symbols, Symbol{
							Name: name.Name,
							Kind: kind,
							Line: fset.Position(name.Pos()).Line,
						})
					}
				}
			}
		}
	}

	return symbols
}

// findImplementedMethods scans plugin.go to find which interface methods are implemented.
func (i *Introspector) findImplementedMethods() []MethodInfo {
	pluginMethods := []string{
		"Install", "Uninstall",
		"FetchNextAccounts", "FetchNextPayments", "FetchNextBalances",
		"FetchNextExternalAccounts", "FetchNextOthers",
		"CreateBankAccount", "CreateTransfer", "ReverseTransfer",
		"PollTransferStatus", "CreatePayout", "ReversePayout", "PollPayoutStatus",
		"CreateWebhooks", "TranslateWebhook", "VerifyWebhook", "TrimWebhook",
		"CreateUser", "CreateUserLink", "CompleteUserLink", "UpdateUserLink",
		"DeleteUser", "DeleteUserConnection",
	}

	methods := make([]MethodInfo, len(pluginMethods))
	for i, m := range pluginMethods {
		methods[i] = MethodInfo{
			Name:        m,
			Implemented: false,
		}
	}

	if i.basePath == "" {
		return methods
	}

	// Scan all Go files for method implementations
	filepath.Walk(i.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, content, 0)
		if err != nil {
			return nil
		}

		relPath, _ := filepath.Rel(i.basePath, path)

		for _, decl := range f.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv != nil {
				// Check if this is a method on Plugin type
				isPluginMethod := false
				if len(fn.Recv.List) > 0 {
					if star, ok := fn.Recv.List[0].Type.(*ast.StarExpr); ok {
						if ident, ok := star.X.(*ast.Ident); ok {
							if ident.Name == "Plugin" {
								isPluginMethod = true
							}
						}
					}
				}

				if isPluginMethod {
					for j := range methods {
						if methods[j].Name == fn.Name.Name {
							methods[j].Implemented = true
							methods[j].File = relPath
							methods[j].Line = fset.Position(fn.Pos()).Line
							break
						}
					}
				}
			}
		}

		return nil
	})

	return methods
}

// SearchCode searches for a pattern in the connector source code.
func (i *Introspector) SearchCode(pattern string) ([]SearchResult, error) {
	if i.basePath == "" {
		return nil, fmt.Errorf("source path not found")
	}

	pattern = strings.ToLower(pattern)
	var results []SearchResult

	filepath.Walk(i.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		relPath, _ := filepath.Rel(i.basePath, path)
		lines := strings.Split(string(content), "\n")

		for lineNum, line := range lines {
			if strings.Contains(strings.ToLower(line), pattern) {
				results = append(results, SearchResult{
					File:    relPath,
					Line:    lineNum + 1,
					Content: strings.TrimSpace(line),
				})
				if len(results) >= 100 {
					return filepath.SkipAll
				}
			}
		}

		return nil
	})

	return results, nil
}

// SearchResult represents a code search result.
type SearchResult struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Content string `json:"content"`
}

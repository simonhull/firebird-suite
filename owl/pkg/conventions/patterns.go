package conventions

import (
	"strings"

	"github.com/simonhull/firebird-suite/owl/pkg/analyzer"
)

// Pattern represents an observable naming or structural pattern
type Pattern struct {
	ID            string
	Name          string   // "Handler" (for classification)
	DisplayName   string   // "Handler Suffix Pattern" (for docs)
	Description   string
	Category      string   // "handlers", "services", etc. (for grouping)
	Confidence    float64  // 0.0-1.0 (how sure are we?)
	Tags          []string
	Examples      []string
	MatchType     func(*analyzer.Type) bool
	MatchFunction func(*analyzer.Function) bool
}

// DefaultPatterns returns observable patterns with high-confidence name-based patterns
func DefaultPatterns() []Pattern {
	return []Pattern{
		// High-confidence naming patterns
		{
			ID:          "suffix-handler",
			Name:        "Handler",
			DisplayName: "Handler Suffix",
			Description: "Types with names ending in 'Handler'",
			Category:    "handlers",
			Confidence:  0.95,
			Tags:        []string{"naming-pattern", "suffix"},
			Examples:    []string{"UserHandler", "PostHandler", "AuthHandler"},
			MatchType: func(t *analyzer.Type) bool {
				return t.Kind == "struct" && strings.HasSuffix(t.Name, "Handler")
			},
		},
		{
			ID:          "suffix-service",
			Name:        "Service",
			DisplayName: "Service Suffix",
			Description: "Types with names ending in 'Service'",
			Category:    "services",
			Confidence:  0.95,
			Tags:        []string{"naming-pattern", "suffix"},
			Examples:    []string{"UserService", "EmailService", "AuthService"},
			MatchType: func(t *analyzer.Type) bool {
				return t.Kind == "struct" && strings.HasSuffix(t.Name, "Service")
			},
		},
		{
			ID:          "suffix-repository",
			Name:        "Repository",
			DisplayName: "Repository Suffix",
			Description: "Types ending in 'Repository' or 'Store'",
			Category:    "repositories",
			Confidence:  0.95,
			Tags:        []string{"naming-pattern", "suffix", "data-access"},
			Examples:    []string{"UserRepository", "DataStore", "CacheStore"},
			MatchType: func(t *analyzer.Type) bool {
				return (t.Kind == "struct" || t.Kind == "interface") &&
					(strings.HasSuffix(t.Name, "Repository") || strings.HasSuffix(t.Name, "Store"))
			},
		},
		{
			ID:          "suffix-generator",
			Name:        "Generator",
			DisplayName: "Generator Suffix",
			Description: "Types with names ending in 'Generator' or 'Builder'",
			Category:    "generators",
			Confidence:  0.95,
			Tags:        []string{"naming-pattern", "suffix", "code-generation"},
			Examples:    []string{"HandlerGenerator", "ModelBuilder", "CodeGenerator"},
			MatchType: func(t *analyzer.Type) bool {
				return t.Kind == "struct" &&
					(strings.HasSuffix(t.Name, "Generator") || strings.HasSuffix(t.Name, "Builder"))
			},
		},
		{
			ID:          "suffix-config",
			Name:        "Config",
			DisplayName: "Config Suffix",
			Description: "Types with names ending in 'Config', 'Options', or 'Settings'",
			Category:    "config",
			Confidence:  0.95,
			Tags:        []string{"naming-pattern", "suffix", "configuration"},
			Examples:    []string{"AppConfig", "ServerOptions", "DatabaseSettings"},
			MatchType: func(t *analyzer.Type) bool {
				return t.Kind == "struct" &&
					(strings.HasSuffix(t.Name, "Config") ||
						strings.HasSuffix(t.Name, "Options") ||
						strings.HasSuffix(t.Name, "Settings"))
			},
		},
		{
			ID:          "suffix-dto",
			Name:        "DTO",
			DisplayName: "DTO Suffix",
			Description: "Types with names ending in 'Request', 'Response', 'Input', 'Output', or 'DTO'",
			Category:    "dtos",
			Confidence:  0.95,
			Tags:        []string{"naming-pattern", "suffix", "data-transfer"},
			Examples:    []string{"CreateUserRequest", "UserResponse", "LoginInput"},
			MatchType: func(t *analyzer.Type) bool {
				return t.Kind == "struct" &&
					(strings.HasSuffix(t.Name, "Request") ||
						strings.HasSuffix(t.Name, "Response") ||
						strings.HasSuffix(t.Name, "Input") ||
						strings.HasSuffix(t.Name, "Output") ||
						strings.HasSuffix(t.Name, "DTO"))
			},
		},
		{
			ID:          "suffix-middleware",
			Name:        "Middleware",
			DisplayName: "Middleware Suffix",
			Description: "Types with names ending in 'Middleware' or containing 'middleware'",
			Category:    "middleware",
			Confidence:  0.95,
			Tags:        []string{"naming-pattern", "suffix", "http"},
			Examples:    []string{"AuthMiddleware", "LoggingMiddleware"},
			MatchType: func(t *analyzer.Type) bool {
				return t.Kind == "struct" &&
					(strings.HasSuffix(t.Name, "Middleware") ||
						strings.Contains(strings.ToLower(t.Name), "middleware"))
			},
		},
		{
			ID:          "suffix-parser",
			Name:        "Parser",
			DisplayName: "Parser/Analyzer Suffix",
			Description: "Types with names ending in 'Parser', 'Analyzer', 'Lexer', or 'Scanner'",
			Category:    "parsers",
			Confidence:  0.95,
			Tags:        []string{"naming-pattern", "suffix", "code-analysis"},
			Examples:    []string{"JSONParser", "CodeAnalyzer", "TokenLexer"},
			MatchType: func(t *analyzer.Type) bool {
				name := strings.ToLower(t.Name)
				return t.Kind == "struct" &&
					(strings.HasSuffix(name, "parser") ||
						strings.HasSuffix(name, "analyzer") ||
						strings.HasSuffix(name, "lexer") ||
						strings.HasSuffix(name, "scanner"))
			},
		},
		{
			ID:          "suffix-template-data",
			Name:        "Template Data",
			DisplayName: "Template Data Suffix",
			Description: "Types with names ending in 'Data' or 'TemplateData'",
			Category:    "templates",
			Confidence:  0.90,
			Tags:        []string{"naming-pattern", "suffix", "templates"},
			Examples:    []string{"HandlerData", "ServiceTemplateData"},
			MatchType: func(t *analyzer.Type) bool {
				return t.Kind == "struct" &&
					(strings.HasSuffix(t.Name, "TemplateData") ||
						(strings.HasSuffix(t.Name, "Data") && !strings.HasSuffix(t.Name, "MetaData")))
			},
		},
		{
			ID:          "suffix-model",
			Name:        "Model",
			DisplayName: "Database Model",
			Description: "Structs with database tags (db:, gorm:, bun:)",
			Category:    "models",
			Confidence:  0.98,
			Tags:        []string{"structural-pattern", "database"},
			Examples:    []string{"User (with db tags)", "Post (with gorm tags)"},
			MatchType: func(t *analyzer.Type) bool {
				if t.Kind != "struct" {
					return false
				}
				for _, field := range t.Fields {
					if strings.Contains(field.Tag, "db:") ||
						strings.Contains(field.Tag, "gorm:") ||
						strings.Contains(field.Tag, "bun:") {
						return true
					}
				}
				return false
			},
		},
		{
			ID:          "prefix-validate",
			Name:        "Validator",
			DisplayName: "Validate Prefix",
			Description: "Functions with names starting with 'Validate', 'Verify', or 'Check'",
			Category:    "validators",
			Confidence:  0.92,
			Tags:        []string{"naming-pattern", "prefix", "validation"},
			Examples:    []string{"ValidateUser", "VerifyToken", "CheckPermission"},
			MatchFunction: func(f *analyzer.Function) bool {
				name := f.Name
				return strings.HasPrefix(name, "Validate") ||
					strings.HasPrefix(name, "Verify") ||
					strings.HasPrefix(name, "Check")
			},
		},

		// Behavioral patterns (slightly lower confidence)
		{
			ID:          "cobra-command",
			Name:        "Command",
			DisplayName: "Cobra Command Function",
			Description: "Functions returning *cobra.Command (CLI commands)",
			Category:    "commands",
			Confidence:  0.98,
			Tags:        []string{"behavioral-pattern", "cli", "cobra"},
			Examples:    []string{"GenerateCmd()", "NewCmd()", "ServeCmd()"},
			MatchFunction: func(f *analyzer.Function) bool {
				if len(f.Returns) == 0 {
					return false
				}
				return strings.Contains(f.Returns[0].Type, "cobra.Command")
			},
		},
		{
			ID:          "http-handler-func",
			Name:        "HTTP Handler",
			DisplayName: "HTTP Handler Signature",
			Description: "Functions matching http.HandlerFunc signature (ResponseWriter, *Request)",
			Category:    "handlers",
			Confidence:  0.98,
			Tags:        []string{"behavioral-pattern", "http"},
			Examples:    []string{"HandleUser(w, r)", "ServeHTTP(w, r)"},
			MatchFunction: func(f *analyzer.Function) bool {
				if len(f.Parameters) != 2 || len(f.Returns) != 0 {
					return false
				}
				return strings.Contains(f.Parameters[0].Type, "ResponseWriter") &&
					strings.Contains(f.Parameters[1].Type, "Request")
			},
		},
	}
}

// RegisterDefaultPatterns registers all default observation patterns with a registry
func RegisterDefaultPatterns(r *Registry) {
	for _, pattern := range DefaultPatterns() {
		r.Register(&Pattern{
			ID:            pattern.ID,
			Name:          pattern.Name,
			DisplayName:   pattern.DisplayName,
			Description:   pattern.Description,
			Category:      pattern.Category,
			Confidence:    pattern.Confidence,
			Tags:          pattern.Tags,
			Examples:      pattern.Examples,
			MatchType:     pattern.MatchType,
			MatchFunction: pattern.MatchFunction,
		})
	}
}

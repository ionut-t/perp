package server

import (
	"fmt"
	"net/url"
	"strings"
)

// ParsedURI represents the components of a parsed connection URI
type ParsedURI struct {
	Protocol string
	Username string
	Password string
	Host     string
	Port     string
	Database string
}

// DefaultPort is the default PostgreSQL port
const DefaultPort = "5432"

// ParseConnectionURI parses a database connection URI into its components.
// Accepts any URI scheme as long as it contains the required components.
// Supports formats like:
//   - postgresql://user:pass@host:5432/dbname
//   - postgres://user:pass@host:5432/dbname
//   - customscheme://user:pass@host:5432/dbname
//   - postgres://user@host:5432/dbname (no password)
//   - postgres://user:pass@host/dbname (default port 5432)
func ParseConnectionURI(uri string) (*ParsedURI, error) {
	if strings.TrimSpace(uri) == "" {
		return nil, fmt.Errorf("connection URI cannot be empty")
	}

	// Parse the URI
	parsedURL, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid URI format: %w", err)
	}

	// Validate scheme/protocol exists
	if parsedURL.Scheme == "" {
		return nil, fmt.Errorf("missing protocol (e.g., postgresql://)")
	}

	protocol := strings.ToLower(parsedURL.Scheme)

	// Check if this looks like a relative URL (no username info)
	// This happens when parsing "user:pass@localhost" without protocol
	if parsedURL.Host == "" && parsedURL.User == nil {
		return nil, fmt.Errorf("invalid URI format: missing host information")
	}

	// Extract username
	username := ""
	password := ""
	if parsedURL.User != nil {
		username = parsedURL.User.Username()
		if pass, ok := parsedURL.User.Password(); ok {
			password = pass
		}
	}

	if username == "" {
		return nil, fmt.Errorf("missing username in URI")
	}

	// Extract host
	host := parsedURL.Hostname()
	if host == "" {
		return nil, fmt.Errorf("missing host in URI")
	}

	// Extract port or use default PostgreSQL port
	port := parsedURL.Port()
	if port == "" {
		port = DefaultPort
	}

	// Extract database name
	database := strings.TrimPrefix(parsedURL.Path, "/")
	if database == "" {
		return nil, fmt.Errorf("missing database name in URI")
	}

	// Remove query parameters from database name if present
	if idx := strings.Index(database, "?"); idx != -1 {
		database = database[:idx]
	}

	return &ParsedURI{
		Protocol: protocol,
		Username: username,
		Password: password,
		Host:     host,
		Port:     port,
		Database: database,
	}, nil
}

// ToCreateServer converts a ParsedURI to a CreateServer struct
func (p *ParsedURI) ToCreateServer(name string, shareDatabaseSchemaLLM bool) CreateServer {
	return CreateServer{
		Name:                   name,
		Address:                p.Host,
		Port:                   p.Port,
		Username:               p.Username,
		Password:               p.Password,
		Database:               p.Database,
		ShareDatabaseSchemaLLM: shareDatabaseSchemaLLM,
	}
}

// String returns a string representation of the parsed URI (with masked password)
func (p *ParsedURI) String() string {
	passwordDisplay := MaskedPassword
	if p.Password == "" {
		passwordDisplay = ""
	}

	return fmt.Sprintf("%s://%s:%s@%s:%s/%s",
		p.Protocol,
		p.Username,
		passwordDisplay,
		p.Host,
		p.Port,
		p.Database,
	)
}

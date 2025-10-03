package migrate

import (
	"net/url"
	"strings"
)

// MaskPassword masks the password in a database connection string
// Supports PostgreSQL URLs, MySQL DSNs, and SQLite paths
func MaskPassword(connectionString string) string {
	// Special case: mysql://user:pass@tcp(...) format from connection.go
	if strings.HasPrefix(connectionString, "mysql://") && strings.Contains(connectionString, "@tcp(") {
		// Remove the mysql:// prefix and treat it like a DSN
		dsn := strings.TrimPrefix(connectionString, "mysql://")
		masked := maskMySQLDSN(dsn)
		return "mysql://" + masked
	}

	// URL format: postgres://user:pass@host:port/db
	if strings.Contains(connectionString, "://") {
		return maskURLPassword(connectionString)
	}

	// MySQL DSN format: user:pass@tcp(host:port)/db
	if strings.Contains(connectionString, "@tcp(") || strings.Contains(connectionString, "@unix(") {
		return maskMySQLDSN(connectionString)
	}

	// SQLite (no password) or unrecognized format
	return connectionString
}

// maskURLPassword masks passwords in URL-formatted connection strings
func maskURLPassword(connStr string) string {
	parsedURL, err := url.Parse(connStr)
	if err != nil {
		// If parsing fails, use fallback masking
		return maskSimple(connStr)
	}

	if parsedURL.User != nil {
		username := parsedURL.User.Username()
		if _, hasPassword := parsedURL.User.Password(); hasPassword {
			// Manually construct the URL to avoid encoding the asterisks
			scheme := parsedURL.Scheme
			host := parsedURL.Host
			path := parsedURL.Path
			query := parsedURL.RawQuery
			fragment := parsedURL.Fragment

			result := scheme + "://" + username + ":****@" + host + path
			if query != "" {
				result += "?" + query
			}
			if fragment != "" {
				result += "#" + fragment
			}
			return result
		}
	}

	return parsedURL.String()
}

// maskMySQLDSN masks passwords in MySQL DSN format
func maskMySQLDSN(dsn string) string {
	// MySQL: user:pass@tcp(host:port)/db or user:pass@unix(/path)/db
	colonIndex := strings.Index(dsn, ":")

	// Find the @ that comes after the password (not in password itself)
	atIndex := -1
	for i := colonIndex + 1; i < len(dsn); i++ {
		if dsn[i] == '@' {
			// Check if this is followed by tcp( or unix( to confirm it's the connection @
			if i+4 <= len(dsn) && dsn[i+1:i+4] == "tcp" {
				atIndex = i
				break
			}
			if i+5 <= len(dsn) && dsn[i+1:i+5] == "unix" {
				atIndex = i
				break
			}
		}
	}

	if colonIndex == -1 || atIndex == -1 || atIndex < colonIndex {
		return dsn
	}

	return dsn[:colonIndex+1] + "****" + dsn[atIndex:]
}

// maskSimple is a fallback that hides everything between : and @
func maskSimple(s string) string {
	parts := strings.Split(s, ":")
	if len(parts) < 3 {
		return s
	}

	atIndex := strings.Index(parts[2], "@")
	if atIndex == -1 {
		return s
	}

	return parts[0] + ":" + parts[1] + ":****@" + parts[2][atIndex+1:]
}

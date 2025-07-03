package db

import (
	"strings"
)

// stripSQLComments removes SQL comments from a query string, correctly handling
// various string literal and comment formats, including PostgreSQL-specific syntax.
func stripSQLComments(q string) string {
	var sb strings.Builder
	sb.Grow(len(q))

	for i := 0; i < len(q); {
		c := q[i]

		// Single-quoted strings
		if c == '\'' {
			sb.WriteByte(c)
			i++
			for i < len(q) {
				sc := q[i]
				sb.WriteByte(sc)
				i++
				if sc == '\'' {
					if i < len(q) && q[i] == '\'' {
						sb.WriteByte(q[i])
						i++
					} else {
						break
					}
				}
			}
			continue
		}

		// Double-quoted identifiers
		if c == '"' {
			sb.WriteByte(c)
			i++
			for i < len(q) {
				dc := q[i]
				sb.WriteByte(dc)
				i++
				if dc == '"' {
					break
				}
			}
			continue
		}

		// Dollar-quoted strings (PostgreSQL)
		if c == '$' && i+1 < len(q) && (q[i+1] == '$' || (q[i+1] >= 'a' && q[i+1] <= 'z')) {
			tagEnd := strings.Index(q[i+1:], "$")
			if tagEnd != -1 {
				tag := q[i : i+tagEnd+2]
				closingTag := tag
				closingTagPos := strings.Index(q[i+len(tag):], closingTag)
				if closingTagPos != -1 {
					fullBlock := q[i : i+len(tag)+closingTagPos+len(closingTag)]
					sb.WriteString(fullBlock)
					i += len(fullBlock)
					continue
				}
			}
		}

		// Single-line comments
		if c == '-' && i+1 < len(q) && q[i+1] == '-' {
			for i < len(q) && q[i] != '\n' && q[i] != '\r' {
				i++
			}
			// Preserve the newline character
			if i < len(q) {
				sb.WriteByte(q[i])
				i++
			}
			continue
		}

		// Multi-line comments
		if c == '/' && i+1 < len(q) && q[i+1] == '*' {
			i += 2
			for i < len(q)-1 {
				if q[i] == '*' && q[i+1] == '/' {
					i += 2
					break
				}
				i++
			}
			continue
		}

		sb.WriteByte(c)
		i++
	}

	return sb.String()
}

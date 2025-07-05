package psql

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ionut-t/perp/pkg/db"
)

type Executor interface {
	// Execute executes a psql command with optional pattern matching
	Execute(ctx context.Context, cmd *Command) (*Result, error)
}

// executor handles the execution of psql commands
type executor struct {
	db db.Database
}

// New creates a new psql command executor
func New(database db.Database) Executor {
	return &executor{db: database}
}

// Result represents the output of a psql command
type Result struct {
	Columns       []string
	Rows          []map[string]any
	Message       string
	IsError       bool
	ExecutionTime time.Duration
}

// Execute executes a psql command with optional pattern matching
func (e *executor) Execute(ctx context.Context, cmd *Command) (*Result, error) {
	pattern := ""
	if len(cmd.Arguments) > 0 {
		pattern = cmd.Arguments[0]
	}

	start := time.Now()
	var result *Result
	var err error

	switch cmd.Type {
	case CmdListTables:
		if pattern != "" {
			result, err = e.listTablesWithPattern(ctx, pattern)
		} else if cmd.IsExtended() {
			result, err = e.listTablesExtended(ctx)
		} else {
			result, err = e.listTables(ctx)
		}

	default:
		return e.execute(ctx, cmd)
	}

	if result != nil {
		result.ExecutionTime = time.Since(start)
	}

	return result, err
}

// execute runs a parsed psql command
func (e *executor) execute(ctx context.Context, cmd *Command) (*Result, error) {
	start := time.Now()

	var result *Result
	var err error

	switch cmd.Type {
	case CmdDescribe:
		if len(cmd.Arguments) == 0 {
			result, err = e.listRelations(ctx)
		} else {
			result, err = e.describeTable(ctx, cmd.Arguments[0])
		}
	case CmdDescribeTable:
		if len(cmd.Arguments) == 0 {
			return nil, fmt.Errorf("\\d requires a table name")
		}
		result, err = e.describeTable(ctx, cmd.Arguments[0])
	case CmdListTables:
		if cmd.IsExtended() {
			result, err = e.listTablesExtended(ctx)
		} else {
			result, err = e.listTables(ctx)
		}
	case CmdListViews:
		if cmd.IsExtended() {
			result, err = e.listViewsExtended(ctx)
		} else {
			result, err = e.listViews(ctx)
		}
	case CmdListIndexes:
		if cmd.IsExtended() {
			result, err = e.listIndexesExtended(ctx)
		} else {
			result, err = e.listIndexes(ctx)
		}
	case CmdListDatabases:
		if cmd.IsExtended() {
			result, err = e.listDatabasesExtended(ctx)
		} else {
			result, err = e.listDatabases(ctx)
		}
	case CmdListSchemas:
		if cmd.IsExtended() {
			result, err = e.listSchemasExtended(ctx)
		} else {
			result, err = e.listSchemas(ctx)
		}
	case CmdListSequences:
		if cmd.IsExtended() {
			result, err = e.listSequencesExtended(ctx)
		} else {
			result, err = e.listSequences(ctx)
		}
	case CmdListUsers:
		if cmd.IsExtended() {
			result, err = e.listUsersExtended(ctx)
		} else {
			result, err = e.listUsers(ctx)
		}
	case CmdListFunctions:
		if cmd.IsExtended() {
			result, err = e.listFunctionsExtended(ctx)
		} else {
			result, err = e.listFunctions(ctx)
		}
	case CmdListForeignTables:
		if cmd.IsExtended() {
			result, err = e.listForeignTablesExtended(ctx)
		} else {
			result, err = e.listForeignTables(ctx)
		}
	default:
		return nil, fmt.Errorf("command not implemented: %s", cmd.Raw)
	}

	if result != nil {
		result.ExecutionTime = time.Since(start)
	}

	return result, err
}

// sanitizeIdentifier validates and sanitizes a PostgreSQL identifier (table/schema name)
func sanitizeIdentifier(identifier string) (string, error) {
	// Allow alphanumeric, underscore, dot (for schema.table), and dollar sign
	validIdentifier := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_$]*(\.[a-zA-Z_][a-zA-Z0-9_$]*)?$`)

	if !validIdentifier.MatchString(identifier) {
		return "", fmt.Errorf("invalid identifier: %s", identifier)
	}

	return identifier, nil
}

// listRelations implements \d command
func (e *executor) listRelations(ctx context.Context) (*Result, error) {
	query := `
		SELECT 
			n.nspname as "Schema",
			c.relname as "Name",
			CASE c.relkind 
				WHEN 'r' THEN 'table'
				WHEN 'v' THEN 'view'
				WHEN 'm' THEN 'materialized view'
				WHEN 'i' THEN 'index'
				WHEN 'S' THEN 'sequence'
				WHEN 's' THEN 'special'
				WHEN 't' THEN 'TOAST table'
				WHEN 'f' THEN 'foreign table'
				WHEN 'p' THEN 'partitioned table'
				WHEN 'I' THEN 'partitioned index'
			END as "Type",
			pg_catalog.pg_get_userbyid(c.relowner) as "Owner"
		FROM pg_catalog.pg_class c
		LEFT JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		LEFT JOIN pg_catalog.pg_am am ON am.oid = c.relam
		WHERE c.relkind IN ('r','p','v','m','S','f','')
		AND n.nspname <> 'pg_catalog'
		AND n.nspname !~ '^pg_toast'
		AND n.nspname <> 'information_schema'
		AND pg_catalog.pg_table_is_visible(c.oid)
		ORDER BY 1,2;`

	result, err := e.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list relations: %w", err)
	}

	rows, columns, err := db.ExtractPsqlResults(result.Rows())
	if err != nil {
		return nil, err
	}

	return &Result{
		Columns: columns,
		Rows:    rows,
		Message: "List of relations",
	}, nil
}

// describeTable implements \d table_name command
func (e *executor) describeTable(ctx context.Context, tableName string) (*Result, error) {
	// Sanitize the table name to prevent SQL injection
	safeName, err := sanitizeIdentifier(tableName)
	if err != nil {
		return nil, err
	}

	// First, get the table columns
	columnsQuery := `
		SELECT 
			a.attname as "Column",
			pg_catalog.format_type(a.atttypid, a.atttypmod) as "Type",
			CASE 
				WHEN a.attnotnull THEN 'not null' 
				ELSE '' 
			END as "Modifiers",
			COALESCE(
				(SELECT pg_catalog.pg_get_expr(d.adbin, d.adrelid)
				FROM pg_catalog.pg_attrdef d
				WHERE d.adrelid = a.attrelid AND d.adnum = a.attnum),
				''
			) as "Default"
		FROM pg_catalog.pg_attribute a
		WHERE a.attrelid = $1::regclass
		AND a.attnum > 0 
		AND NOT a.attisdropped
		ORDER BY a.attnum;`

	result, err := e.db.Query(ctx, columnsQuery, safeName)
	if err != nil {
		return nil, fmt.Errorf("failed to describe table: %w", err)
	}

	rows, columns, err := db.ExtractPsqlResults(result.Rows())
	if err != nil {
		return nil, err
	}

	// Get indexes
	indexRows, err := e.getTableIndexes(ctx, safeName)
	if err == nil && len(indexRows) > 0 {
		// Add a separator
		rows = append(rows, map[string]any{
			"Column":    "",
			"Type":      "Indexes:",
			"Modifiers": "",
			"Default":   "",
		})
		for _, idx := range indexRows {
			rows = append(rows, map[string]any{
				"Column":    "    " + idx["indexname"].(string),
				"Type":      idx["indexdef"].(string),
				"Modifiers": "",
				"Default":   "",
			})
		}
	}

	// Get constraints
	constraintRows, err := e.getTableConstraints(ctx, safeName)
	if err == nil && len(constraintRows) > 0 {
		rows = append(rows, map[string]any{
			"Column":    "",
			"Type":      "Constraints:",
			"Modifiers": "",
			"Default":   "",
		})
		for _, con := range constraintRows {
			rows = append(rows, map[string]any{
				"Column":    "    " + con["conname"].(string),
				"Type":      con["condef"].(string),
				"Modifiers": "",
				"Default":   "",
			})
		}
	}

	// Get foreign key constraints
	fkRows, err := e.getTableForeignKeys(ctx, safeName)
	if err == nil && len(fkRows) > 0 {
		rows = append(rows, map[string]any{
			"Column":    "",
			"Type":      "Foreign-key constraints:",
			"Modifiers": "",
			"Default":   "",
		})
		for _, fk := range fkRows {
			rows = append(rows, map[string]any{
				"Column":    "    " + fk["constraint_name"].(string),
				"Type":      fk["definition"].(string),
				"Modifiers": "",
				"Default":   "",
			})
		}
	}

	// Get referenced by foreign keys
	refRows, err := e.getTableReferencedBy(ctx, safeName)
	if err == nil && len(refRows) > 0 {
		rows = append(rows, map[string]any{
			"Column":    "",
			"Type":      "Referenced by:",
			"Modifiers": "",
			"Default":   "",
		})
		for _, ref := range refRows {
			rows = append(rows, map[string]any{
				"Column":    "    TABLE " + ref["table_name"].(string),
				"Type":      ref["definition"].(string),
				"Modifiers": "",
				"Default":   "",
			})
		}
	}

	return &Result{
		Columns: columns,
		Rows:    rows,
		Message: fmt.Sprintf("Table \"%s\"", tableName),
	}, nil
}

// getTableIndexes retrieves indexes for a table
func (e *executor) getTableIndexes(ctx context.Context, tableName string) ([]map[string]any, error) {
	query := `
		SELECT 
			indexname,
			indexdef
		FROM pg_indexes
		WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
		AND tablename = $1
		ORDER BY indexname;`

	// Extract table name without schema if present
	parts := strings.Split(tableName, ".")
	tableOnly := parts[len(parts)-1]

	result, err := e.db.Query(ctx, query, tableOnly)
	if err != nil {
		return nil, err
	}

	rows, _, err := db.ExtractPsqlResults(result.Rows())
	return rows, err
}

// getTableConstraints retrieves constraints for a table
func (e *executor) getTableConstraints(ctx context.Context, tableName string) ([]map[string]any, error) {
	query := `
		SELECT 
			conname,
			pg_catalog.pg_get_constraintdef(oid, true) as condef
		FROM pg_constraint
		WHERE conrelid = $1::regclass
		AND contype IN ('c', 'u')
		ORDER BY conname;`

	result, err := e.db.Query(ctx, query, tableName)
	if err != nil {
		return nil, err
	}

	rows, _, err := db.ExtractPsqlResults(result.Rows())
	return rows, err
}

// getTableForeignKeys retrieves foreign key constraints for a table
func (e *executor) getTableForeignKeys(ctx context.Context, tableName string) ([]map[string]any, error) {
	query := `
		SELECT 
			conname as constraint_name,
			pg_catalog.pg_get_constraintdef(oid, true) as definition
		FROM pg_constraint
		WHERE conrelid = $1::regclass
		AND contype = 'f'
		ORDER BY conname;`

	result, err := e.db.Query(ctx, query, tableName)
	if err != nil {
		return nil, err
	}

	rows, _, err := db.ExtractPsqlResults(result.Rows())
	return rows, err
}

// getTableReferencedBy retrieves tables that reference this table
func (e *executor) getTableReferencedBy(ctx context.Context, tableName string) ([]map[string]any, error) {
	query := `
		SELECT 
			conrelid::regclass::text as table_name,
			conname || ' ' || pg_catalog.pg_get_constraintdef(oid, true) as definition
		FROM pg_constraint
		WHERE confrelid = $1::regclass
		AND contype = 'f'
		ORDER BY conrelid::regclass::text, conname;`

	result, err := e.db.Query(ctx, query, tableName)
	if err != nil {
		return nil, err
	}

	rows, _, err := db.ExtractPsqlResults(result.Rows())
	return rows, err
}

// listTables implements \dt command
func (e *executor) listTables(ctx context.Context) (*Result, error) {
	query := `
		SELECT 
			n.nspname as "Schema",
			c.relname as "Name",
			CASE c.relkind 
				WHEN 'r' THEN 'table'
				WHEN 'p' THEN 'partitioned table'
			END as "Type",
			pg_catalog.pg_get_userbyid(c.relowner) as "Owner"
		FROM pg_catalog.pg_class c
		LEFT JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind IN ('r','p')
		AND n.nspname <> 'pg_catalog'
		AND n.nspname <> 'information_schema'
		AND n.nspname !~ '^pg_toast'
		AND pg_catalog.pg_table_is_visible(c.oid)
		ORDER BY 1,2;`

	result, err := e.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}

	rows, columns, err := db.ExtractPsqlResults(result.Rows())
	if err != nil {
		return nil, err
	}

	return &Result{
		Columns: columns,
		Rows:    rows,
		Message: "List of relations",
	}, nil
}

// listTablesExtended implements \dt+ command
func (e *executor) listTablesExtended(ctx context.Context) (*Result, error) {
	query := `
		SELECT 
			n.nspname as "Schema",
			c.relname as "Name",
			CASE c.relkind 
				WHEN 'r' THEN 'table'
				WHEN 'p' THEN 'partitioned table'
			END as "Type",
			pg_catalog.pg_get_userbyid(c.relowner) as "Owner",
			CASE c.relpersistence 
				WHEN 'p' THEN 'permanent' 
				WHEN 't' THEN 'temporary' 
				WHEN 'u' THEN 'unlogged' 
			END as "Persistence",
			pg_catalog.pg_size_pretty(pg_catalog.pg_table_size(c.oid)) as "Size",
			obj_description(c.oid, 'pg_class') as "Description"
		FROM pg_catalog.pg_class c
		LEFT JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind IN ('r','p')
		AND n.nspname <> 'pg_catalog'
		AND n.nspname <> 'information_schema'
		AND n.nspname !~ '^pg_toast'
		AND pg_catalog.pg_table_is_visible(c.oid)
		ORDER BY 1,2;`

	result, err := e.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}

	rows, columns, err := db.ExtractPsqlResults(result.Rows())
	if err != nil {
		return nil, err
	}

	return &Result{
		Columns: columns,
		Rows:    rows,
		Message: "List of relations",
	}, nil
}

// listViews implements \dv command
func (e *executor) listViews(ctx context.Context) (*Result, error) {
	query := `
		SELECT 
			n.nspname as "Schema",
			c.relname as "Name",
			CASE c.relkind 
				WHEN 'v' THEN 'view'
				WHEN 'm' THEN 'materialized view'
			END as "Type",
			pg_catalog.pg_get_userbyid(c.relowner) as "Owner"
		FROM pg_catalog.pg_class c
		LEFT JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind IN ('v', 'm')
		AND n.nspname <> 'pg_catalog'
		AND n.nspname <> 'information_schema'
		AND pg_catalog.pg_table_is_visible(c.oid)
		ORDER BY 1,2;`

	result, err := e.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list views: %w", err)
	}

	rows, columns, err := db.ExtractPsqlResults(result.Rows())
	if err != nil {
		return nil, err
	}

	return &Result{
		Columns: columns,
		Rows:    rows,
		Message: "List of relations",
	}, nil
}

// listViewsExtended implements \dv+ command
func (e *executor) listViewsExtended(ctx context.Context) (*Result, error) {
	query := `
		SELECT 
			n.nspname as "Schema",
			c.relname as "Name",
			CASE c.relkind 
				WHEN 'v' THEN 'view'
				WHEN 'm' THEN 'materialized view'
			END as "Type",
			pg_catalog.pg_get_userbyid(c.relowner) as "Owner",
			CASE c.relpersistence 
				WHEN 'p' THEN 'permanent' 
				WHEN 't' THEN 'temporary' 
				WHEN 'u' THEN 'unlogged' 
			END as "Persistence",
			pg_catalog.pg_size_pretty(pg_catalog.pg_table_size(c.oid)) as "Size",
			obj_description(c.oid, 'pg_class') as "Description"
		FROM pg_catalog.pg_class c
		LEFT JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind IN ('v', 'm')
		AND n.nspname <> 'pg_catalog'
		AND n.nspname <> 'information_schema'
		AND pg_catalog.pg_table_is_visible(c.oid)
		ORDER BY 1,2;`

	result, err := e.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list views: %w", err)
	}

	rows, columns, err := db.ExtractPsqlResults(result.Rows())
	if err != nil {
		return nil, err
	}

	return &Result{
		Columns: columns,
		Rows:    rows,
		Message: "List of relations",
	}, nil
}

// listIndexes implements \di command
func (e *executor) listIndexes(ctx context.Context) (*Result, error) {
	query := `
		SELECT 
			n.nspname as "Schema",
			c.relname as "Name",
			t.relname as "Table",
			a.amname as "Type",
			pg_catalog.pg_get_userbyid(c.relowner) as "Owner"
		FROM pg_catalog.pg_class c
		LEFT JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		LEFT JOIN pg_catalog.pg_index i ON i.indexrelid = c.oid
		LEFT JOIN pg_catalog.pg_class t ON i.indrelid = t.oid
		LEFT JOIN pg_catalog.pg_am a ON c.relam = a.oid
		WHERE c.relkind = 'i'
		AND n.nspname <> 'pg_catalog'
		AND n.nspname <> 'information_schema'
		AND n.nspname !~ '^pg_toast'
		AND pg_catalog.pg_table_is_visible(c.oid)
		ORDER BY 1,2;`

	result, err := e.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list indexes: %w", err)
	}

	rows, columns, err := db.ExtractPsqlResults(result.Rows())
	if err != nil {
		return nil, err
	}

	return &Result{
		Columns: columns,
		Rows:    rows,
		Message: "List of relations",
	}, nil
}

// listIndexesExtended implements \di+ command
func (e *executor) listIndexesExtended(ctx context.Context) (*Result, error) {
	query := `
		SELECT 
			n.nspname as "Schema",
			c.relname as "Name",
			t.relname as "Table",
			a.amname as "Type",
			pg_catalog.pg_get_userbyid(c.relowner) as "Owner",
			CASE c.relpersistence 
				WHEN 'p' THEN 'permanent' 
				WHEN 't' THEN 'temporary' 
				WHEN 'u' THEN 'unlogged' 
			END as "Persistence",
			pg_catalog.pg_size_pretty(pg_catalog.pg_relation_size(c.oid)) as "Size",
			obj_description(c.oid, 'pg_class') as "Description"
		FROM pg_catalog.pg_class c
		LEFT JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		LEFT JOIN pg_catalog.pg_index i ON i.indexrelid = c.oid
		LEFT JOIN pg_catalog.pg_class t ON i.indrelid = t.oid
		LEFT JOIN pg_catalog.pg_am a ON c.relam = a.oid
		WHERE c.relkind = 'i'
		AND n.nspname <> 'pg_catalog'
		AND n.nspname <> 'information_schema'
		AND n.nspname !~ '^pg_toast'
		AND pg_catalog.pg_table_is_visible(c.oid)
		ORDER BY 1,2;`

	result, err := e.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list indexes: %w", err)
	}

	rows, columns, err := db.ExtractPsqlResults(result.Rows())
	if err != nil {
		return nil, err
	}

	return &Result{
		Columns: columns,
		Rows:    rows,
		Message: "List of relations",
	}, nil
}

// listDatabases implements \l command
func (e *executor) listDatabases(ctx context.Context) (*Result, error) {
	query := `
		SELECT 
			d.datname as "Name",
			pg_catalog.pg_get_userbyid(d.datdba) as "Owner",
			pg_catalog.pg_encoding_to_char(d.encoding) as "Encoding",
			d.datcollate as "Collate",
			d.datctype as "Ctype",
			pg_catalog.array_to_string(d.datacl, E'\n') AS "Access privileges"
		FROM pg_catalog.pg_database d
		ORDER BY 1;`

	result, err := e.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}

	rows, columns, err := db.ExtractPsqlResults(result.Rows())
	if err != nil {
		return nil, err
	}

	return &Result{
		Columns: columns,
		Rows:    rows,
		Message: "List of databases",
	}, nil
}

// listDatabasesExtended implements \l+ command
func (e *executor) listDatabasesExtended(ctx context.Context) (*Result, error) {
	query := `
		SELECT 
			d.datname as "Name",
			pg_catalog.pg_get_userbyid(d.datdba) as "Owner",
			pg_catalog.pg_encoding_to_char(d.encoding) as "Encoding",
			d.datcollate as "Collate",
			d.datctype as "Ctype",
			pg_catalog.array_to_string(d.datacl, E'\n') AS "Access privileges",
			CASE 
				WHEN pg_catalog.has_database_privilege(d.datname, 'CONNECT')
				THEN pg_catalog.pg_size_pretty(pg_catalog.pg_database_size(d.oid))
				ELSE 'No Access'
			END as "Size",
			t.spcname as "Tablespace",
			pg_catalog.shobj_description(d.oid, 'pg_database') as "Description"
		FROM pg_catalog.pg_database d
		JOIN pg_catalog.pg_tablespace t on d.dattablespace = t.oid
		ORDER BY 1;`

	result, err := e.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}

	rows, columns, err := db.ExtractPsqlResults(result.Rows())
	if err != nil {
		return nil, err
	}

	return &Result{
		Columns: columns,
		Rows:    rows,
		Message: "List of databases",
	}, nil
}

// listSchemas implements \dn command
func (e *executor) listSchemas(ctx context.Context) (*Result, error) {
	query := `
		SELECT 
			n.nspname AS "Name",
			pg_catalog.pg_get_userbyid(n.nspowner) AS "Owner"
		FROM pg_catalog.pg_namespace n
		WHERE n.nspname !~ '^pg_' AND n.nspname <> 'information_schema'
		ORDER BY 1;`

	result, err := e.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list schemas: %w", err)
	}

	rows, columns, err := db.ExtractPsqlResults(result.Rows())
	if err != nil {
		return nil, err
	}

	return &Result{
		Columns: columns,
		Rows:    rows,
		Message: "List of schemas",
	}, nil
}

// listSchemasExtended implements \dn+ command
func (e *executor) listSchemasExtended(ctx context.Context) (*Result, error) {
	query := `
		SELECT 
			n.nspname AS "Name",
			pg_catalog.pg_get_userbyid(n.nspowner) AS "Owner",
			pg_catalog.array_to_string(n.nspacl, E'\n') AS "Access privileges",
			pg_catalog.obj_description(n.oid, 'pg_namespace') AS "Description"
		FROM pg_catalog.pg_namespace n
		WHERE n.nspname !~ '^pg_' AND n.nspname <> 'information_schema'
		ORDER BY 1;`

	result, err := e.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list schemas: %w", err)
	}

	rows, columns, err := db.ExtractPsqlResults(result.Rows())
	if err != nil {
		return nil, err
	}

	return &Result{
		Columns: columns,
		Rows:    rows,
		Message: "List of schemas",
	}, nil
}

// listSequences implements \ds command
func (e *executor) listSequences(ctx context.Context) (*Result, error) {
	query := `
		SELECT 
			n.nspname as "Schema",
			c.relname as "Name",
			CASE c.relkind WHEN 'S' THEN 'sequence' END as "Type",
			pg_catalog.pg_get_userbyid(c.relowner) as "Owner"
		FROM pg_catalog.pg_class c
		LEFT JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind = 'S'
		AND n.nspname <> 'pg_catalog'
		AND n.nspname <> 'information_schema'
		AND n.nspname !~ '^pg_toast'
		AND pg_catalog.pg_table_is_visible(c.oid)
		ORDER BY 1,2;`

	result, err := e.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list sequences: %w", err)
	}

	rows, columns, err := db.ExtractPsqlResults(result.Rows())
	if err != nil {
		return nil, err
	}

	return &Result{
		Columns: columns,
		Rows:    rows,
		Message: "List of relations",
	}, nil
}

// listSequencesExtended implements \ds+ command
func (e *executor) listSequencesExtended(ctx context.Context) (*Result, error) {
	query := `
		SELECT 
			n.nspname as "Schema",
			c.relname as "Name",
			CASE c.relkind WHEN 'S' THEN 'sequence' END as "Type",
			pg_catalog.pg_get_userbyid(c.relowner) as "Owner",
			CASE c.relpersistence 
				WHEN 'p' THEN 'permanent' 
				WHEN 't' THEN 'temporary' 
				WHEN 'u' THEN 'unlogged' 
			END as "Persistence",
			pg_catalog.pg_size_pretty(pg_catalog.pg_table_size(c.oid)) as "Size",
			obj_description(c.oid, 'pg_class') as "Description"
		FROM pg_catalog.pg_class c
		LEFT JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind = 'S'
		AND n.nspname <> 'pg_catalog'
		AND n.nspname <> 'information_schema'
		AND n.nspname !~ '^pg_toast'
		AND pg_catalog.pg_table_is_visible(c.oid)
		ORDER BY 1,2;`

	result, err := e.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list sequences: %w", err)
	}

	rows, columns, err := db.ExtractPsqlResults(result.Rows())
	if err != nil {
		return nil, err
	}

	return &Result{
		Columns: columns,
		Rows:    rows,
		Message: "List of relations",
	}, nil
}

// listUsers implements \du command
func (e *executor) listUsers(ctx context.Context) (*Result, error) {
	query := `
		SELECT 
			r.rolname as "Role name",
			ARRAY(
				SELECT b.rolname
				FROM pg_catalog.pg_auth_members m
				JOIN pg_catalog.pg_roles b ON (m.roleid = b.oid)
				WHERE m.member = r.oid
			) as "Member of"
		FROM pg_catalog.pg_roles r
		WHERE r.rolname !~ '^pg_'
		ORDER BY 1;`

	result, err := e.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	rows, columns, err := db.ExtractPsqlResults(result.Rows())
	if err != nil {
		return nil, err
	}

	// Format the "Member of" column to show array values properly
	for _, row := range rows {
		if memberOf, ok := row["Member of"]; ok {
			if arr, ok := memberOf.([]interface{}); ok {
				members := make([]string, len(arr))
				for i, m := range arr {
					members[i] = fmt.Sprintf("%v", m)
				}
				row["Member of"] = "{" + strings.Join(members, ",") + "}"
			}
		}
	}

	return &Result{
		Columns: columns,
		Rows:    rows,
		Message: "List of roles",
	}, nil
}

// listUsersExtended implements \du+ command
func (e *executor) listUsersExtended(ctx context.Context) (*Result, error) {
	query := `
		SELECT 
			r.rolname as "Role name",
			ARRAY_TO_STRING(ARRAY(
				SELECT a.rolname || '=' || m.admin_option
				FROM pg_catalog.pg_auth_members m
				JOIN pg_catalog.pg_roles a ON (m.roleid = a.oid)
				WHERE m.member = r.oid
				ORDER BY 1
			), E'\n') as "Member of",
			r.rolsuper as "Superuser",
			r.rolinherit as "Inherit",
			r.rolcreaterole as "Create role",
			r.rolcreatedb as "Create DB",
			r.rolcanlogin as "Can login",
			r.rolreplication as "Replication",
			CASE r.rolconnlimit 
				WHEN -1 THEN 'unlimited' 
				ELSE r.rolconnlimit::text 
			END as "Connections",
			r.rolvaliduntil as "Valid until",
			ARRAY_TO_STRING(r.rolconfig, E'\n') as "Config",
			pg_catalog.shobj_description(r.oid, 'pg_authid') as "Description"
		FROM pg_catalog.pg_roles r
		WHERE r.rolname !~ '^pg_'
		ORDER BY 1;`

	result, err := e.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	rows, columns, err := db.ExtractPsqlResults(result.Rows())
	if err != nil {
		return nil, err
	}

	// Format boolean values to be more readable
	for _, row := range rows {
		for _, col := range []string{"Superuser", "Inherit", "Create role", "Create DB", "Can login", "Replication"} {
			if val, ok := row[col]; ok {
				if b, ok := val.(bool); ok {
					if b {
						row[col] = "yes"
					} else {
						row[col] = "no"
					}
				}
			}
		}
	}

	return &Result{
		Columns: columns,
		Rows:    rows,
		Message: "List of roles",
	}, nil
}

// listFunctions implements \df command
func (e *executor) listFunctions(ctx context.Context) (*Result, error) {
	query := `
		SELECT 
			n.nspname as "Schema",
			p.proname as "Name",
			pg_catalog.pg_get_function_result(p.oid) as "Result data type",
			pg_catalog.pg_get_function_arguments(p.oid) as "Argument data types",
			CASE p.prokind
				WHEN 'a' THEN 'agg'
				WHEN 'w' THEN 'window'
				WHEN 'p' THEN 'proc'
				ELSE 'func'
			END as "Type"
		FROM pg_catalog.pg_proc p
		LEFT JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
		WHERE pg_catalog.pg_function_is_visible(p.oid)
		AND n.nspname <> 'pg_catalog'
		AND n.nspname <> 'information_schema'
		ORDER BY 1, 2, 4;`

	result, err := e.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list functions: %w", err)
	}

	rows, columns, err := db.ExtractPsqlResults(result.Rows())
	if err != nil {
		return nil, err
	}

	return &Result{
		Columns: columns,
		Rows:    rows,
		Message: "List of functions",
	}, nil
}

// listFunctionsExtended implements \df+ command
func (e *executor) listFunctionsExtended(ctx context.Context) (*Result, error) {
	query := `
		SELECT 
			n.nspname as "Schema",
			p.proname as "Name",
			pg_catalog.pg_get_function_result(p.oid) as "Result data type",
			pg_catalog.pg_get_function_arguments(p.oid) as "Argument data types",
			CASE p.prokind
				WHEN 'a' THEN 'agg'
				WHEN 'w' THEN 'window'
				WHEN 'p' THEN 'proc'
				ELSE 'func'
			END as "Type",
			CASE
				WHEN p.provolatile = 'i' THEN 'immutable'
				WHEN p.provolatile = 's' THEN 'stable'
				WHEN p.provolatile = 'v' THEN 'volatile'
			END as "Volatility",
			CASE
				WHEN p.proparallel = 'r' THEN 'restricted'
				WHEN p.proparallel = 's' THEN 'safe'
				WHEN p.proparallel = 'u' THEN 'unsafe'
			END as "Parallel",
			pg_catalog.pg_get_userbyid(p.proowner) as "Owner",
			CASE 
				WHEN p.prosecdef THEN 'definer' 
				ELSE 'invoker' 
			END AS "Security",
			pg_catalog.array_to_string(p.proacl, E'\n') AS "Access privileges",
			l.lanname as "Language",
			p.prosrc as "Source code",
			pg_catalog.obj_description(p.oid, 'pg_proc') as "Description"
		FROM pg_catalog.pg_proc p
		LEFT JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
		LEFT JOIN pg_catalog.pg_language l ON l.oid = p.prolang
		WHERE pg_catalog.pg_function_is_visible(p.oid)
		AND n.nspname <> 'pg_catalog'
		AND n.nspname <> 'information_schema'
		ORDER BY 1, 2, 4;`

	result, err := e.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list functions: %w", err)
	}

	rows, columns, err := db.ExtractPsqlResults(result.Rows())
	if err != nil {
		return nil, err
	}

	return &Result{
		Columns: columns,
		Rows:    rows,
		Message: "List of functions",
	}, nil
}

// listForeignTables implements \dE command
func (e *executor) listForeignTables(ctx context.Context) (*Result, error) {
	query := `
		SELECT 
			n.nspname as "Schema",
			c.relname as "Name",
			CASE c.relkind 
				WHEN 'f' THEN 'foreign table'
			END as "Type",
			pg_catalog.pg_get_userbyid(c.relowner) as "Owner"
		FROM pg_catalog.pg_class c
		LEFT JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind = 'f'
		AND n.nspname <> 'pg_catalog'
		AND n.nspname <> 'information_schema'
		AND n.nspname !~ '^pg_toast'
		AND pg_catalog.pg_table_is_visible(c.oid)
		ORDER BY 1,2;`

	result, err := e.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list foreign tables: %w", err)
	}

	rows, columns, err := db.ExtractPsqlResults(result.Rows())
	if err != nil {
		return nil, err
	}

	return &Result{
		Columns: columns,
		Rows:    rows,
		Message: "List of relations",
	}, nil
}

// listForeignTablesExtended implements \dE+ command
func (e *executor) listForeignTablesExtended(ctx context.Context) (*Result, error) {
	query := `
		SELECT 
			n.nspname as "Schema",
			c.relname as "Name",
			CASE c.relkind 
				WHEN 'f' THEN 'foreign table'
			END as "Type",
			pg_catalog.pg_get_userbyid(c.relowner) as "Owner",
			s.srvname as "FDW server",
			pg_catalog.pg_size_pretty(pg_catalog.pg_table_size(c.oid)) as "Size",
			obj_description(c.oid, 'pg_class') as "Description"
		FROM pg_catalog.pg_class c
		LEFT JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		LEFT JOIN pg_catalog.pg_foreign_table f ON f.ftrelid = c.oid
		LEFT JOIN pg_catalog.pg_foreign_server s ON s.oid = f.ftserver
		WHERE c.relkind = 'f'
		AND n.nspname <> 'pg_catalog'
		AND n.nspname <> 'information_schema'
		AND n.nspname !~ '^pg_toast'
		AND pg_catalog.pg_table_is_visible(c.oid)
		ORDER BY 1,2;`

	result, err := e.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list foreign tables: %w", err)
	}

	rows, columns, err := db.ExtractPsqlResults(result.Rows())
	if err != nil {
		return nil, err
	}

	return &Result{
		Columns: columns,
		Rows:    rows,
		Message: "List of relations",
	}, nil
}

// patternToLike converts a psql pattern to a SQL LIKE pattern
func patternToLike(pattern string) string {
	// In psql, * means any sequence of characters (like % in SQL)
	// and ? means any single character (like _ in SQL)
	result := strings.ReplaceAll(pattern, "*", "%")
	result = strings.ReplaceAll(result, "?", "_")
	return result
}

// parseSchemaAndTable splits a pattern into schema and table parts
func parseSchemaAndTable(pattern string) (schema, table string) {
	parts := strings.SplitN(pattern, ".", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", parts[0]
}

// buildPatternCondition builds SQL conditions for pattern matching
func buildPatternCondition(pattern string, schemaCol, nameCol string) string {
	if pattern == "" {
		return ""
	}

	schema, name := parseSchemaAndTable(pattern)
	conditions := []string{}

	if schema != "" {
		schemaPattern := patternToLike(schema)
		conditions = append(conditions, fmt.Sprintf("%s LIKE '%s'", schemaCol, schemaPattern))
	}

	if name != "" {
		namePattern := patternToLike(name)
		conditions = append(conditions, fmt.Sprintf("%s LIKE '%s'", nameCol, namePattern))
	}

	if len(conditions) > 0 {
		return " AND " + strings.Join(conditions, " AND ")
	}
	return ""
}

// listTablesWithPattern lists tables matching the given pattern
func (e *executor) listTablesWithPattern(ctx context.Context, pattern string) (*Result, error) {
	patternCondition := buildPatternCondition(pattern, "n.nspname", "c.relname")

	query := fmt.Sprintf(`
		SELECT 
			n.nspname as "Schema",
			c.relname as "Name",
			CASE c.relkind 
				WHEN 'r' THEN 'table'
				WHEN 'p' THEN 'partitioned table'
			END as "Type",
			pg_catalog.pg_get_userbyid(c.relowner) as "Owner"
		FROM pg_catalog.pg_class c
		LEFT JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind IN ('r','p')
		AND n.nspname <> 'pg_catalog'
		AND n.nspname <> 'information_schema'
		AND n.nspname !~ '^pg_toast'
		AND pg_catalog.pg_table_is_visible(c.oid)
		%s
		ORDER BY 1,2;`, patternCondition)

	result, err := e.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}

	rows, columns, err := db.ExtractPsqlResults(result.Rows())
	if err != nil {
		return nil, err
	}

	return &Result{
		Columns: columns,
		Rows:    rows,
		Message: "List of relations",
	}, nil
}

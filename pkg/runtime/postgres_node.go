package runtime

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/tcmartin/flowlib"
)

// PostgresManager manages PostgreSQL operations
type PostgresManager struct {
	db        *sql.DB
	tableName string
}

// Global PostgreSQL manager
var globalPostgresManager *PostgresManager

// GetPostgresManager returns the global PostgreSQL manager
func GetPostgresManager(config map[string]interface{}) (*PostgresManager, error) {
	if globalPostgresManager != nil {
		return globalPostgresManager, nil
	}

	// Extract connection parameters
	host, _ := config["host"].(string)
	if host == "" {
		host = "localhost" // Default host
	}

	port := 5432 // Default port
	if portParam, ok := config["port"].(float64); ok {
		port = int(portParam)
	}

	user, _ := config["user"].(string)
	if user == "" {
		user = "postgres" // Default user
	}

	fmt.Printf("PostgreSQL user from config: %s\n", user)

	password, _ := config["password"].(string)

	dbname, _ := config["dbname"].(string)
	if dbname == "" {
		dbname = "postgres" // Default database
	}

	// Extract table name
	tableName, _ := config["table_name"].(string)
	if tableName == "" {
		tableName = "flowrunner_store" // Default table name
	}

	// Create connection string
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	// Print connection string for debugging (without password)
	debugConnStr := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=disable",
		host, port, user, dbname)
	fmt.Printf("Connecting to PostgreSQL with: %s\n", debugConnStr)

	// Connect to database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	// Create manager
	globalPostgresManager = &PostgresManager{
		db:        db,
		tableName: tableName,
	}

	// Ensure table exists
	if err := globalPostgresManager.ensureTableExists(); err != nil {
		return nil, fmt.Errorf("failed to ensure table exists: %w", err)
	}

	return globalPostgresManager, nil
}

// ensureTableExists creates the PostgreSQL table if it doesn't exist
func (pm *PostgresManager) ensureTableExists() error {
	// Create table
	_, err := pm.db.Exec(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			key TEXT PRIMARY KEY,
			value JSONB NOT NULL,
			ttl TIMESTAMP WITH TIME ZONE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`, pm.tableName))

	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Create index on TTL
	_, err = pm.db.Exec(fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS %s_ttl_idx ON %s (ttl)
	`, pm.tableName, pm.tableName))

	if err != nil {
		return fmt.Errorf("failed to create TTL index: %w", err)
	}

	return nil
}

// Get retrieves an item from PostgreSQL
func (pm *PostgresManager) Get(key string) (interface{}, error) {
	// Get item
	var valueStr string
	var ttl sql.NullTime

	err := pm.db.QueryRow(fmt.Sprintf(`
		SELECT value, ttl FROM %s WHERE key = $1 AND (ttl IS NULL OR ttl > NOW())
	`, pm.tableName), key).Scan(&valueStr, &ttl)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("item not found: %s", key)
		}
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	// Parse value
	var value interface{}
	if err := json.Unmarshal([]byte(valueStr), &value); err != nil {
		return nil, fmt.Errorf("failed to parse value: %w", err)
	}

	return value, nil
}

// Set stores an item in PostgreSQL
func (pm *PostgresManager) Set(key string, value interface{}, ttl time.Duration) error {
	// Marshal value
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	// Calculate TTL
	var ttlValue interface{}
	if ttl > 0 {
		ttlValue = time.Now().Add(ttl)
	} else {
		ttlValue = nil
	}

	// Upsert item
	_, err = pm.db.Exec(fmt.Sprintf(`
		INSERT INTO %s (key, value, ttl)
		VALUES ($1, $2, $3)
		ON CONFLICT (key) DO UPDATE
		SET value = $2, ttl = $3
	`, pm.tableName), key, string(valueBytes), ttlValue)

	if err != nil {
		return fmt.Errorf("failed to set item: %w", err)
	}

	return nil
}

// Delete removes an item from PostgreSQL
func (pm *PostgresManager) Delete(key string) (bool, error) {
	// Delete item
	result, err := pm.db.Exec(fmt.Sprintf(`
		DELETE FROM %s WHERE key = $1
	`, pm.tableName), key)

	if err != nil {
		return false, fmt.Errorf("failed to delete item: %w", err)
	}

	// Check if item existed
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected > 0, nil
}

// List returns all keys in PostgreSQL
func (pm *PostgresManager) List() ([]string, error) {
	// List keys
	rows, err := pm.db.Query(fmt.Sprintf(`
		SELECT key FROM %s WHERE ttl IS NULL OR ttl > NOW()
	`, pm.tableName))

	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}
	defer rows.Close()

	// Extract keys
	keys := make([]string, 0)
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("failed to scan key: %w", err)
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return keys, nil
}

// Query performs a query on PostgreSQL
func (pm *PostgresManager) Query(filter map[string]interface{}, sortField string, limit int) ([]map[string]interface{}, error) {
	// Build query
	query := fmt.Sprintf(`
		SELECT key, value FROM %s WHERE (ttl IS NULL OR ttl > NOW())
	`, pm.tableName)

	// Add filter conditions
	var conditions []string
	var args []interface{}
	argIndex := 1

	for key, value := range filter {
		if valueMap, ok := value.(map[string]interface{}); ok {
			for op, opValue := range valueMap {
				var condition string
				switch op {
				case "$eq":
					condition = fmt.Sprintf("value->>'%s' = $%d", key, argIndex)
					args = append(args, fmt.Sprintf("%v", opValue))
				case "$ne":
					condition = fmt.Sprintf("value->>'%s' != $%d", key, argIndex)
					args = append(args, fmt.Sprintf("%v", opValue))
				case "$gt":
					condition = fmt.Sprintf("(value->>'%s')::numeric > $%d", key, argIndex)
					args = append(args, fmt.Sprintf("%v", opValue))
				case "$gte":
					condition = fmt.Sprintf("(value->>'%s')::numeric >= $%d", key, argIndex)
					args = append(args, fmt.Sprintf("%v", opValue))
				case "$lt":
					condition = fmt.Sprintf("(value->>'%s')::numeric < $%d", key, argIndex)
					args = append(args, fmt.Sprintf("%v", opValue))
				case "$lte":
					condition = fmt.Sprintf("(value->>'%s')::numeric <= $%d", key, argIndex)
					args = append(args, fmt.Sprintf("%v", opValue))
				case "$contains":
					condition = fmt.Sprintf("value->>'%s' LIKE $%d", key, argIndex)
					args = append(args, fmt.Sprintf("%%%v%%", opValue))
				}
				if condition != "" {
					conditions = append(conditions, condition)
					argIndex++
				}
			}
		} else {
			condition := fmt.Sprintf("value->>'%s' = $%d", key, argIndex)
			args = append(args, fmt.Sprintf("%v", value))
			conditions = append(conditions, condition)
			argIndex++
		}
	}

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	// Add sort
	if sortField != "" {
		direction := "ASC"
		if strings.HasPrefix(sortField, "-") {
			sortField = sortField[1:]
			direction = "DESC"
		}
		query += fmt.Sprintf(" ORDER BY value->>'%s' %s", sortField, direction)
	}

	// Add limit
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	// Execute query
	rows, err := pm.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Extract results
	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		var key string
		var valueStr string

		if err := rows.Scan(&key, &valueStr); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		var value map[string]interface{}
		if err := json.Unmarshal([]byte(valueStr), &value); err != nil {
			return nil, fmt.Errorf("failed to parse value: %w", err)
		}

		// Add key to value
		value["_key"] = key

		results = append(results, value)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return results, nil
}

// ExecuteSQL executes a SQL query
func (pm *PostgresManager) ExecuteSQL(query string, args []interface{}) (interface{}, error) {
	// Check if query is a SELECT
	isSelect := strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "SELECT")

	if isSelect {
		// Execute SELECT query
		rows, err := pm.db.Query(query, args...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute query: %w", err)
		}
		defer rows.Close()

		// Get column names
		columns, err := rows.Columns()
		if err != nil {
			return nil, fmt.Errorf("failed to get columns: %w", err)
		}

		// Extract results
		results := make([]map[string]interface{}, 0)
		for rows.Next() {
			// Create a slice of interface{} to hold the values
			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for i := range values {
				valuePtrs[i] = &values[i]
			}

			// Scan the row into the slice
			if err := rows.Scan(valuePtrs...); err != nil {
				return nil, fmt.Errorf("failed to scan row: %w", err)
			}

			// Create a map for the row
			row := make(map[string]interface{})
			for i, col := range columns {
				val := values[i]
				// Handle null values
				if val == nil {
					row[col] = nil
				} else {
					// Try to convert bytes to string
					if b, ok := val.([]byte); ok {
						// Try to parse as JSON
						var jsonVal interface{}
						if err := json.Unmarshal(b, &jsonVal); err == nil {
							row[col] = jsonVal
						} else {
							row[col] = string(b)
						}
					} else {
						row[col] = val
					}
				}
			}

			results = append(results, row)
		}

		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating rows: %w", err)
		}

		return results, nil
	} else {
		// Execute non-SELECT query
		result, err := pm.db.Exec(query, args...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute query: %w", err)
		}

		// Get rows affected
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("failed to get rows affected: %w", err)
		}

		return map[string]interface{}{
			"rows_affected": rowsAffected,
		}, nil
	}
}

// BeginTransaction begins a transaction
func (pm *PostgresManager) BeginTransaction() (*sql.Tx, error) {
	return pm.db.Begin()
}

// NewPostgresNodeWrapper creates a new PostgreSQL node wrapper
func NewPostgresNodeWrapper(initParams map[string]interface{}) (flowlib.Node, error) {
	// Create the base node
	baseNode := flowlib.NewNode(3, 1*time.Second)

	// Store connection parameters
	host, _ := initParams["host"].(string)
	user, _ := initParams["user"].(string)
	password, _ := initParams["password"].(string)
	dbname, _ := initParams["dbname"].(string)
	tableName, _ := initParams["table_name"].(string)

	// Print the connection parameters for debugging
	fmt.Printf("PostgreSQL connection parameters: host=%s, user=%s, dbname=%s, table_name=%s\n",
		host, user, dbname, tableName)

	// Create the wrapper
	wrapper := &NodeWrapper{
		node: baseNode,
		exec: func(input interface{}) (interface{}, error) {
			// Handle both old format (direct params) and new format (combined input)
			var params map[string]interface{}
			
			if combinedInput, ok := input.(map[string]interface{}); ok {
				if nodeParams, hasParams := combinedInput["params"]; hasParams {
					// New format: combined input with params and input
					if paramsMap, ok := nodeParams.(map[string]interface{}); ok {
						params = paramsMap
					} else {
						return nil, fmt.Errorf("expected params to be map[string]interface{}")
					}
				} else {
					// Old format: direct params (backwards compatibility)
					params = combinedInput
				}
			} else {
				return nil, fmt.Errorf("expected map[string]interface{}, got %T", input)
			}

			// Print the parameters for debugging
			fmt.Printf("PostgreSQL node exec parameters: %v\n", params)

			// Add connection parameters to exec parameters
			if host != "" {
				params["host"] = host
			}
			if user != "" {
				params["user"] = user
			}
			if password != "" {
				params["password"] = password
			}
			if dbname != "" {
				params["dbname"] = dbname
			}
			if tableName != "" {
				params["table_name"] = tableName
			}

			// Print the combined parameters for debugging
			fmt.Printf("PostgreSQL combined parameters: %v\n", params)

			// Get PostgreSQL manager
			manager, err := GetPostgresManager(params)
			if err != nil {
				return nil, fmt.Errorf("failed to get PostgreSQL manager: %w", err)
			}

			// Get operation
			operation, ok := params["operation"].(string)
			if !ok {
				return nil, fmt.Errorf("operation parameter is required")
			}

			switch operation {
			case "get":
				// Get key
				key, ok := params["key"].(string)
				if !ok {
					return nil, fmt.Errorf("key parameter is required for get operation")
				}

				// Get item
				value, err := manager.Get(key)
				if err != nil {
					return nil, err
				}

				return value, nil

			case "set":
				// Get key
				key, ok := params["key"].(string)
				if !ok {
					return nil, fmt.Errorf("key parameter is required for set operation")
				}

				// Get value
				value, ok := params["value"]
				if !ok {
					return nil, fmt.Errorf("value parameter is required for set operation")
				}

				// Get TTL
				var ttl time.Duration
				if ttlStr, ok := params["ttl"].(string); ok {
					var err error
					ttl, err = time.ParseDuration(ttlStr)
					if err != nil {
						return nil, fmt.Errorf("invalid ttl format: %w", err)
					}
				}

				// Set item
				if err := manager.Set(key, value, ttl); err != nil {
					return nil, err
				}

				return value, nil

			case "delete":
				// Get key
				key, ok := params["key"].(string)
				if !ok {
					return nil, fmt.Errorf("key parameter is required for delete operation")
				}

				// Delete item
				exists, err := manager.Delete(key)
				if err != nil {
					return nil, err
				}

				return map[string]interface{}{
					"deleted": exists,
					"key":     key,
				}, nil

			case "list":
				// List keys
				keys, err := manager.List()
				if err != nil {
					return nil, err
				}

				return keys, nil

			case "query":
				// Get filter
				var filter map[string]interface{}
				if filterParam, ok := params["filter"].(map[string]interface{}); ok {
					filter = filterParam
				} else {
					filter = make(map[string]interface{})
				}

				// Get sort
				var sortField string
				if sortParam, ok := params["sort"].(string); ok {
					sortField = sortParam
				}

				// Get limit
				limit := 0
				if limitParam, ok := params["limit"].(float64); ok {
					limit = int(limitParam)
				}

				// Query items
				results, err := manager.Query(filter, sortField, limit)
				if err != nil {
					return nil, err
				}

				return results, nil

			case "execute":
				// Get SQL query
				query, ok := params["query"].(string)
				if !ok {
					return nil, fmt.Errorf("query parameter is required for execute operation")
				}

				// Get arguments
				var args []interface{}
				if argsParam, ok := params["args"].([]interface{}); ok {
					args = argsParam
				}

				// Execute query
				result, err := manager.ExecuteSQL(query, args)
				if err != nil {
					return nil, err
				}

				return result, nil

			case "transaction":
				// Get statements
				statements, ok := params["statements"].([]interface{})
				if !ok {
					return nil, fmt.Errorf("statements parameter is required for transaction operation")
				}

				// Begin transaction
				tx, err := manager.BeginTransaction()
				if err != nil {
					return nil, fmt.Errorf("failed to begin transaction: %w", err)
				}

				// Ensure transaction is rolled back on error
				defer func() {
					if err != nil {
						tx.Rollback()
					}
				}()

				// Execute statements
				results := make([]interface{}, len(statements))
				for i, stmt := range statements {
					stmtMap, ok := stmt.(map[string]interface{})
					if !ok {
						err = fmt.Errorf("statement must be a map")
						return nil, err
					}

					// Get SQL query
					query, ok := stmtMap["query"].(string)
					if !ok {
						err = fmt.Errorf("query parameter is required for statement")
						return nil, err
					}

					// Get arguments
					var args []interface{}
					if argsParam, ok := stmtMap["args"].([]interface{}); ok {
						args = argsParam
					}

					// Check if query is a SELECT
					isSelect := strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "SELECT")

					if isSelect {
						// Execute SELECT query
						rows, err := tx.Query(query, args...)
						if err != nil {
							return nil, fmt.Errorf("failed to execute query: %w", err)
						}

						// Get column names
						columns, err := rows.Columns()
						if err != nil {
							rows.Close()
							return nil, fmt.Errorf("failed to get columns: %w", err)
						}

						// Extract results
						stmtResults := make([]map[string]interface{}, 0)
						for rows.Next() {
							// Create a slice of interface{} to hold the values
							values := make([]interface{}, len(columns))
							valuePtrs := make([]interface{}, len(columns))
							for i := range values {
								valuePtrs[i] = &values[i]
							}

							// Scan the row into the slice
							if err := rows.Scan(valuePtrs...); err != nil {
								rows.Close()
								return nil, fmt.Errorf("failed to scan row: %w", err)
							}

							// Create a map for the row
							row := make(map[string]interface{})
							for i, col := range columns {
								val := values[i]
								// Handle null values
								if val == nil {
									row[col] = nil
								} else {
									// Try to convert bytes to string
									if b, ok := val.([]byte); ok {
										// Try to parse as JSON
										var jsonVal interface{}
										if err := json.Unmarshal(b, &jsonVal); err == nil {
											row[col] = jsonVal
										} else {
											row[col] = string(b)
										}
									} else {
										row[col] = val
									}
								}
							}

							stmtResults = append(stmtResults, row)
						}

						if err := rows.Err(); err != nil {
							rows.Close()
							return nil, fmt.Errorf("error iterating rows: %w", err)
						}

						rows.Close()
						results[i] = stmtResults
					} else {
						// Execute non-SELECT query
						result, err := tx.Exec(query, args...)
						if err != nil {
							return nil, fmt.Errorf("failed to execute query: %w", err)
						}

						// Get rows affected
						rowsAffected, err := result.RowsAffected()
						if err != nil {
							return nil, fmt.Errorf("failed to get rows affected: %w", err)
						}

						results[i] = map[string]interface{}{
							"rows_affected": rowsAffected,
						}
					}
				}

				// Commit transaction
				if err := tx.Commit(); err != nil {
					return nil, fmt.Errorf("failed to commit transaction: %w", err)
				}

				return map[string]interface{}{
					"results": results,
				}, nil

			default:
				return nil, fmt.Errorf("unknown operation: %s", operation)
			}
		},
	}

	// Set the parameters
	wrapper.SetParams(initParams)

	return wrapper, nil
}

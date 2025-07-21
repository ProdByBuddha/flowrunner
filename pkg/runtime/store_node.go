package runtime

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tcmartin/flowlib"
)

// StoreManager manages in-memory and persistent storage
type StoreManager struct {
	// In-memory store
	memStore map[string]interface{}

	// TTL store for expiring values
	ttlStore map[string]time.Time

	// File path for persistent storage
	filePath string

	// Mutex for thread safety
	mutex sync.RWMutex

	// Auto-save flag
	autoSave bool
}

// Global store manager
var globalStoreManager *StoreManager
var storeOnce sync.Once

// GetStoreManager returns the global store manager
func GetStoreManager() *StoreManager {
	storeOnce.Do(func() {
		globalStoreManager = &StoreManager{
			memStore: make(map[string]interface{}),
			ttlStore: make(map[string]time.Time),
			filePath: "flowrunner_store.json",
			autoSave: true,
		}

		// Try to load from file
		globalStoreManager.LoadFromFile()

		// Start TTL cleanup goroutine
		go globalStoreManager.ttlCleanupLoop()
	})

	return globalStoreManager
}

// ttlCleanupLoop periodically cleans up expired values
func (sm *StoreManager) ttlCleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		sm.cleanupExpiredValues()
	}
}

// cleanupExpiredValues removes expired values from the store
func (sm *StoreManager) cleanupExpiredValues() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	now := time.Now()
	keysToDelete := []string{}

	for key, expiry := range sm.ttlStore {
		if now.After(expiry) {
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		delete(sm.memStore, key)
		delete(sm.ttlStore, key)
	}

	if len(keysToDelete) > 0 && sm.autoSave {
		sm.saveToFile()
	}
}

// Get retrieves a value from the store
func (sm *StoreManager) Get(key string) (interface{}, bool) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	// Check if the key has expired
	if expiry, ok := sm.ttlStore[key]; ok && time.Now().After(expiry) {
		// Clean up expired key
		go func() {
			sm.mutex.Lock()
			defer sm.mutex.Unlock()
			delete(sm.memStore, key)
			delete(sm.ttlStore, key)
			if sm.autoSave {
				sm.saveToFile()
			}
		}()
		return nil, false
	}

	value, exists := sm.memStore[key]
	return value, exists
}

// Set stores a value in the store
func (sm *StoreManager) Set(key string, value interface{}, ttl time.Duration) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.memStore[key] = value

	// Set TTL if provided
	if ttl > 0 {
		sm.ttlStore[key] = time.Now().Add(ttl)
	} else {
		delete(sm.ttlStore, key)
	}

	if sm.autoSave {
		sm.saveToFile()
	}
}

// Delete removes a value from the store
func (sm *StoreManager) Delete(key string) bool {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	_, exists := sm.memStore[key]
	if exists {
		delete(sm.memStore, key)
		delete(sm.ttlStore, key)

		if sm.autoSave {
			sm.saveToFile()
		}
	}

	return exists
}

// List returns all keys in the store
func (sm *StoreManager) List() []string {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	keys := make([]string, 0, len(sm.memStore))
	for key := range sm.memStore {
		// Skip expired keys
		if expiry, ok := sm.ttlStore[key]; ok && time.Now().After(expiry) {
			continue
		}
		keys = append(keys, key)
	}

	sort.Strings(keys)
	return keys
}

// Query performs a query on the store
func (sm *StoreManager) Query(filter map[string]interface{}, sort string, limit int) []map[string]interface{} {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	// Collect all objects that are maps
	results := []map[string]interface{}{}

	for key, value := range sm.memStore {
		// Skip expired keys
		if expiry, ok := sm.ttlStore[key]; ok && time.Now().After(expiry) {
			continue
		}

		// Only process map values
		valueMap, ok := value.(map[string]interface{})
		if !ok {
			continue
		}

		// Add the key to the map
		valueMap["_key"] = key

		// Check if the value matches the filter
		if matchesFilter(valueMap, filter) {
			results = append(results, valueMap)
		}
	}

	// Sort results if sort field is provided
	if sort != "" {
		sortField := sort
		ascending := true

		if strings.HasPrefix(sort, "-") {
			sortField = sort[1:]
			ascending = false
		}

		sortResults(results, sortField, ascending)
	}

	// Apply limit if provided
	if limit > 0 && limit < len(results) {
		results = results[:limit]
	}

	return results
}

// matchesFilter checks if a value matches a filter
func matchesFilter(value map[string]interface{}, filter map[string]interface{}) bool {
	for k, v := range filter {
		fieldValue, exists := value[k]
		if !exists {
			return false
		}

		// Handle different comparison operators
		if filterMap, ok := v.(map[string]interface{}); ok {
			for op, opValue := range filterMap {
				switch op {
				case "$eq":
					if fieldValue != opValue {
						return false
					}
				case "$ne":
					if fieldValue == opValue {
						return false
					}
				case "$gt":
					if !isGreaterThan(fieldValue, opValue) {
						return false
					}
				case "$gte":
					if !isGreaterThanOrEqual(fieldValue, opValue) {
						return false
					}
				case "$lt":
					if !isLessThan(fieldValue, opValue) {
						return false
					}
				case "$lte":
					if !isLessThanOrEqual(fieldValue, opValue) {
						return false
					}
				case "$in":
					if !isInArray(fieldValue, opValue) {
						return false
					}
				case "$contains":
					if !containsValue(fieldValue, opValue) {
						return false
					}
				}
			}
		} else if fieldValue != v {
			return false
		}
	}

	return true
}

// Helper functions for comparisons
func isGreaterThan(a, b interface{}) bool {
	switch aVal := a.(type) {
	case int:
		if bVal, ok := b.(int); ok {
			return aVal > bVal
		}
	case float64:
		if bVal, ok := b.(float64); ok {
			return aVal > bVal
		}
	case string:
		if bVal, ok := b.(string); ok {
			return aVal > bVal
		}
	}
	return false
}

func isGreaterThanOrEqual(a, b interface{}) bool {
	switch aVal := a.(type) {
	case int:
		if bVal, ok := b.(int); ok {
			return aVal >= bVal
		}
	case float64:
		if bVal, ok := b.(float64); ok {
			return aVal >= bVal
		}
	case string:
		if bVal, ok := b.(string); ok {
			return aVal >= bVal
		}
	}
	return false
}

func isLessThan(a, b interface{}) bool {
	switch aVal := a.(type) {
	case int:
		if bVal, ok := b.(int); ok {
			return aVal < bVal
		}
	case float64:
		if bVal, ok := b.(float64); ok {
			return aVal < bVal
		}
	case string:
		if bVal, ok := b.(string); ok {
			return aVal < bVal
		}
	}
	return false
}

func isLessThanOrEqual(a, b interface{}) bool {
	switch aVal := a.(type) {
	case int:
		if bVal, ok := b.(int); ok {
			return aVal <= bVal
		}
	case float64:
		if bVal, ok := b.(float64); ok {
			return aVal <= bVal
		}
	case string:
		if bVal, ok := b.(string); ok {
			return aVal <= bVal
		}
	}
	return false
}

func isInArray(value, array interface{}) bool {
	arr, ok := array.([]interface{})
	if !ok {
		return false
	}

	for _, item := range arr {
		if value == item {
			return true
		}
	}

	return false
}

func containsValue(container, value interface{}) bool {
	switch c := container.(type) {
	case string:
		if v, ok := value.(string); ok {
			return strings.Contains(c, v)
		}
	case []interface{}:
		for _, item := range c {
			if item == value {
				return true
			}
		}
	}
	return false
}

// sortResults sorts the results by the given field
func sortResults(results []map[string]interface{}, field string, ascending bool) {
	sort.Slice(results, func(i, j int) bool {
		a := results[i][field]
		b := results[j][field]

		if ascending {
			return compareValues(a, b) < 0
		}
		return compareValues(a, b) > 0
	})
}

// compareValues compares two values
func compareValues(a, b interface{}) int {
	switch aVal := a.(type) {
	case string:
		if bVal, ok := b.(string); ok {
			return strings.Compare(aVal, bVal)
		}
	case int:
		if bVal, ok := b.(int); ok {
			if aVal < bVal {
				return -1
			} else if aVal > bVal {
				return 1
			}
			return 0
		}
	case float64:
		if bVal, ok := b.(float64); ok {
			if aVal < bVal {
				return -1
			} else if aVal > bVal {
				return 1
			}
			return 0
		}
	case bool:
		if bVal, ok := b.(bool); ok {
			if aVal == bVal {
				return 0
			} else if aVal {
				return 1
			}
			return -1
		}
	}

	// If types are different or not comparable, compare string representations
	return strings.Compare(fmt.Sprintf("%v", a), fmt.Sprintf("%v", b))
}

// LoadFromFile loads the store from a file
func (sm *StoreManager) LoadFromFile() error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Check if file exists
	if _, err := os.Stat(sm.filePath); os.IsNotExist(err) {
		return nil // File doesn't exist, nothing to load
	}

	// Read file
	data, err := ioutil.ReadFile(sm.filePath)
	if err != nil {
		return fmt.Errorf("failed to read store file: %w", err)
	}

	// Parse JSON
	var fileData struct {
		Store    map[string]interface{} `json:"store"`
		TTLStore map[string]string      `json:"ttl_store"`
	}

	if err := json.Unmarshal(data, &fileData); err != nil {
		return fmt.Errorf("failed to parse store file: %w", err)
	}

	// Update store
	sm.memStore = fileData.Store

	// Update TTL store
	sm.ttlStore = make(map[string]time.Time)
	for key, timeStr := range fileData.TTLStore {
		if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
			sm.ttlStore[key] = t
		}
	}

	return nil
}

// SaveToFile saves the store to a file
func (sm *StoreManager) SaveToFile() error {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return sm.saveToFile()
}

// saveToFile is an internal method that saves the store to a file without locking
func (sm *StoreManager) saveToFile() error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(sm.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Prepare data for serialization
	fileData := struct {
		Store    map[string]interface{} `json:"store"`
		TTLStore map[string]string      `json:"ttl_store"`
	}{
		Store:    sm.memStore,
		TTLStore: make(map[string]string),
	}

	// Convert TTL times to strings
	for key, t := range sm.ttlStore {
		fileData.TTLStore[key] = t.Format(time.RFC3339)
	}

	// Serialize to JSON
	data, err := json.MarshalIndent(fileData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize store: %w", err)
	}

	// Write to file
	if err := ioutil.WriteFile(sm.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write store file: %w", err)
	}

	return nil
}

// SetFilePath sets the file path for persistent storage
func (sm *StoreManager) SetFilePath(path string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.filePath = path
}

// SetAutoSave sets the auto-save flag
func (sm *StoreManager) SetAutoSave(autoSave bool) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.autoSave = autoSave
}

// Increment increments a numeric value
func (sm *StoreManager) Increment(key string, amount float64) (float64, error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Get current value
	currentValue, exists := sm.memStore[key]

	var newValue float64

	if !exists {
		// Key doesn't exist, initialize with amount
		newValue = amount
	} else {
		// Convert current value to float64
		switch v := currentValue.(type) {
		case int:
			newValue = float64(v) + amount
		case float64:
			newValue = v + amount
		case string:
			// Try to parse string as float
			var floatVal float64
			if _, err := fmt.Sscanf(v, "%f", &floatVal); err != nil {
				return 0, fmt.Errorf("cannot increment non-numeric value: %v", currentValue)
			}
			newValue = floatVal + amount
		default:
			return 0, fmt.Errorf("cannot increment non-numeric value: %v", currentValue)
		}
	}

	// Update store
	sm.memStore[key] = newValue

	if sm.autoSave {
		sm.saveToFile()
	}

	return newValue, nil
}

// Append appends a value to an array
func (sm *StoreManager) Append(key string, value interface{}) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Get current value
	currentValue, exists := sm.memStore[key]

	if !exists {
		// Key doesn't exist, initialize with array containing value
		sm.memStore[key] = []interface{}{value}
	} else {
		// Check if current value is an array
		arr, ok := currentValue.([]interface{})
		if !ok {
			return fmt.Errorf("cannot append to non-array value: %v", currentValue)
		}

		// Append value to array
		arr = append(arr, value)
		sm.memStore[key] = arr
	}

	if sm.autoSave {
		sm.saveToFile()
	}

	return nil
}

// NewEnhancedStoreNodeWrapper creates a new enhanced store node wrapper
func NewEnhancedStoreNodeWrapper(params map[string]interface{}) (flowlib.Node, error) {
	// Create the base node
	baseNode := flowlib.NewNode(1, 0)

	// Get store manager
	storeManager := GetStoreManager()

	// Configure store manager
	if filePath, ok := params["file_path"].(string); ok && filePath != "" {
		storeManager.SetFilePath(filePath)
	}

	if autoSave, ok := params["auto_save"].(bool); ok {
		storeManager.SetAutoSave(autoSave)
	}

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

			operation, ok := params["operation"].(string)
			if !ok {
				return nil, fmt.Errorf("operation parameter is required")
			}

			switch operation {
			case "get":
				key, ok := params["key"].(string)
				if !ok {
					return nil, fmt.Errorf("key parameter is required for get operation")
				}

				value, exists := storeManager.Get(key)
				if !exists {
					return nil, fmt.Errorf("key not found: %s", key)
				}

				return value, nil

			case "set":
				key, ok := params["key"].(string)
				if !ok {
					return nil, fmt.Errorf("key parameter is required for set operation")
				}

				value, ok := params["value"]
				if !ok {
					return nil, fmt.Errorf("value parameter is required for set operation")
				}

				// Parse TTL if provided
				var ttl time.Duration
				if ttlStr, ok := params["ttl"].(string); ok {
					var err error
					ttl, err = time.ParseDuration(ttlStr)
					if err != nil {
						return nil, fmt.Errorf("invalid ttl format: %w", err)
					}
				}

				storeManager.Set(key, value, ttl)
				return value, nil

			case "delete":
				key, ok := params["key"].(string)
				if !ok {
					return nil, fmt.Errorf("key parameter is required for delete operation")
				}

				exists := storeManager.Delete(key)
				return map[string]interface{}{
					"deleted": exists,
					"key":     key,
				}, nil

			case "list":
				keys := storeManager.List()
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

				results := storeManager.Query(filter, sortField, limit)
				return results, nil

			case "increment":
				key, ok := params["key"].(string)
				if !ok {
					return nil, fmt.Errorf("key parameter is required for increment operation")
				}

				amount := 1.0
				if amountParam, ok := params["amount"].(float64); ok {
					amount = amountParam
				}

				newValue, err := storeManager.Increment(key, amount)
				if err != nil {
					return nil, err
				}

				return map[string]interface{}{
					"key":         key,
					"new_value":   newValue,
					"incremented": amount,
				}, nil

			case "append":
				key, ok := params["key"].(string)
				if !ok {
					return nil, fmt.Errorf("key parameter is required for append operation")
				}

				value, ok := params["value"]
				if !ok {
					return nil, fmt.Errorf("value parameter is required for append operation")
				}

				if err := storeManager.Append(key, value); err != nil {
					return nil, err
				}

				return map[string]interface{}{
					"key":      key,
					"appended": value,
				}, nil

			case "save":
				if err := storeManager.SaveToFile(); err != nil {
					return nil, fmt.Errorf("failed to save store: %w", err)
				}

				return map[string]interface{}{
					"saved": true,
					"file":  storeManager.filePath,
				}, nil

			case "load":
				if err := storeManager.LoadFromFile(); err != nil {
					return nil, fmt.Errorf("failed to load store: %w", err)
				}

				return map[string]interface{}{
					"loaded": true,
					"file":   storeManager.filePath,
				}, nil

			default:
				return nil, fmt.Errorf("unknown operation: %s", operation)
			}
		},
	}

	// Set the parameters
	wrapper.SetParams(params)

	return wrapper, nil
}

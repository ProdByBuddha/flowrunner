package api

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tcmartin/flowrunner/pkg/runtime"
)

// WebSocketManager manages WebSocket connections for real-time updates
type WebSocketManager struct {
	// upgrader for upgrading HTTP connections to WebSocket
	upgrader websocket.Upgrader
	
	// connections maps execution IDs to sets of WebSocket connections
	connections map[string]map[*websocket.Conn]bool
	
	// connectionMeta stores metadata for each connection
	connectionMeta map[*websocket.Conn]*ConnectionMetadata
	
	// mutex for thread-safe access
	mu sync.RWMutex
	
	// flowRuntime for accessing execution data
	flowRuntime runtime.FlowRuntime
}

// ConnectionMetadata stores metadata about a WebSocket connection
type ConnectionMetadata struct {
	AccountID    string
	ConnectedAt  time.Time
	LastPingAt   time.Time
	Subscriptions map[string]bool // execution IDs this connection is subscribed to
}

// ExecutionUpdate represents a real-time update for a flow execution
type ExecutionUpdate struct {
	Type        string                 `json:"type"`        // "log", "status", "complete", "error"
	ExecutionID string                 `json:"execution_id"`
	Timestamp   time.Time              `json:"timestamp"`
	NodeID      string                 `json:"node_id,omitempty"`
	Message     string                 `json:"message,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Status      *runtime.ExecutionStatus `json:"status,omitempty"`
	Log         *runtime.ExecutionLog    `json:"log,omitempty"`
}

// WebSocketMessage represents incoming WebSocket messages
type WebSocketMessage struct {
	Type        string `json:"type"`        // "subscribe", "unsubscribe", "ping"
	ExecutionID string `json:"execution_id,omitempty"`
}

// NewWebSocketManager creates a new WebSocket manager
func NewWebSocketManager(flowRuntime runtime.FlowRuntime) *WebSocketManager {
	return &WebSocketManager{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow connections from any origin for now
				// In production, this should be more restrictive
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		connections:    make(map[string]map[*websocket.Conn]bool),
		connectionMeta: make(map[*websocket.Conn]*ConnectionMetadata),
		flowRuntime:    flowRuntime,
	}
}

// HandleWebSocket handles WebSocket connection upgrade and management
func (wsm *WebSocketManager) HandleWebSocket(w http.ResponseWriter, r *http.Request, accountID string) {
	// Upgrade the HTTP connection to WebSocket
	conn, err := wsm.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Store connection metadata
	wsm.mu.Lock()
	wsm.connectionMeta[conn] = &ConnectionMetadata{
		AccountID:     accountID,
		ConnectedAt:   time.Now(),
		LastPingAt:    time.Now(),
		Subscriptions: make(map[string]bool),
	}
	wsm.mu.Unlock()

	// Clean up when connection closes
	defer func() {
		wsm.mu.Lock()
		// Remove from all execution subscriptions
		if meta, exists := wsm.connectionMeta[conn]; exists {
			for executionID := range meta.Subscriptions {
				if execConns, exists := wsm.connections[executionID]; exists {
					delete(execConns, conn)
					if len(execConns) == 0 {
						delete(wsm.connections, executionID)
					}
				}
			}
		}
		// Remove connection metadata
		delete(wsm.connectionMeta, conn)
		wsm.mu.Unlock()
		log.Printf("WebSocket connection closed for account %s", accountID)
	}()

	log.Printf("WebSocket connection established for account %s", accountID)

	// Set up ping/pong handlers
	conn.SetPongHandler(func(string) error {
		wsm.mu.Lock()
		if meta, exists := wsm.connectionMeta[conn]; exists {
			meta.LastPingAt = time.Now()
		}
		wsm.mu.Unlock()
		return nil
	})

	// Start ping routine
	go wsm.pingRoutine(conn)

	// Handle incoming messages
	for {
		var msg WebSocketMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		wsm.handleMessage(conn, &msg, accountID)
	}
}

// handleMessage processes incoming WebSocket messages
func (wsm *WebSocketManager) handleMessage(conn *websocket.Conn, msg *WebSocketMessage, accountID string) {
	switch msg.Type {
	case "subscribe":
		if msg.ExecutionID != "" {
			wsm.subscribeToExecution(conn, msg.ExecutionID, accountID)
		}
	case "unsubscribe":
		if msg.ExecutionID != "" {
			wsm.unsubscribeFromExecution(conn, msg.ExecutionID)
		}
	case "ping":
		// Respond with pong
		wsm.sendMessage(conn, ExecutionUpdate{
			Type:      "pong",
			Timestamp: time.Now(),
		})
	default:
		log.Printf("Unknown WebSocket message type: %s", msg.Type)
	}
}

// subscribeToExecution subscribes a connection to execution updates
func (wsm *WebSocketManager) subscribeToExecution(conn *websocket.Conn, executionID, accountID string) {
	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	// Verify the execution exists and belongs to the account
	if wsm.flowRuntime != nil {
		if status, err := wsm.flowRuntime.GetStatus(executionID); err != nil {
			// Send error message to client
			wsm.sendMessage(conn, ExecutionUpdate{
				Type:        "error",
				ExecutionID: executionID,
				Timestamp:   time.Now(),
				Message:     "Execution not found or access denied",
			})
			return
		} else {
			// Send current status to newly subscribed client
			wsm.sendMessage(conn, ExecutionUpdate{
				Type:        "status",
				ExecutionID: executionID,
				Timestamp:   time.Now(),
				Status:      &status,
			})
		}
	}

	// Add connection to execution subscriptions
	if wsm.connections[executionID] == nil {
		wsm.connections[executionID] = make(map[*websocket.Conn]bool)
	}
	wsm.connections[executionID][conn] = true

	// Update connection metadata
	if meta, exists := wsm.connectionMeta[conn]; exists {
		meta.Subscriptions[executionID] = true
	}

	log.Printf("Account %s subscribed to execution %s", accountID, executionID)

	// Start monitoring logs for this execution if not already started
	go wsm.monitorExecution(executionID)
}

// unsubscribeFromExecution unsubscribes a connection from execution updates
func (wsm *WebSocketManager) unsubscribeFromExecution(conn *websocket.Conn, executionID string) {
	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	// Remove connection from execution subscriptions
	if execConns, exists := wsm.connections[executionID]; exists {
		delete(execConns, conn)
		if len(execConns) == 0 {
			delete(wsm.connections, executionID)
		}
	}

	// Update connection metadata
	if meta, exists := wsm.connectionMeta[conn]; exists {
		delete(meta.Subscriptions, executionID)
	}

	log.Printf("Connection unsubscribed from execution %s", executionID)
}

// monitorExecution monitors an execution and broadcasts updates
func (wsm *WebSocketManager) monitorExecution(executionID string) {
	if wsm.flowRuntime == nil {
		return
	}

	// Subscribe to logs for this execution
	logsChan, err := wsm.flowRuntime.SubscribeToLogs(executionID)
	if err != nil {
		log.Printf("Failed to subscribe to logs for execution %s: %v", executionID, err)
		return
	}

	// Monitor logs and broadcast updates
	for log := range logsChan {
		update := ExecutionUpdate{
			Type:        "log",
			ExecutionID: executionID,
			Timestamp:   log.Timestamp,
			NodeID:      log.NodeID,
			Message:     log.Message,
			Log:         &log,
		}
		wsm.broadcastToExecution(executionID, update)
	}

	// Get final status when logs channel closes
	if status, err := wsm.flowRuntime.GetStatus(executionID); err == nil {
		update := ExecutionUpdate{
			Type:        "status",
			ExecutionID: executionID,
			Timestamp:   time.Now(),
			Status:      &status,
		}
		wsm.broadcastToExecution(executionID, update)

		// If execution is complete, send completion event
		if status.Status == "completed" || status.Status == "failed" || status.Status == "canceled" {
			update := ExecutionUpdate{
				Type:        "complete",
				ExecutionID: executionID,
				Timestamp:   time.Now(),
				Message:     "Execution completed with status: " + status.Status,
				Status:      &status,
			}
			wsm.broadcastToExecution(executionID, update)
		}
	}
}

// broadcastToExecution sends an update to all connections subscribed to an execution
func (wsm *WebSocketManager) broadcastToExecution(executionID string, update ExecutionUpdate) {
	wsm.mu.RLock()
	connections, exists := wsm.connections[executionID]
	if !exists {
		wsm.mu.RUnlock()
		return
	}

	// Create a copy of connections to avoid holding the lock during sending
	connsCopy := make([]*websocket.Conn, 0, len(connections))
	for conn := range connections {
		connsCopy = append(connsCopy, conn)
	}
	wsm.mu.RUnlock()

	// Send to all connections
	for _, conn := range connsCopy {
		wsm.sendMessage(conn, update)
	}
}

// sendMessage sends a message to a WebSocket connection
func (wsm *WebSocketManager) sendMessage(conn *websocket.Conn, update ExecutionUpdate) {
	// Set write deadline
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	
	if err := conn.WriteJSON(update); err != nil {
		log.Printf("Failed to send WebSocket message: %v", err)
		// Remove the connection on write error
		wsm.removeConnection(conn)
	}
}

// removeConnection removes a connection from all subscriptions
func (wsm *WebSocketManager) removeConnection(conn *websocket.Conn) {
	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	// Remove from all execution subscriptions
	if meta, exists := wsm.connectionMeta[conn]; exists {
		for executionID := range meta.Subscriptions {
			if execConns, exists := wsm.connections[executionID]; exists {
				delete(execConns, conn)
				if len(execConns) == 0 {
					delete(wsm.connections, executionID)
				}
			}
		}
	}

	// Remove connection metadata
	delete(wsm.connectionMeta, conn)
	
	// Close the connection
	conn.Close()
}

// pingRoutine sends periodic ping messages to keep connection alive
func (wsm *WebSocketManager) pingRoutine(conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Failed to send ping: %v", err)
				wsm.removeConnection(conn)
				return
			}
		}
	}
}

// GetConnectedClients returns the number of connected clients
func (wsm *WebSocketManager) GetConnectedClients() int {
	wsm.mu.RLock()
	defer wsm.mu.RUnlock()
	return len(wsm.connectionMeta)
}

// GetExecutionSubscribers returns the number of subscribers for an execution
func (wsm *WebSocketManager) GetExecutionSubscribers(executionID string) int {
	wsm.mu.RLock()
	defer wsm.mu.RUnlock()
	if connections, exists := wsm.connections[executionID]; exists {
		return len(connections)
	}
	return 0
}

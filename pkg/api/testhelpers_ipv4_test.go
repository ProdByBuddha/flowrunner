package api

import (
    "net"
    "net/http"
    "net/http/httptest"
)

// NewIPv4Server starts an httptest server bound to IPv4 localhost. This avoids
// environments that disallow binding to ::1 during tests.
func NewIPv4Server(handler http.Handler) *httptest.Server {
    srv := httptest.NewUnstartedServer(handler)
    ln, err := net.Listen("tcp4", "127.0.0.1:0")
    if err == nil {
        srv.Listener = ln
        srv.Start()
        return srv
    }
    // Fallback to default behavior if IPv4 bind fails for any reason
    srv = httptest.NewServer(handler)
    return srv
}


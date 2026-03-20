package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	garage "git.deuxfleurs.fr/garage-sdk/garage-admin-sdk-golang"
)

// testGarageServer wraps an httptest.Server with Garage API route handling.
type testGarageServer struct {
	Server *httptest.Server
	Mux    *http.ServeMux
}

// newTestGarageServer creates a test server with a configurable mux.
// Register handlers on srv.Mux before using srv.Client().
func newTestGarageServer(t *testing.T) *testGarageServer {
	t.Helper()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return &testGarageServer{Server: server, Mux: mux}
}

// Client returns a GarageClient configured to talk to the test server.
func (s *testGarageServer) Client(t *testing.T) *GarageClient {
	t.Helper()
	cfg := garage.NewConfiguration()
	cfg.Scheme = "http"
	cfg.Host = strings.TrimPrefix(s.Server.URL, "http://")
	cfg.DefaultHeader["Authorization"] = "Bearer test-token"
	client := garage.NewAPIClient(cfg)
	return &GarageClient{Client: client}
}

// respondJSON writes a JSON response for a registered handler.
func respondJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

// respondNoContent writes a 204 No Content response.
func respondNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// respondOK writes a 200 OK with optional body.
func respondOK(w http.ResponseWriter, body string) {
	w.WriteHeader(http.StatusOK)
	if body != "" {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}
}

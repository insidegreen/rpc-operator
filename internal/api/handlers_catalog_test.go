package api_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestHandlerCatalog_ListReturnsAllItems(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/catalog")
	if err != nil {
		t.Fatalf("GET catalog: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var result struct {
		Items []json.RawMessage `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// 9 originals + 9 new (http_client×2, nats×2, aws_s3×2, sql_select, sql_insert, sql_raw)
	// + 4 NATS (nats_jetstream×2, nats_kv×2) + 1 (nats_request_reply processor) = 23
	if len(result.Items) != 23 {
		t.Errorf("expected 23 items, got %d", len(result.Items))
	}
}

func TestHandlerCatalog_GetMappingProcessor(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/catalog/processors/mapping")
	if err != nil {
		t.Fatalf("GET catalog entry: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var comp map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&comp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if string(comp["bodyKind"]) != `"scalar"` {
		t.Errorf("expected bodyKind=scalar, got %s", comp["bodyKind"])
	}
}

func TestHandlerCatalog_GetNotFound(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/catalog/inputs/no-such")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// TestHandlerCatalog_WrongMethodReturnsCatchAll documents that with the SPA
// catch-all handler in place, a POST to the catalog path hits the static file
// server (no matching file) and returns 404, not 405.
func TestHandlerCatalog_WrongMethodReturnsCatchAll(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/catalog", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST catalog: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 (static catch-all), got %d", resp.StatusCode)
	}
}

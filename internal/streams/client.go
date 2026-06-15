/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

// Package streams talks to the per-pod Redpanda Connect streams HTTP API
// (POST/PUT/DELETE/GET /streams/{id}) used by F47 PipelineClusters.
package streams

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ConfigRejectedError is returned by EnsureStream/EnsureCacheResource when the
// streams API rejects the config permanently: a 4xx (lint errors) or a 502 whose
// body reports a component initialisation failure (see configRejected). Retrying
// the identical config against the same cluster state fails the same way, so
// callers should surface it (record it in status) rather than requeue. Transport
// failures and bodyless/other 5xx are returned as plain errors because they are
// transient and worth retrying.
type ConfigRejectedError struct {
	StreamID string
	Status   int
	Body     string
}

func (e *ConfigRejectedError) Error() string {
	return fmt.Sprintf("ensure stream %s: status %d: %s", e.StreamID, e.Status, e.Body)
}

// configRejected reports whether a non-2xx streams-API response is a permanent
// config rejection (vs. a transient error worth retrying). A 4xx is always a
// rejection (lint errors). Redpanda Connect also reports component
// initialisation failures as 502 with a "failed to init" body — e.g. a
// multilevel cache with fewer than two levels, or an output/processor
// referencing a cache resource that is not registered. These are permanent for
// an identical config + cluster state, unlike a bodyless gateway 502 from a
// restarting pod, which stays transient.
func configRejected(status int, body string) bool {
	if status >= 400 && status < 500 {
		return true
	}
	if status == http.StatusBadGateway && strings.Contains(body, "failed to init") {
		return true
	}
	return false
}

// ErrStreamNotFound is returned by GetStreamStatus when the instance reports the
// stream does not exist (HTTP 404). Callers use it to distinguish a vanished
// stream from a transport/5xx failure.
var ErrStreamNotFound = errors.New("stream not found")

// StreamStatus is the subset of GET /streams/{id} the operator consumes.
type StreamStatus struct {
	Active bool
	Uptime float64 // seconds; from the "uptime" field
}

// Client manages streams on a single Redpanda Connect instance addressed by its
// base URL (e.g. http://etl-small-1.etl-small.ns.svc:4195).
type Client interface {
	// EnsureStream upserts a stream with the given id and config (PUT is idempotent).
	EnsureStream(ctx context.Context, podBaseURL, streamID, configYAML string) error
	// DeleteStream removes a stream; a 404 (already gone) is treated as success.
	DeleteStream(ctx context.Context, podBaseURL, streamID string) error
	// ListStreams returns the set of stream ids currently running on the instance.
	ListStreams(ctx context.Context, podBaseURL string) (map[string]struct{}, error)
	// GetStreamStatus reads one stream's runtime status. A 404 returns
	// (StreamStatus{}, ErrStreamNotFound); other non-2xx returns a plain error.
	GetStreamStatus(ctx context.Context, podBaseURL, streamID string) (StreamStatus, error)
	// EnsureCacheResource upserts a cache resource (POST /resources/cache/{label}).
	// A 4xx (lint) or a 502 init failure returns *ConfigRejectedError (see
	// configRejected); other 5xx/transport errors are plain (transient) errors.
	EnsureCacheResource(ctx context.Context, podBaseURL, label, configYAML string) error
	// DeleteCacheResource is a no-op on HTTPClient: DELETE is not supported by the
	// RPC streams API. The FakeClient records the removal so controller tests work.
	DeleteCacheResource(ctx context.Context, podBaseURL, label string) error
}

// HTTPClient is the production Client over HTTP.
type HTTPClient struct {
	HTTP *http.Client
}

var _ Client = (*HTTPClient)(nil)

// NewHTTPClient returns an HTTPClient with a sane request timeout.
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{HTTP: &http.Client{Timeout: 10 * time.Second}}
}

// EnsureStream upserts a stream. The Redpanda Connect streams API uses POST to
// create and PUT to update; a PUT on a stream that does not exist yet returns
// 404. So we PUT first (the common case once a stream exists) and fall back to
// POST when the stream is absent.
func (c *HTTPClient) EnsureStream(ctx context.Context, podBaseURL, streamID, configYAML string) error {
	status, body, err := c.streamReq(ctx, http.MethodPut, podBaseURL, streamID, configYAML)
	if err != nil {
		return err
	}
	if status == http.StatusNotFound {
		status, body, err = c.streamReq(ctx, http.MethodPost, podBaseURL, streamID, configYAML)
		if err != nil {
			return err
		}
	}
	if configRejected(status, body) {
		return &ConfigRejectedError{StreamID: streamID, Status: status, Body: body}
	}
	if status >= 300 {
		return fmt.Errorf("ensure stream %s: status %d: %s", streamID, status, body)
	}
	return nil
}

// streamReq sends one config-bearing request to /streams/{id}, drains the
// response body, and returns the status code and body text.
func (c *HTTPClient) streamReq(ctx context.Context, method, podBaseURL, streamID, configYAML string) (int, string, error) {
	url := fmt.Sprintf("%s/streams/%s", strings.TrimRight(podBaseURL, "/"), streamID)
	req, err := http.NewRequestWithContext(ctx, method, url, strings.NewReader(configYAML))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/x-yaml")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("%s stream %s: %w", method, streamID, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body), nil
}

func (c *HTTPClient) DeleteStream(ctx context.Context, podBaseURL, streamID string) error {
	url := fmt.Sprintf("%s/streams/%s", strings.TrimRight(podBaseURL, "/"), streamID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE stream %s: %w", streamID, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotFound {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("DELETE stream %s: status %d: %s", streamID, resp.StatusCode, string(body))
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

func (c *HTTPClient) ListStreams(ctx context.Context, podBaseURL string) (map[string]struct{}, error) {
	url := fmt.Sprintf("%s/streams", strings.TrimRight(podBaseURL, "/"))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET streams: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GET streams: status %d: %s", resp.StatusCode, string(body))
	}
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode streams: %w", err)
	}
	out := make(map[string]struct{}, len(raw))
	for id := range raw {
		out[id] = struct{}{}
	}
	return out, nil
}

// EnsureCacheResource upserts a cache resource via POST /resources/cache/{label}.
// POST is the only supported verb (spike-verified: PUT/DELETE return "verb not supported").
func (c *HTTPClient) EnsureCacheResource(ctx context.Context, podBaseURL, label, configYAML string) error {
	url := fmt.Sprintf("%s/resources/cache/%s", strings.TrimRight(podBaseURL, "/"), label)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(configYAML))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-yaml")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("POST cache resource %s: %w", label, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if configRejected(resp.StatusCode, string(body)) {
		return &ConfigRejectedError{StreamID: label, Status: resp.StatusCode, Body: string(body)}
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("POST cache resource %s: status %d: %s", label, resp.StatusCode, string(body))
	}
	return nil
}

// DeleteCacheResource is a no-op: DELETE /resources/cache/{label} is not supported
// by the RPC streams API. Removal is effected by deleting the NATS KV bucket and
// not re-pushing the resource; the instance in-memory registration clears on restart.
func (c *HTTPClient) DeleteCacheResource(_ context.Context, _, _ string) error {
	return nil
}

func (c *HTTPClient) GetStreamStatus(ctx context.Context, podBaseURL, streamID string) (StreamStatus, error) {
	url := fmt.Sprintf("%s/streams/%s", strings.TrimRight(podBaseURL, "/"), streamID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return StreamStatus{}, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return StreamStatus{}, fmt.Errorf("GET stream %s: %w", streamID, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotFound {
		_, _ = io.Copy(io.Discard, resp.Body)
		return StreamStatus{}, ErrStreamNotFound
	}
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return StreamStatus{}, fmt.Errorf("GET stream %s: status %d: %s", streamID, resp.StatusCode, string(body))
	}
	var raw struct {
		Active bool    `json:"active"`
		Uptime float64 `json:"uptime"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return StreamStatus{}, fmt.Errorf("decode stream %s status: %w", streamID, err)
	}
	return StreamStatus{Active: raw.Active, Uptime: raw.Uptime}, nil
}

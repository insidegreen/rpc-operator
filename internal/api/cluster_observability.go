package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
)

// Cluster-mode observability constants. F47 Phase 3a.
// streamLogField / streamMetricLabel are confirmed by the ds9s3 spike
// (see docs/test/f47-streams/phase3a-spike.md); change here if the spike differs.
const (
	streamLogField    = "stream" // JSON log field carrying the stream id (= pipeline name)
	streamMetricLabel = "stream" // Prometheus label carrying the stream id

	clusterLogPodWindow    = 2000 // pod-lines tailed for the cluster-mode log backlog
	clusterLogBacklogLines = 200  // max filtered backlog lines sent before live follow
)

// streamLogMatch reports whether line is a JSON object whose streamLogField
// value is exactly streamID. Non-JSON, non-object, missing/non-string/mismatched
// field → false (strict per-stream filter). F47 Phase 3a.
func streamLogMatch(line []byte, streamID string) bool {
	if !bytes.HasPrefix(bytes.TrimSpace(line), []byte("{")) {
		return false
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(line, &obj); err != nil {
		return false
	}
	raw, ok := obj[streamLogField]
	if !ok {
		return false
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return false
	}
	return s == streamID
}

// filterBacklog reads newline-delimited log lines from r, keeps those matching
// streamID, and returns at most the last capN matches (oldest→newest). Used for
// the cluster-mode log backlog. F47 Phase 3a.
func filterBacklog(r io.Reader, streamID string, capN int) [][]byte {
	var matches [][]byte
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if streamLogMatch(line, streamID) {
			cp := append([]byte(nil), line...) // scanner reuses the buffer
			matches = append(matches, cp)
		}
	}
	if len(matches) > capN {
		matches = matches[len(matches)-capN:]
	}
	return matches
}

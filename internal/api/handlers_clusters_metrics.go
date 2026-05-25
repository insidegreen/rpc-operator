package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	rpcv1alpha1 "github.com/insidegreen/rpc-operator-claude/api/v1alpha1"
)

// handleClusterMetrics returns cluster-wide aggregate metrics (sum across all
// instances). Mirrors handleMetrics but builds a sum() query via
// buildClusterMetricQuery. F47 Phase 3b.
func (s *Server) handleClusterMetrics(w http.ResponseWriter, r *http.Request) {
	ns := r.PathValue("namespace")
	name := r.PathValue("name")
	queryName := r.URL.Query().Get("query")
	start := r.URL.Query().Get("start")
	end := r.URL.Query().Get("end")
	step := r.URL.Query().Get("step")

	if step == "" {
		step = "30s"
	}
	now := time.Now().Unix()
	if end == "" {
		end = strconv.FormatInt(now, 10)
	}
	if start == "" {
		start = strconv.FormatInt(now-1800, 10)
	}

	q, ok := knownQueries[queryName]
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "unknown_query",
			fmt.Sprintf("unknown query %q; valid: throughput, error_rate, input_rate, processor_error_rate", queryName))
		return
	}

	c, err := s.clientForRequest(r)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal error", err.Error())
		return
	}
	var cluster rpcv1alpha1.PipelineCluster
	if err := c.Get(r.Context(), client.ObjectKey{Namespace: ns, Name: name}, &cluster); err != nil {
		writeK8sError(w, err)
		return
	}

	if s.PrometheusURL == "" {
		writeJSONError(w, http.StatusServiceUnavailable, "prometheus_unavailable",
			"prometheus is not configured; set --prometheus-url")
		return
	}

	promQL := buildClusterMetricQuery(q.metric, ns, name)
	datapoints, err := s.queryPrometheus(r.Context(), promQL, start, end, step)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "prometheus_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, MetricsResponse{
		Query:      queryName,
		Unit:       q.unit,
		Datapoints: datapoints,
	})
}

package api

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/coder/websocket"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rpcv1alpha1 "github.com/insidegreen/rpc-operator-claude/api/v1alpha1"
)

func (s *Server) handleLogStream(w http.ResponseWriter, r *http.Request) {
	ns := r.PathValue("namespace")
	name := r.PathValue("name")

	// In Mode B the token arrives in `?token=…` because browsers cannot set
	// headers on `new WebSocket(...)`. Verify and inject into context BEFORE
	// the WS upgrade, because after Accept we can no longer write HTTP errors.
	//
	// F42: AnonymousLogs is a SEPARATE flag from AnonymousRead — log content
	// can carry payloads/secrets. Anonymous logs require explicit opt-in.
	if s.AuthEnabled {
		if token := tokenFromRequest(r); token != "" {
			ctx := context.WithValue(r.Context(), tokenContextKey, token)
			r = r.WithContext(ctx)
		} else if s.AnonymousLogs {
			ctx := context.WithValue(r.Context(), anonymousContextKey, true)
			r = r.WithContext(ctx)
		} else {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized", "missing token query parameter")
			return
		}
	}

	// 1. Pipeline holen — HTTP-Fehler sind hier noch möglich (vor WS-Upgrade)
	c, err := s.clientForRequest(r)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal error", err.Error())
		return
	}
	var pipe rpcv1alpha1.Pipeline
	if err := c.Get(r.Context(), client.ObjectKey{Namespace: ns, Name: name}, &pipe); err != nil {
		writeK8sError(w, err)
		return
	}
	// F47 Phase 3a: pick the pod + stream filter from placement.
	targetPod := pipe.Status.PodName
	streamID := ""
	if pipe.Status.AssignedInstance != "" {
		targetPod = pipe.Status.AssignedInstance // cluster mode: the shared instance pod
		streamID = pipe.Name                     // filter the pod's logs to this stream
	}
	if targetPod == "" {
		writeJSONError(w, http.StatusConflict, "no pod", "pipeline has no running pod")
		return
	}

	cs, err := s.clientsetForRequest(r)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal error", err.Error())
		return
	}
	if cs == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "not available", "log streaming not configured")
		return
	}

	// 2. WebSocket-Upgrade — danach keine HTTP-Error-Responses mehr möglich
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // Origin-Check folgt mit F20b (OIDC) oder eigenem Hardening-PRP
	})
	if err != nil {
		return // Accept schreibt selbst die Fehlerantwort
	}
	defer func() { _ = conn.CloseNow() }()

	// CloseRead gibt einen Context zurück, der abbricht wenn der Client trennt
	ctx := conn.CloseRead(r.Context())

	// 3. Pod-Log-Stream öffnen
	if streamID == "" {
		// Pod mode (unchanged): tail 200 lines and follow.
		tailLines := int64(200)
		req := cs.CoreV1().Pods(ns).GetLogs(targetPod, &corev1.PodLogOptions{
			Container: "connect",
			Follow:    true,
			TailLines: &tailLines,
		})
		logStream, err := req.Stream(ctx)
		if err != nil {
			_ = conn.Write(ctx, websocket.MessageText, fmt.Appendf(nil, "error: %v", err))
			_ = conn.Close(websocket.StatusInternalError, "stream open failed")
			return
		}
		defer func() { _ = logStream.Close() }()

		scanner := bufio.NewScanner(logStream)
		for scanner.Scan() {
			if err := conn.Write(ctx, websocket.MessageText, scanner.Bytes()); err != nil {
				return
			}
		}
		_ = conn.Close(websocket.StatusNormalClosure, "")
		return
	}

	// Cluster mode: filter the shared pod's logs down to this stream.
	// Phase 1 — capped backlog from a larger pod-line window.
	start := metav1.NewTime(time.Now())
	backlogTail := int64(clusterLogPodWindow)
	bReq := cs.CoreV1().Pods(ns).GetLogs(targetPod, &corev1.PodLogOptions{
		Container: "connect",
		Follow:    false,
		TailLines: &backlogTail,
	})
	if bStream, err := bReq.Stream(ctx); err == nil {
		for _, line := range filterBacklog(bStream, streamID, clusterLogBacklogLines) {
			if werr := conn.Write(ctx, websocket.MessageText, line); werr != nil {
				_ = bStream.Close()
				return
			}
		}
		_ = bStream.Close()
	} else {
		_ = conn.Write(ctx, websocket.MessageText, fmt.Appendf(nil, "error: %v", err))
		_ = conn.Close(websocket.StatusInternalError, "stream open failed")
		return
	}

	// Phase 2 — live follow from start time, filtered per line.
	lReq := cs.CoreV1().Pods(ns).GetLogs(targetPod, &corev1.PodLogOptions{
		Container: "connect",
		Follow:    true,
		SinceTime: &start,
	})
	logStream, err := lReq.Stream(ctx)
	if err != nil {
		_ = conn.Write(ctx, websocket.MessageText, fmt.Appendf(nil, "error: %v", err))
		_ = conn.Close(websocket.StatusInternalError, "stream open failed")
		return
	}
	defer func() { _ = logStream.Close() }()

	scanner := bufio.NewScanner(logStream)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if !streamLogMatch(scanner.Bytes(), streamID) {
			continue
		}
		if err := conn.Write(ctx, websocket.MessageText, scanner.Bytes()); err != nil {
			return
		}
	}
	_ = conn.Close(websocket.StatusNormalClosure, "")
}

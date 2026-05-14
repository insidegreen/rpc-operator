package api

import (
	"bufio"
	"fmt"
	"net/http"

	"github.com/coder/websocket"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rpcv1alpha1 "github.com/insidegreen/rpc-operator-claude/api/v1alpha1"
)

func (s *Server) handleLogStream(w http.ResponseWriter, r *http.Request) {
	ns := r.PathValue("namespace")
	name := r.PathValue("name")

	// 1. Pipeline holen — HTTP-Fehler sind hier noch möglich (vor WS-Upgrade)
	var pipe rpcv1alpha1.Pipeline
	if err := s.Client.Get(r.Context(), client.ObjectKey{Namespace: ns, Name: name}, &pipe); err != nil {
		writeK8sError(w, err)
		return
	}
	if pipe.Status.PodName == "" {
		writeJSONError(w, http.StatusConflict, "no pod", "pipeline has no running pod")
		return
	}
	if s.Clientset == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "not available", "log streaming not configured")
		return
	}

	// 2. WebSocket-Upgrade — danach keine HTTP-Error-Responses mehr möglich
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // Origin-Check folgt mit F20 (OIDC)
	})
	if err != nil {
		return // Accept schreibt selbst die Fehlerantwort
	}
	defer func() { _ = conn.CloseNow() }()

	// CloseRead gibt einen Context zurück, der abbricht wenn der Client trennt
	ctx := conn.CloseRead(r.Context())

	// 3. Pod-Log-Stream öffnen
	tailLines := int64(200)
	req := s.Clientset.CoreV1().Pods(ns).GetLogs(pipe.Status.PodName, &corev1.PodLogOptions{
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

	// 4. Zeilen zeilenweise an den Client senden
	scanner := bufio.NewScanner(logStream)
	for scanner.Scan() {
		if err := conn.Write(ctx, websocket.MessageText, scanner.Bytes()); err != nil {
			return // Client hat getrennt
		}
	}
	_ = conn.Close(websocket.StatusNormalClosure, "")
}

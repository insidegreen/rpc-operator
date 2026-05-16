package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// contextKey is an un-exported named type so other packages cannot collide
// with the keys we store in request contexts.
type contextKey string

// tokenContextKey holds the Bearer token between the auth middleware and
// the per-request-client helpers.
const tokenContextKey contextKey = "rpc-bearer-token"

// AnonymousUser is the username returned by /api/v1/auth/whoami when auth
// is disabled (Mode A). Mirrors the K8s convention for unauthenticated users.
const AnonymousUser = "system:anonymous"

// tokenFromRequest extracts a Bearer token from the Authorization header,
// or — as a WebSocket fallback — from the `token` query parameter. Returns
// an empty string when no token is present (caller decides on 401).
func tokenFromRequest(r *http.Request) string {
	if h := r.Header.Get("Authorization"); h != "" {
		if tok, ok := strings.CutPrefix(h, "Bearer "); ok {
			return strings.TrimSpace(tok)
		}
	}
	// Browsers cannot set headers on `new WebSocket(...)`, so the WS client
	// passes the token in the URL.
	if t := r.URL.Query().Get("token"); t != "" {
		return t
	}
	return ""
}

// authMiddleware rejects requests without a Bearer token with 401 and stores
// the token in the request context for downstream handlers. Only installed
// when s.AuthEnabled is true — see authIfEnabled.
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := tokenFromRequest(r)
		if token == "" {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized", "missing or empty Bearer token")
			return
		}
		ctx := context.WithValue(r.Context(), tokenContextKey, token)
		next(w, r.WithContext(ctx))
	}
}

// authIfEnabled wraps a handler with authMiddleware only when auth is on.
// In Mode A this returns the handler unchanged — there is no per-request
// `if !enabled` check; the middleware is simply not installed.
func (s *Server) authIfEnabled(next http.HandlerFunc) http.HandlerFunc {
	if !s.AuthEnabled {
		return next
	}
	return s.authMiddleware(next)
}

// clientForRequest returns a controller-runtime client appropriate for the
// caller. Mode A (auth off): the operator-SA cached client. Mode B (auth on):
// a fresh, uncached client built from the user's Bearer token. The Mode B
// client never reads from the manager cache — Cache contents are op-SA-scoped.
func (s *Server) clientForRequest(r *http.Request) (client.Client, error) {
	if !s.AuthEnabled {
		return s.Client, nil
	}
	token, _ := r.Context().Value(tokenContextKey).(string)
	if token == "" {
		return nil, errors.New("no token in context (authMiddleware did not run?)")
	}
	if s.RestConfig == nil || s.Scheme == nil {
		return nil, errors.New("server missing RestConfig/Scheme; cannot build per-request client")
	}
	cfg := rest.CopyConfig(s.RestConfig)
	cfg.BearerToken = token
	cfg.BearerTokenFile = "" // override any in-cluster token file
	cfg.Username = ""
	cfg.Password = ""
	return client.New(cfg, client.Options{Scheme: s.Scheme})
}

// clientsetForRequest returns a client-go Clientset for the caller. Same
// Mode A / Mode B semantics as clientForRequest. Used by handlers that need
// CoreV1 (pod logs, SelfSubjectReview).
func (s *Server) clientsetForRequest(r *http.Request) (*kubernetes.Clientset, error) {
	if !s.AuthEnabled {
		return s.Clientset, nil
	}
	token, _ := r.Context().Value(tokenContextKey).(string)
	if token == "" {
		return nil, errors.New("no token in context (authMiddleware did not run?)")
	}
	if s.RestConfig == nil {
		return nil, errors.New("server missing RestConfig; cannot build per-request clientset")
	}
	cfg := rest.CopyConfig(s.RestConfig)
	cfg.BearerToken = token
	cfg.BearerTokenFile = ""
	cfg.Username = ""
	cfg.Password = ""
	return kubernetes.NewForConfig(cfg)
}

// handleWhoami returns the current user identity. Mode A: anonymous.
// Mode B: result of K8s SelfSubjectReview using the user's token.
func (s *Server) handleWhoami(w http.ResponseWriter, r *http.Request) {
	if !s.AuthEnabled {
		writeJSON(w, http.StatusOK, map[string]any{
			"user":      map[string]any{"name": AnonymousUser},
			"anonymous": true,
		})
		return
	}
	cs, err := s.clientsetForRequest(r)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal error", err.Error())
		return
	}
	res, err := cs.AuthenticationV1().SelfSubjectReviews().Create(r.Context(),
		&authv1.SelfSubjectReview{}, metav1.CreateOptions{})
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized", fmt.Sprintf("token rejected by apiserver: %v", err))
		return
	}
	info := res.Status.UserInfo
	writeJSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"name":   info.Username,
			"uid":    info.UID,
			"groups": info.Groups,
		},
		"anonymous": false,
	})
}

package server

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"vocat/internal/store"
)

// recordAudit writes one security-relevant event to the audit trail. Failures
// are logged but never block the request being audited.
func (s *Server) recordAudit(
	ctx context.Context,
	actor string,
	action string,
	entityType string,
	entityID string,
	outcome string,
	remoteAddr string,
) {
	if s.store == nil {
		return
	}
	_, err := s.store.AppendAuditEvent(ctx, store.AuditEvent{
		Actor:      actor,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		Outcome:    outcome,
		RemoteAddr: remoteAddr,
		CreatedAt:  time.Now().UTC(),
	})
	if err != nil {
		s.logger.Warn("write audit event failed", "action", action, "error", err)
	}
}

// audit records an event for an already-authenticated request, resolving the
// actor from the session and the source address from the raw connection (proxy
// headers are deliberately not trusted for the audit trail).
func (s *Server) audit(r *http.Request, action string, entityType string, entityID string, outcome string) {
	actor := ""
	if s.auth != nil {
		if cookie, err := r.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
			if session, authErr := s.auth.Authenticate(r.Context(), cookie.Value); authErr == nil {
				actor = session.Principal.Username
			}
		}
	}
	s.recordAudit(r.Context(), actor, action, entityType, entityID, outcome, requestRemoteHost(r))
}

// auditAuth records an authentication event where no session exists yet (the
// actor is the username that was attempted).
func (s *Server) auditAuth(r *http.Request, username string, outcome string) {
	s.recordAudit(r.Context(), username, "auth.login", "session", username, outcome, requestRemoteHost(r))
}

func requestRemoteHost(r *http.Request) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return host
}

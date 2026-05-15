package manager

import (
	"strings"

	"github.com/oneliang/aura/session/pkg/model"
)

// Router handles session routing based on subscription rules.
type Router struct{}

// NewRouter creates a new router.
func NewRouter() *Router {
	return &Router{}
}

// MatchSession finds the best matching session for the given source and content.
// Returns empty string if no match found.
func (r *Router) MatchSession(sessions []*model.Session, source string, content string) string {
	content = strings.ToLower(content)

	for _, session := range sessions {
		for _, sub := range session.Subscriptions {
			if !sub.Active {
				continue
			}

			// Check source match
			// Empty source or "*" matches everything
			if sub.Source != "" && sub.Source != "*" && sub.Source != source {
				continue
			}

			// Check trigger keyword match
			if sub.Trigger != "" && !strings.Contains(content, strings.ToLower(sub.Trigger)) {
				continue
			}

			// Found a match
			return session.ID
		}
	}

	return ""
}

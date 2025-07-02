package barertc

import (
	"git.kirsle.net/apps/barertc/pkg/log"
	"git.kirsle.net/apps/barertc/pkg/messages"
)

// SendPing delivers the Ping message to connected subscribers.
func (sub *Subscriber) SendPing() {
	// Send a ping, and a refreshed JWT token if the user sent one.
	var (
		token string
		rules = map[string]bool{}
	)
	if sub.JWTClaims != nil {
		if jwt, err := sub.JWTClaims.ReSign(); err != nil {
			log.Error("ReSign JWT token for %s#%d: %s", sub.Username, sub.ID, err)
		} else {
			token = jwt
			rules = sub.JWTClaims.Rules.ToDict()
		}
	}

	sub.SendJSON(messages.Message{
		Action:   messages.ActionPing,
		JWTToken: token,
		JWTRules: rules,
	})
}

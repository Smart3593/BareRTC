package barertc

import (
	"git.kirsle.net/apps/barertc/pkg/config"
	"git.kirsle.net/apps/barertc/pkg/log"
)

/*
GetModerationRule loads any moderation rules applied to the user.

Moderation rules can be applied by your chat server (in settings.toml) or provided
by your website (in the custom JWT claims "rules" key).
*/
func (sub *Subscriber) GetModerationRule() *config.ModerationRule {
	// Get server side mod rules to start.
	rules := config.Current.GetModerationRule(sub.Username)
	if rules == nil {
		rules = &config.ModerationRule{}
	}

	// Add in client side (JWT) rules.
	if sub.JWTClaims != nil {
		for _, rule := range sub.JWTClaims.Rules {
			if rule.IsRedCamRule() {
				rules.CameraAlwaysNSFW = true
			}
			if rule.IsNoVideoRule() {
				rules.NoVideo = true
			}
			if rule.IsNoBroadcastRule() {
				rules.NoBroadcast = true
			}
			if rule.IsNoDarkVideoRule() {
				rules.NoDarkVideo = true
			}
		}
	}

	log.Error("GetModerationRule(%s): %+v", sub.Username, rules)

	return rules
}

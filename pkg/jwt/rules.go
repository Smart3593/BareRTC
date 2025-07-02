package jwt

// Rule options for the JWT custom key.
//
// Safely check its settings with the Is() functions which handle superset rules
// which imply other rules, for example novideo > nobroadcast.
type Rule string

// Available Rules your site can include in the JWT token: to enforce moderator
// rules on the user logging into chat.
const (
	// Webcam restrictions.
	NoVideoRule     = Rule("novideo")     // Can not use video features at all
	NoBroadcastRule = Rule("nobroadcast") // They can not share their webcam
	NoImageRule     = Rule("noimage")     // Can not upload or see images
	RedCamRule      = Rule("redcam")      // Their camera is force marked NSFW
	NoDarkVideoRule = Rule("nodvd")       // Exempt user from the dark video detector
)

func (r Rule) IsNoVideoRule() bool {
	return r == NoVideoRule
}

func (r Rule) IsNoImageRule() bool {
	return r == NoImageRule
}

func (r Rule) IsNoBroadcastRule() bool {
	return r == NoVideoRule || r == NoBroadcastRule
}

func (r Rule) IsRedCamRule() bool {
	return r == RedCamRule
}

func (r Rule) IsNoDarkVideoRule() bool {
	return r == NoDarkVideoRule
}

// Rules are the plural set of rules as shown on a JWT token (string array),
// with some extra functionality attached such as an easy serializer to JSON.
type Rules []Rule

// ToDict serializes a Rules string-array into a map of the Is* functions, for easy
// front-end access to the currently enabled rules.
func (r Rules) ToDict() map[string]bool {
	var result = map[string]bool{
		"IsNoVideoRule":     false,
		"IsNoImageRule":     false,
		"IsNoBroadcastRule": false,
		"IsRedCamRule":      false,
		"IsNoDarkVideoRule": false,
	}

	for _, rule := range r {
		if v := rule.IsNoVideoRule(); v {
			result["IsNoVideoRule"] = true
		}
		if v := rule.IsNoImageRule(); v {
			result["IsNoImageRule"] = true
		}
		if v := rule.IsNoBroadcastRule(); v {
			result["IsNoBroadcastRule"] = true
		}
		if v := rule.IsRedCamRule(); v {
			result["IsRedCamRule"] = true
		}
		if v := rule.IsNoDarkVideoRule(); v {
			result["IsNoDarkVideoRule"] = true
		}
	}

	return result
}

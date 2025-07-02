package config

import (
	"regexp"
	"sync"

	"git.kirsle.net/apps/barertc/pkg/log"
)

// MessageFilter configures censored or auto-flagged messages in chat.
type MessageFilter struct {
	Enabled            bool
	PublicChannels     bool
	PrivateChannels    bool
	KeywordPhrases     []string
	CensorMessage      bool
	ForwardMessage     bool
	ReportMessage      bool
	ChatServerResponse string

	// Private use variables.
	isRegexpCompiled bool
	regexps          []*regexp.Regexp
	regexpMu         sync.Mutex
}

// IterPhrases returns the keyword phrases as regular expressions.
func (mf *MessageFilter) IterPhrases() []*regexp.Regexp {
	if mf.isRegexpCompiled {
		return mf.regexps
	}

	// Compile and return the regexps.
	mf.regexpMu.Lock()
	defer mf.regexpMu.Unlock()
	mf.regexps = []*regexp.Regexp{}
	for _, phrase := range mf.KeywordPhrases {
		re, err := regexp.Compile(phrase)
		if err != nil {
			log.Error("MessageFilter: phrase '%s' did not compile as a regexp: %s", phrase, err)
			continue
		}
		mf.regexps = append(mf.regexps, re)
	}

	return mf.regexps
}

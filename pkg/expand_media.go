package barertc

import (
	"fmt"
	"regexp"
)

// Media regexps
var (
	ytLinkRegexp = regexp.MustCompile(`(?:youtube(?:-nocookie)?\.com/(?:[^/]+/.+/|(?:v|e(?:mbed)?)/|.*[?&]v=)|youtu\.be/)([^"&?/\s]{11})`)
	ytIdRegexp   = regexp.MustCompile(`[0-9A-Za-z_-]{10}[048AEIMQUYcgkosw]`) // YT ID validator
)

// YT embed template
const youtubeEmbedTemplate = `<iframe class="youtube-embed" width="560" height="315" src="https://www.youtube.com/embed/%s" title="YouTube video player" frameborder="0" allow="autoplay; encrypted-media; picture-in-picture; web-share" allowfullscreen></iframe>`

// ExpandMedia detects media URLs such as YouTube videos and stylizes the message up with embeds.
func (s *Server) ExpandMedia(message string) string {
	// YouTube links.
	if m := ytLinkRegexp.FindStringSubmatch(message); len(m) > 0 {
		var ytid = m[1]

		// Sanity check the ID parsed OK (e.g. multiple youtube links can throw it off)
		if ytIdRegexp.Match([]byte(ytid)) {
			message += fmt.Sprintf(youtubeEmbedTemplate, ytid)
		}
	}
	return message
}

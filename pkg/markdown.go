package barertc

import (
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/shurcooL/github_flavored_markdown"
)

// Rendermarkdown from untrusted sources.
func RenderMarkdown(input string) string {
	// Render Markdown to HTML.
	html := github_flavored_markdown.Markdown([]byte(input))

	// Sanitize the HTML from any nasties.
	p := bluemonday.UGCPolicy()
	safened := p.SanitizeBytes(html)
	return strings.TrimSpace(string(safened))
}

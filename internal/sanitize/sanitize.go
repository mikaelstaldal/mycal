package sanitize

import "github.com/microcosm-cc/bluemonday"

var policy *bluemonday.Policy

func init() {
	policy = bluemonday.NewPolicy()

	policy.AllowElements(
		"b", "i", "u", "em", "strong",
		"p", "br", "hr",
		"ul", "ol", "li",
		"h1", "h2", "h3", "h4", "h5", "h6",
		"blockquote", "code", "pre",
		"sub", "sup",
		"div", "span",
		"table", "thead", "tbody", "tr", "th", "td",
	)

	policy.AllowAttrs("href").OnElements("a")
	policy.AllowAttrs("title").OnElements("a")
	policy.AllowStandardURLs()
	policy.AllowURLSchemes("http", "https", "mailto")
	policy.RequireParseableURLs(true)

	// Force target="_blank" and rel="noopener noreferrer" on all links
	policy.AddTargetBlankToFullyQualifiedLinks(true)
	policy.RequireNoFollowOnLinks(false)
	policy.RequireNoReferrerOnLinks(true)
}

// HTML sanitizes an HTML string, keeping only allowed tags and attributes.
// Dangerous tags like <script>, <iframe>, <style> and event handler attributes
// are removed. For <a> tags, javascript: URLs are stripped.
func HTML(s string) string {
	return policy.Sanitize(s)
}

package sanitize

import (
	"regexp"
	"strings"
)

// allowedTags is the set of HTML tags that are safe to render.
var allowedTags = map[string]bool{
	"b": true, "i": true, "u": true, "em": true, "strong": true,
	"p": true, "br": true, "hr": true,
	"ul": true, "ol": true, "li": true,
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"blockquote": true, "code": true, "pre": true,
	"sub": true, "sup": true,
	"a": true,
	"div": true, "span": true,
	"table": true, "thead": true, "tbody": true, "tr": true, "th": true, "td": true,
}

// allowedAttrs defines which attributes are allowed per tag.
var allowedAttrs = map[string]map[string]bool{
	"a": {"href": true, "title": true},
}

var (
	tagRe  = regexp.MustCompile(`<(/?)([a-zA-Z][a-zA-Z0-9]*)\b([^>]*)(/?)>`)
	attrRe = regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9-]*)\s*=\s*(?:"([^"]*)"|'([^']*)')`)
)

// HTML sanitizes an HTML string, keeping only allowed tags and attributes.
// Dangerous tags like <script>, <iframe>, <style> and event handler attributes
// are removed. For <a> tags, javascript: URLs are stripped.
func HTML(s string) string {
	return tagRe.ReplaceAllStringFunc(s, func(match string) string {
		parts := tagRe.FindStringSubmatch(match)
		if parts == nil {
			return ""
		}
		closing := parts[1]  // "/" or ""
		tagName := parts[2]  // tag name
		attrStr := parts[3]  // attributes
		selfClose := parts[4] // "/" or ""

		lower := strings.ToLower(tagName)
		if !allowedTags[lower] {
			return ""
		}

		if closing == "/" {
			return "</" + lower + ">"
		}

		// Filter attributes
		allowed := allowedAttrs[lower]
		var attrs []string
		if allowed != nil {
			for _, m := range attrRe.FindAllStringSubmatch(attrStr, -1) {
				attrName := strings.ToLower(m[1])
				attrVal := m[2]
				if attrVal == "" {
					attrVal = m[3]
				}
				if !allowed[attrName] {
					continue
				}
				// Block javascript: URLs
				if attrName == "href" {
					trimmed := strings.TrimSpace(strings.ToLower(attrVal))
					if strings.HasPrefix(trimmed, "javascript:") {
						continue
					}
				}
				attrs = append(attrs, attrName+`="`+attrVal+`"`)
			}
		}

		result := "<" + lower
		if len(attrs) > 0 {
			result += " " + strings.Join(attrs, " ")
		}
		if selfClose == "/" || lower == "br" || lower == "hr" {
			result += " />"
		} else {
			result += ">"
		}
		return result
	})
}

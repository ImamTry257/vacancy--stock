package htmlutil

import (
	"regexp"
	"strings"
)

var (
	reTag        = regexp.MustCompile(`<[^>]+>`)
	reMultiSpace = regexp.MustCompile(`[ \t]+`)
	reMultiLine  = regexp.MustCompile(`\n{3,}`)
	reEntities   = strings.NewReplacer(
		"&amp;", "&",
		"&lt;", "<",
		"&gt;", ">",
		"&quot;", `"`,
		"&#39;", "'",
		"&apos;", "'",
		"&nbsp;", " ",
		"&#160;", " ",
		"&ndash;", "-",
		"&mdash;", "--",
		"&bull;", "*",
		"&middot;", "*",
		"&#8226;", "*",
		"&#8211;", "-",
		"&#8212;", "--",
	)
	// block-level tags that should become newlines
	reBlockTag = regexp.MustCompile(`(?i)<(br\s*/?|/?(p|div|li|ul|ol|h[1-6]|blockquote|pre|tr|td|th|section|article|header|footer|aside|main)(\s[^>]*)?)>`)
)

// StripHTML converts an HTML string to clean plain text.
// Block-level tags become newlines; inline tags are removed.
// HTML entities are decoded. Consecutive blank lines are collapsed.
func StripHTML(html string) string {
	if html == "" {
		return ""
	}

	// replace block tags with newline first
	text := reBlockTag.ReplaceAllString(html, "\n")

	// strip remaining tags
	text = reTag.ReplaceAllString(text, "")

	// decode entities
	text = reEntities.Replace(text)

	// normalise whitespace per line
	lines := strings.Split(text, "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		line = reMultiSpace.ReplaceAllString(line, " ")
		line = strings.TrimSpace(line)
		cleaned = append(cleaned, line)
	}
	text = strings.Join(cleaned, "\n")

	// collapse 3+ consecutive blank lines to 2
	text = reMultiLine.ReplaceAllString(text, "\n\n")

	return strings.TrimSpace(text)
}

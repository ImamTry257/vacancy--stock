package htmlutil

import "testing"

func TestStripHTMLBasic(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty",
			input: "",
			want:  "",
		},
		{
			name:  "plain text passthrough",
			input: "Hello world",
			want:  "Hello world",
		},
		{
			name:  "strip inline tag",
			input: "<strong>Bold</strong> text",
			want:  "Bold text",
		},
		{
			name:  "br becomes newline",
			input: "line one<br>line two",
			want:  "line one\nline two",
		},
		{
			name:  "br self-closing",
			input: "line one<br/>line two",
			want:  "line one\nline two",
		},
		{
			name:  "p tags become newlines",
			// </p> and <p> each become \n, collapsed to max 2 blank lines
			input: "<p>First paragraph</p><p>Second paragraph</p>",
			want:  "First paragraph\n\nSecond paragraph",
		},
		{
			name:  "li items become newlines",
			// <ul>, <li>, </li>, </ul> each become \n; consecutive blanks collapse
			input: "<ul><li>Item one</li><li>Item two</li></ul>",
			want:  "Item one\n\nItem two",
		},
		{
			name:  "decode amp entity",
			input: "A &amp; B",
			want:  "A & B",
		},
		{
			name:  "decode nbsp",
			input: "A&nbsp;B",
			want:  "A B",
		},
		{
			name:  "collapse multiple blank lines",
			input: "A\n\n\n\nB",
			want:  "A\n\nB",
		},
		{
			name:  "trim leading and trailing whitespace",
			input: "  <p>  Hello  </p>  ",
			want:  "Hello",
		},
		{
			name:  "real kalibrr snippet",
			input: "<ul>\n\t<li>Build backend services</li>\n\t<li>Write unit tests</li>\n</ul>\n\n<ul>\n\t<li>Min. 3 years experience</li>\n\t<li>Proficient in Go &amp; MySQL</li>\n</ul>",
			// each <li> and </li> become \n; blank lines between groups collapse to max 2
			want: "Build backend services\n\nWrite unit tests\n\nMin. 3 years experience\n\nProficient in Go & MySQL",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := StripHTML(tc.input)
			if got != tc.want {
				t.Errorf("StripHTML(%q)\n  got:  %q\n  want: %q", tc.input, got, tc.want)
			}
		})
	}
}

package sanitize

import "testing"

func TestHTML(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "plain text unchanged",
			in:   "hello world",
			want: "hello world",
		},
		{
			name: "allowed tags preserved",
			in:   "<b>bold</b> and <em>italic</em>",
			want: "<b>bold</b> and <em>italic</em>",
		},
		{
			name: "script tag removed",
			in:   `<script>alert("xss")</script>`,
			want: `alert("xss")`,
		},
		{
			name: "iframe removed",
			in:   `<iframe src="evil.com"></iframe>`,
			want: ``,
		},
		{
			name: "style tag removed",
			in:   `<style>body{display:none}</style>`,
			want: `body{display:none}`,
		},
		{
			name: "event handler attributes stripped",
			in:   `<b onclick="alert(1)">text</b>`,
			want: `<b>text</b>`,
		},
		{
			name: "safe link preserved",
			in:   `<a href="https://example.com" title="Example">link</a>`,
			want: `<a href="https://example.com" title="Example">link</a>`,
		},
		{
			name: "javascript href removed",
			in:   `<a href="javascript:alert(1)">link</a>`,
			want: `<a>link</a>`,
		},
		{
			name: "mixed content",
			in:   `<p>Hello</p><script>evil()</script><ul><li>item</li></ul>`,
			want: `<p>Hello</p>evil()<ul><li>item</li></ul>`,
		},
		{
			name: "br self-closing",
			in:   `line one<br>line two<br/>line three`,
			want: `line one<br />line two<br />line three`,
		},
		{
			name: "img tag removed",
			in:   `<img src="x" onerror="alert(1)">`,
			want: ``,
		},
		{
			name: "object tag removed",
			in:   `<object data="evil.swf"></object>`,
			want: ``,
		},
		{
			name: "nested formatting",
			in:   `<p><strong>bold <em>and italic</em></strong></p>`,
			want: `<p><strong>bold <em>and italic</em></strong></p>`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HTML(tt.in)
			if got != tt.want {
				t.Errorf("HTML(%q)\n got  %q\n want %q", tt.in, got, tt.want)
			}
		})
	}
}

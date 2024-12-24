// Copyright (c) 2024 Chris Watson
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package styling

import "testing"

func TestParseMarkdownV2(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Style
	}{
		{
			name:  "escaped characters in styled text",
			input: "*bold \\*text*",
			expected: []Style{
				{Type: "bold", Text: "bold *text"},
			},
		},
		{
			name:  "escaped characters in italic",
			input: "_italic \\*text_",
			expected: []Style{
				{Type: "italic", Text: "italic *text"},
			},
		},
		{
			name:  "underline",
			input: "__underline__",
			expected: []Style{
				{Type: "underline", Text: "underline"},
			},
		},
		{
			name:  "double pipe spoiler",
			input: "||spoiler||",
			expected: []Style{
				{Type: "spoiler", Text: "spoiler"},
			},
		},
		{
			name:  "complex nested styles",
			input: "*bold _italic bold ~italic bold strikethrough ||italic bold strikethrough spoiler||~ __underline italic bold___ bold*",
			expected: []Style{
				{Type: "bold", Text: "bold italic bold italic bold strikethrough italic bold strikethrough spoiler underline italic bold bold"},
			},
		},
		{
			name:  "user mention link",
			input: "[inline mention of a user](tg://user?id=123456789)",
			expected: []Style{
				{Type: "text_url", Text: "inline mention of a user", URL: "tg://user?id=123456789"},
			},
		},
		{
			name:  "emoji link",
			input: "![👍](tg://emoji?id=5368324170671202286)",
			expected: []Style{
				{Type: "text_url", Text: "👍", URL: "tg://emoji?id=5368324170671202286"},
			},
		},
		{
			name:  "code block with language",
			input: "```python\npre-formatted fixed-width code block written in the Python programming language\n```",
			expected: []Style{
				{Type: "pre", Text: "pre-formatted fixed-width code block written in the Python programming language"},
			},
		},
		{
			name:  "expandable block quote",
			input: "**>The expandable block quotation started\n>Hidden by default part\n>The last line||",
			expected: []Style{
				{Type: "blockquote", Text: "The expandable block quotation started\nHidden by default part\nThe last line", Collapsed: true},
			},
		},
		{
			name:  "plain text",
			input: "Hello world",
			expected: []Style{
				{Type: "plain", Text: "Hello world"},
			},
		},
		{
			name:  "bold text",
			input: "*bold*",
			expected: []Style{
				{Type: "bold", Text: "bold"},
			},
		},
		{
			name:  "italic text",
			input: "_italic_",
			expected: []Style{
				{Type: "italic", Text: "italic"},
			},
		},
		{
			name:  "strike text",
			input: "~strike~",
			expected: []Style{
				{Type: "strike", Text: "strike"},
			},
		},
		{
			name:  "code text",
			input: "`code`",
			expected: []Style{
				{Type: "code", Text: "code"},
			},
		},
		{
			name:  "link",
			input: "[text](https://example.com)",
			expected: []Style{
				{Type: "text_url", Text: "text", URL: "https://example.com"},
			},
		},
		{
			name:  "blockquote",
			input: ">quoted text\nother text",
			expected: []Style{
				{Type: "blockquote", Text: "quoted text", Collapsed: false},
				{Type: "plain", Text: "other text"},
			},
		},
		{
			name:  "spoiler",
			input: "||spoiler||",
			expected: []Style{
				{Type: "spoiler", Text: "spoiler"},
			},
		},
		{
			name:  "escaped characters",
			input: "\\*not bold\\* \\[not link\\]",
			expected: []Style{
				{Type: "plain", Text: "*not bold* [not link]"},
			},
		},
		{
			name:  "mixed styling",
			input: "Hello *bold* and _italic_ and `code`",
			expected: []Style{
				{Type: "plain", Text: "Hello "},
				{Type: "bold", Text: "bold"},
				{Type: "plain", Text: " and "},
				{Type: "italic", Text: "italic"},
				{Type: "plain", Text: " and "},
				{Type: "code", Text: "code"},
			},
		},
		{
			name:  "unclosed styles",
			input: "*bold text without end",
			expected: []Style{
				{Type: "plain", Text: "*bold text without end"},
			},
		},
		{
			name:  "nested styles are treated as plain text",
			input: "*bold _italic_*",
			expected: []Style{
				{Type: "bold", Text: "bold _italic_"},
			},
		},
		{
			name:  "invalid link format",
			input: "[text](invalid url",
			expected: []Style{
				{Type: "plain", Text: "[text](invalid url"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder()
			for _, style := range tt.expected {
				builder.Append(style)
			}
			got := ParseMarkdownV2(tt.input)
			gotStyles := builder.styles

			if len(got) != len(tt.expected) {
				t.Errorf("ParseMarkdownV2() returned %d styles, want %d", len(got), len(tt.expected))
				return
			}

			for i, style := range tt.expected {
				if i >= len(gotStyles) {
					t.Errorf("Missing expected style at index %d: %+v", i, style)
					continue
				}
				if gotStyles[i].Type != style.Type {
					t.Errorf("Style[%d].Type = %q, want %q", i, gotStyles[i].Type, style.Type)
				}
				if gotStyles[i].Text != style.Text {
					t.Errorf("Style[%d].Text = %q, want %q", i, gotStyles[i].Text, style.Text)
				}
				if gotStyles[i].URL != style.URL {
					t.Errorf("Style[%d].URL = %q, want %q", i, gotStyles[i].URL, style.URL)
				}
				if gotStyles[i].Collapsed != style.Collapsed {
					t.Errorf("Style[%d].Collapsed = %v, want %v", i, gotStyles[i].Collapsed, style.Collapsed)
				}
			}
		})
	}
}

func TestParseMarkdownV2_RealWorldExamples(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Style
	}{
		{
			name: "full markdown example",
			input: "*bold \\*text*\n" +
				"_italic \\*text_\n" +
				"__underline__\n" +
				"~~strikethrough~~\n" +
				"||spoiler||\n" +
				"*bold _italic bold ~italic bold strikethrough ||italic bold strikethrough spoiler||~ __underline italic bold___ bold*\n" +
				"[inline URL](http://www.example.com/)\n" +
				"[inline mention of a user](tg://user?id=123456789)\n" +
				"![👍](tg://emoji?id=5368324170671202286)\n" +
				"`inline fixed-width code`\n" +
				"```\npre-formatted fixed-width code block\n```\n" +
				"```python\npre-formatted fixed-width code block written in the Python programming language\n```\n" +
				">Block quotation started\n" +
				">Block quotation continued\n" +
				">Block quotation continued\n" +
				">Block quotation continued\n" +
				">The last line of the block quotation\n" +
				"**>The expandable block quotation started right after the previous block quotation\n" +
				">It is separated from the previous block quotation by an empty bold entity\n" +
				">Expandable block quotation continued\n" +
				">Hidden by default part of the expandable block quotation started\n" +
				">Expandable block quotation continued\n" +
				">The last line of the expandable block quotation with the expandability mark||",
			expected: []Style{
				{Type: "bold", Text: "bold *text"},
				{Type: "plain", Text: "\n"},
				{Type: "italic", Text: "italic *text"},
				{Type: "plain", Text: "\n"},
				{Type: "underline", Text: "underline"},
				{Type: "plain", Text: "\n"},
				{Type: "strike", Text: "strikethrough"},
				{Type: "plain", Text: "\n"},
				{Type: "spoiler", Text: "spoiler"},
				{Type: "plain", Text: "\n"},
				{Type: "bold", Text: "bold italic bold italic bold strikethrough italic bold strikethrough spoiler underline italic bold bold"},
				{Type: "plain", Text: "\n"},
				{Type: "text_url", Text: "inline URL", URL: "http://www.example.com/"},
				{Type: "plain", Text: "\n"},
				{Type: "text_url", Text: "inline mention of a user", URL: "tg://user?id=123456789"},
				{Type: "plain", Text: "\n"},
				{Type: "text_url", Text: "👍", URL: "tg://emoji?id=5368324170671202286"},
				{Type: "plain", Text: "\n"},
				{Type: "code", Text: "inline fixed-width code"},
				{Type: "plain", Text: "\n"},
				{Type: "pre", Text: "pre-formatted fixed-width code block"},
				{Type: "plain", Text: "\n"},
				{Type: "pre", Text: "pre-formatted fixed-width code block written in the Python programming language"},
				{Type: "plain", Text: "\n"},
				{Type: "blockquote", Text: "Block quotation started\nBlock quotation continued\nBlock quotation continued\nBlock quotation continued\nThe last line of the block quotation", Collapsed: false},
				{Type: "plain", Text: "\n"},
				{Type: "blockquote", Text: "The expandable block quotation started right after the previous block quotation\nIt is separated from the previous block quotation by an empty bold entity\nExpandable block quotation continued\nHidden by default part of the expandable block quotation started\nExpandable block quotation continued\nThe last line of the expandable block quotation with the expandability mark", Collapsed: true},
			},
		},
		// {
		// 	name: "complex message",
		// 	input: "Hello! Here's a *bold announcement*:\n" +
		// 		">Important message for everyone\n" +
		// 		"Check out our website [here](https://example.com)\n" +
		// 		"Use code `SAVE20` for a _20%_ discount!\n" +
		// 		"||Secret: launch date is tomorrow!||",
		// 	expected: []Style{
		// 		{Type: "plain", Text: "Hello! Here's a "},
		// 		{Type: "bold", Text: "bold announcement"},
		// 		{Type: "plain", Text: ":\n"},
		// 		{Type: "blockquote", Text: "Important message for everyone", Collapsed: false},
		// 		{Type: "plain", Text: "Check out our website "},
		// 		{Type: "text_url", Text: "here", URL: "https://example.com"},
		// 		{Type: "plain", Text: "\nUse code "},
		// 		{Type: "code", Text: "SAVE20"},
		// 		{Type: "plain", Text: " for a "},
		// 		{Type: "italic", Text: "20%"},
		// 		{Type: "plain", Text: " discount!\n"},
		// 		{Type: "spoiler", Text: "Secret: launch date is tomorrow!"},
		// 	},
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder()
			for _, style := range tt.expected {
				builder.Append(style)
			}
			got := ParseMarkdownV2(tt.input)
			gotStyles := builder.styles

			if len(got) != len(tt.expected) {
				t.Errorf("ParseMarkdownV2() returned %d styles, want %d", len(got), len(tt.expected))
				return
			}

			for i, style := range tt.expected {
				if i >= len(gotStyles) {
					t.Errorf("Missing expected style at index %d: %+v", i, style)
					continue
				}
				if gotStyles[i].Type != style.Type {
					t.Errorf("Style[%d].Type = %q, want %q", i, gotStyles[i].Type, style.Type)
				}
				if gotStyles[i].Text != style.Text {
					t.Errorf("Style[%d].Text = %q, want %q", i, gotStyles[i].Text, style.Text)
				}
				if gotStyles[i].URL != style.URL {
					t.Errorf("Style[%d].URL = %q, want %q", i, gotStyles[i].URL, style.URL)
				}
				if gotStyles[i].Collapsed != style.Collapsed {
					t.Errorf("Style[%d].Collapsed = %v, want %v", i, gotStyles[i].Collapsed, style.Collapsed)
				}
			}
		})
	}
}

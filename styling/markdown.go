// Copyright (c) 2024 Chris Watson
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT
package styling

import (
	"io"
	"strconv"
	"strings"

	"github.com/gotd/td/telegram/message/styling"
)

// ParseMarkdownV2 parses a string of Markdown text into a list of styled text options
// compatible with gotd's message styling system.
// Example input:
//
//	*bold \*text*
//	_italic \*text_
//	__underline__
//	~strikethrough~
//	||spoiler||
//	*bold _italic bold ~italic bold strikethrough ||italic bold strikethrough spoiler||~ __underline italic bold___ bold*
//	[inline URL](http://www.example.com/)
//	[inline mention of a user](tg://user?id=123456789)
//	![👍](tg://emoji?id=5368324170671202286)
//	`inline fixed-width code`
//	```
//	pre-formatted fixed-width code block
//	```
//	```python
//	pre-formatted fixed-width code block written in the Python programming language
//	```
//	>Block quotation started
//	>Block quotation continued
//	>Block quotation continued
//	>Block quotation continued
//	>The last line of the block quotation
//	**>The expandable block quotation started right after the previous block quotation
//	>It is separated from the previous block quotation by an empty bold entity
//	>Expandable block quotation continued
//	>Hidden by default part of the expandable block quotation started
//	>Expandable block quotation continued
//	>The last line of the expandable block quotation with the expandability mark||
func ParseMarkdownV2(text string) []styling.StyledTextOption {
	builder := NewBuilder()
	reader := NewReader(text)

	for {
		c, _, err := reader.ReadRune()
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}

		escaped := false

		if escaped {
			builder.Text(string(c))
			continue
		}

		switch c {
		case '\\':
			escaped = true
			continue
		case '*', '_', '~', '|':
			var styleType string
			var delim string
			nextByte, err := reader.PeekByte()
			if err != nil {
				break // TODO: do this correctly
			}
			switch {
			case c == '|' && nextByte == '|':
				reader.ReadByte()
				styleType = "spoiler"
				delim = "||"
			case c == '~' && nextByte == '~':
				reader.ReadByte()
				styleType = "strike"
				delim = "~~"
			case c == '_':
				if nextByte == '_' {
					reader.ReadByte()
					styleType = "underline"
					delim = "__"
				} else {
					styleType = "italic"
					delim = "_"
				}
			case c == '*':
				styleType = "bold"
				delim = "*"
			}

			if styleType != "" {
				contents, err := reader.ReadUntil(delim)
				if err != nil {
					break // TODO: do this correctly
				}

				_, err = reader.Skip(len(delim))
				if err != nil {
					break // TODO: do this correctly
				}

				if styleType == "bold" {
					builder.Bold(contents)
				} else if styleType == "italic" {
					builder.Italic(contents)
				} else if styleType == "strike" {
					builder.Strike(contents)
				} else if styleType == "code" {
					builder.Code(contents)
				} else if styleType == "spoiler" {
					builder.Spoiler(contents)
				} else if styleType == "underline" {
					builder.Underline(contents)
				}
			} else {
				builder.Text(string(c))
			}
		case '!':
			if nextByte, err := reader.PeekByte(); err == nil {
				if nextByte == '[' {
					// Potential image/emoji link
					reader.ReadByte()
					contents, err := reader.ReadUntil("]")
					if err != nil {
						break // TODO: do this correctly
					}
					_, err = reader.Skip(1)
					if err != nil {
						break // TODO: do this correctly
					}
					if strings.HasPrefix(contents, "tg://emoji?id=") {
						// Parse out the emoji ID
						idStr := strings.TrimPrefix(contents, "tg://emoji?id=")
						if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
							builder.CustomEmoji(contents, id)
							// Skip any following newline
							if nextByte, err := reader.PeekByte(); err == nil && nextByte == '\n' {
								reader.ReadByte()
							}
							continue
						}
					}
				}
			}
		}
	}
}

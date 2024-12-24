// Copyright (c) 2024 Chris Watson
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package styling

import (
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"
)

type Style struct {
	Type       string
	Text       string
	URL        string
	Collapsed  bool
	DocumentID int64
}

type Builder struct {
	styles []Style
}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) Append(styles ...Style) {
	b.styles = append(b.styles, styles...)
}

func (b *Builder) Reset() {
	b.styles = b.styles[:0]
}

func (b *Builder) Len() int {
	return len(b.styles)
}

func (b *Builder) IsEmpty() bool {
	return b.Len() == 0
}

func (b *Builder) Styles() []Style {
	return b.styles
}

func (b *Builder) Remove(i int) {
	b.styles = append(b.styles[:i], b.styles[i+1:]...)
}

func (b *Builder) RemoveLast() {
	b.Remove(b.Len() - 1)
}

func (b *Builder) Insert(i int, styles ...Style) {
	b.styles = append(b.styles[:i], append(styles, b.styles[i:]...)...)
}

func (b *Builder) Get(i int) Style {
	return b.styles[i]
}

func (b *Builder) Build() []styling.StyledTextOption {
	var result []styling.StyledTextOption
	for _, style := range b.styles {
		switch style.Type {
		case "plain":
			result = append(result, styling.Plain(style.Text))
		case "bold":
			result = append(result, styling.Bold(style.Text))
		case "italic":
			result = append(result, styling.Italic(style.Text))
		case "code":
			result = append(result, styling.Code(style.Text))
		case "pre":
			result = append(result, styling.Pre(style.Text, ""))
		case "text_url":
			result = append(result, styling.TextURL(style.Text, style.URL))
		case "mention":
			result = append(result, styling.Mention(style.Text))
		case "hashtag":
			result = append(result, styling.Hashtag(style.Text))
		case "bot_command":
			result = append(result, styling.BotCommand(style.Text))
		case "email":
			result = append(result, styling.Email(style.Text))
		case "cashtag":
			result = append(result, styling.Cashtag(style.Text))
		case "underline":
			result = append(result, styling.Underline(style.Text))
		case "strike":
			result = append(result, styling.Strike(style.Text))
		case "bank_card":
			result = append(result, styling.BankCard(style.Text))
		case "spoiler":
			result = append(result, styling.Spoiler(style.Text))
		case "custom_emoji":
			result = append(result, styling.CustomEmoji(style.Text, style.DocumentID))
		case "blockquote":
			result = append(result, styling.Blockquote(style.Text, style.Collapsed))
		}
	}
	return result
}

func (b *Builder) Text(text string) {
	b.Append(Style{Type: "plain", Text: text})
}

func (b *Builder) Mention(name string) {
	b.Append(Style{Type: "mention", Text: name})
}

func (b *Builder) Hashtag(name string) {
	b.Append(Style{Type: "hashtag", Text: name})
}

func (b *Builder) BotCommand(name string) {
	b.Append(Style{Type: "bot_command", Text: name})
}

func (b *Builder) Url(url string) {
	b.Append(Style{Type: "url", Text: url})
}

func (b *Builder) Email(email string) {
	b.Append(Style{Type: "email", Text: email})
}

func (b *Builder) Bold(text string) {
	b.Append(Style{Type: "bold", Text: text})
}

func (b *Builder) Italic(text string) {
	b.Append(Style{Type: "italic", Text: text})
}

func (b *Builder) Code(text string) {
	b.Append(Style{Type: "code", Text: text})
}

func (b *Builder) Pre(text string, language string) {
	b.Append(Style{Type: "pre", Text: text})
}

func (b *Builder) TextUrl(text string, url string) {
	b.Append(Style{Type: "text_url", Text: text, URL: url})
}

func (b *Builder) MentionName(name string, userId tg.InputUserClass) {
	b.Append(Style{Type: "mention_name", Text: name})
}

func (b *Builder) Cashtag(name string) {
	b.Append(Style{Type: "cashtag", Text: name})
}

func (b *Builder) Underline(text string) {
	b.Append(Style{Type: "underline", Text: text})
}

func (b *Builder) Strike(text string) {
	b.Append(Style{Type: "strike", Text: text})
}

func (b *Builder) BankCard(text string) {
	b.Append(Style{Type: "bank_card", Text: text})
}

func (b *Builder) Spoiler(text string) {
	b.Append(Style{Type: "spoiler", Text: text})
}

func (b *Builder) CustomEmoji(text string, documentId int64) {
	b.Append(Style{Type: "custom_emoji", Text: text, DocumentID: documentId})
}

func (b *Builder) Blockquote(text string, collapsed bool) {
	b.Append(Style{Type: "blockquote", Text: text, Collapsed: collapsed})
}

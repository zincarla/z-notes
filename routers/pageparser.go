package routers

import (
	"bytes"
	"html/template"

	embed "github.com/zincarla/goldmark-embed"

	mathjax "github.com/litao91/goldmark-mathjax"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	highlighting "github.com/yuin/goldmark-highlighting"
	"github.com/yuin/goldmark/extension"
)

var markdownParser goldmark.Markdown

//InitMarkdown initializes a markdown parser
func InitMarkdown() {
	markdownParser = goldmark.New(
		goldmark.WithExtensions(
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
			extension.DefinitionList,
			extension.Footnote,
			highlighting.Highlighting,
			emoji.Emoji,
			mathjax.MathJax,
			embed.DefaultEmbed,
		),
	)
}

//GetParsedPage converts the markdown into a template.HTML
func GetParsedPage(markdownContent string) (template.HTML, error) {
	var buf bytes.Buffer
	if err := markdownParser.Convert([]byte(markdownContent), &buf); err != nil {
		return template.HTML(""), err
	}
	return template.HTML(buf.String()), nil
}

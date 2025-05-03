// Package doc formats texts using the go doc format
// as a more limited alternative to Markdown.
package doc

import "go/doc/comment"
import "html/template"

func HTML(in string) template.HTML {
	parser := comment.Parser{}
	doc := parser.Parse(in)
	pr := comment.Printer{}
	return template.HTML(string(pr.HTML(doc)))
}

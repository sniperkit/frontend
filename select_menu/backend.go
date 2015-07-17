// +build !js

package select_menu

import (
	"fmt"
	"html/template"
	"net/url"
	"strconv"

	"github.com/shurcooL/htmlg"
	"golang.org/x/net/html"
)

// New creates the HTML for a select menu instance with the specified parameters.
func New(options []string, defaultOption string, query url.Values, queryParameter string) template.HTML {
	selectElement := &html.Node{Type: html.ElementNode, Data: "select"}

	var selectedOption = defaultOption
	if query.Get(queryParameter) != "" {
		selectedOption = query.Get(queryParameter)
	}
	if !contains(options, selectedOption) {
		options = append(options, selectedOption)
	}
	for _, option := range options {
		o := &html.Node{Type: html.ElementNode, Data: "option"}
		o.AppendChild(htmlg.Text(option))
		if option == selectedOption {
			o.Attr = append(o.Attr, html.Attribute{Key: "selected"})
		}
		selectElement.AppendChild(o)
	}

	selectElement.Attr = append(selectElement.Attr, html.Attribute{
		Key: "oninput",
		// HACK: Don't use Sprintf, properly encode (as json at this time).
		Val: fmt.Sprintf(`SelectMenuOnInput(event, this, %q, %q);`, strconv.Quote(defaultOption), strconv.Quote(queryParameter)),
	})

	html, err := htmlg.RenderNodes(selectElement)
	if err != nil {
		panic(err)
	}
	return html
}

func contains(ss []string, t string) bool {
	for _, s := range ss {
		if s == t {
			return true
		}
	}
	return false
}

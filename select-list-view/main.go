// +build js

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/gopherjs/gopherjs/js"
	"github.com/shurcooL/go/html_gen"

	"golang.org/x/net/html"

	"honnef.co/go/js/dom"
)

var document = dom.GetWindow().Document().(dom.HTMLDocument)

var headers []dom.Element

type filterableElement struct {
	Id               string
	TextContent      string
	LowerTextContent string
}

var headers2 []filterableElement

var selected int

var baseHash string
var baseX, baseY int

var entryHeight float64
var entries []dom.Node
var manuallyPicked string

func main() {
	overlay := document.CreateElement("div").(*dom.HTMLDivElement)
	overlay.SetID("gts-overlay")

	container := document.CreateElement("div")
	overlay.AppendChild(container)
	container.Underlying().Set("outerHTML", `<div><input id="gts-command"></input><div id="gts-results"></div></div>`)

	document.Body().AppendChild(overlay)

	command := document.GetElementByID("gts-command").(*dom.HTMLInputElement)
	results := document.GetElementByID("gts-results").(*dom.HTMLDivElement)

	var timer int
	//var ch = make(chan struct{})

	command.AddEventListener("input", false, func(event dom.Event) {
		//updateResults(false, nil)

		dom.GetWindow().ClearTimeout(timer)
		timer = dom.GetWindow().SetTimeout(func() { updateResults(false, nil) }, 200)

		/*select {
		case ch <- struct{}{}:
		default:
		}*/
	})

	/*go func() {
		for {
			<-ch

			updateResults(false, nil)

			time.Sleep(200 * time.Millisecond)
		}
	}()*/

	/*mousedown := false
	results.AddEventListener("mousedown", false, func(event dom.Event) {
		mousedown = true

		command.Focus()

		me := event.(*dom.MouseEvent)
		y := (me.ClientY - results.GetBoundingClientRect().Top) + results.Underlying().Get("scrollTop").Int()
		selected = int(float64(y) / entryHeight)
		updateResultSelection()
	})
	results.AddEventListener("mouseup", false, func(event dom.Event) {
		mousedown = false
	})
	results.AddEventListener("mouseleave", false, func(event dom.Event) {
		mousedown = false
	})
	results.AddEventListener("mousemove", false, func(event dom.Event) {
		if !mousedown {
			return
		}

		command.Focus()

		me := event.(*dom.MouseEvent)
		y := (me.ClientY - results.GetBoundingClientRect().Top) + results.Underlying().Get("scrollTop").Int()
		selected = int(float64(y) / entryHeight)
		updateResultSelection()
	})*/
	results.AddEventListener("click", false, func(event dom.Event) {
		command.Focus()

		me := event.(*dom.MouseEvent)
		y := (me.ClientY - results.GetBoundingClientRect().Top) + results.Underlying().Get("scrollTop").Int()
		selected = int(float64(y) / entryHeight)
		updateResultSelection()
	})
	results.AddEventListener("dblclick", false, func(event dom.Event) {
		event.PreventDefault()

		hideOverlay(overlay)
	})

	overlay.AddEventListener("keydown", false, func(event dom.Event) {
		switch ke := event.(*dom.KeyboardEvent); {
		case ke.KeyIdentifier == "U+001B": // Escape.
			ke.PreventDefault()

			hideOverlay(overlay)

			if document.ActiveElement().Underlying() == command.Underlying() {
				//dom.GetWindow().Location().Hash = baseHash
				js.Global.Get("window").Get("history").Call("replaceState", nil, nil, "#"+baseHash)
				dom.GetWindow().ScrollTo(baseX, baseY)
			}
		case ke.KeyIdentifier == "Enter":
			ke.PreventDefault()

			hideOverlay(overlay)
		case ke.KeyIdentifier == "Down":
			ke.PreventDefault()

			switch {
			case !ke.CtrlKey && !ke.AltKey && ke.MetaKey:
				selected = len(entries) - 1
			case ke.CtrlKey && ke.AltKey && !ke.MetaKey:
				results.Underlying().Set("scrollTop", results.Underlying().Get("scrollTop").Float()+entryHeight)
				return
			case !ke.CtrlKey && !ke.AltKey && !ke.MetaKey:
				selected++
			}
			updateResultSelection()
		case ke.KeyIdentifier == "Up":
			ke.PreventDefault()

			switch {
			case !ke.CtrlKey && !ke.AltKey && ke.MetaKey:
				selected = 0
			case ke.CtrlKey && ke.AltKey && !ke.MetaKey:
				results.Underlying().Set("scrollTop", results.Underlying().Get("scrollTop").Float()-entryHeight)
				return
			case !ke.CtrlKey && !ke.AltKey && !ke.MetaKey:
				selected--
			}
			updateResultSelection()
		}
	})

	document.Body().AddEventListener("keydown", false, func(event dom.Event) {
		switch ke := event.(*dom.KeyboardEvent); {
		case ke.KeyIdentifier == "U+0052": // Cmd+R (or just R, since some browsers don't let us intercept Cmd+R).
			// Ignore R when command elment has focus (it means the user is typing).
			if document.ActiveElement().Underlying() == command.Underlying() {
				break
			}

			ke.PreventDefault()

			if display := overlay.Style().GetPropertyValue("display"); display != "none" && display != "null" {
				command.Select()
				break
			}

			command.Value = ""
			manuallyPicked = ""

			{
				headers = nil
				for _, header := range append(document.Body().GetElementsByTagName("h3"), document.Body().GetElementsByTagName("h4")...) {
					if header.ID() == "" {
						continue
					}
					headers = append(headers, header)
				}

				headers2 = nil
				for _, header := range headers {
					headers2 = append(headers2, filterableElement{
						Id:               header.ID(),
						TextContent:      header.TextContent(),
						LowerTextContent: strings.ToLower(header.TextContent()),
					})
				}

				baseHash = strings.TrimPrefix(dom.GetWindow().Location().Hash, "#")
				baseX, baseY = dom.GetWindow().ScrollX(), dom.GetWindow().ScrollY()

				updateResults(true, overlay)
			}

			command.Select()
		case ke.KeyIdentifier == "U+001B": // Escape.
			ke.PreventDefault()

			hideOverlay(overlay)
		}
	})
}

var previouslyHighlightedHeader dom.HTMLElement

func hideOverlay(overlay dom.HTMLElement) {
	overlay.Style().SetProperty("display", "none", "")

	if previouslyHighlightedHeader != nil {
		previouslyHighlightedHeader.Class().Remove("highlighted")
		previouslyHighlightedHeader.Class().Add("highlighted-fade")
	}
}

var previouslySelected int

func updateResultSelection() {
	results := document.GetElementByID("gts-results").(*dom.HTMLDivElement)

	if selected < 0 {
		selected = 0
	} else if selected > len(entries)-1 {
		selected = len(entries) - 1
	}

	if selected == previouslySelected {
		return
	}

	entries[previouslySelected].(dom.Element).Class().Remove("gts-highlighted")
	if previouslyHighlightedHeader != nil {
		previouslyHighlightedHeader.Class().Remove("highlighted")
	}

	{
		element := entries[selected].(dom.Element)

		if element.GetBoundingClientRect().Top <= results.GetBoundingClientRect().Top {
			element.Underlying().Call("scrollIntoView", true)
		} else if element.GetBoundingClientRect().Bottom >= results.GetBoundingClientRect().Bottom {
			element.Underlying().Call("scrollIntoView", false)
		}

		element.Class().Add("gts-highlighted")
		//dom.GetWindow().Location().Hash = "#" + element.GetAttribute("data-id")
		//dom.GetWindow().History().ReplaceState(nil, nil, "#"+element.GetAttribute("data-id"))
		js.Global.Get("window").Get("history").Call("replaceState", nil, nil, "#"+element.GetAttribute("data-id"))
		target := document.GetElementByID(element.GetAttribute("data-id")).(dom.HTMLElement)
		target.Class().Add("highlighted")
		previouslyHighlightedHeader = target
		centerOnTargetIfOffscreen(target)

		manuallyPicked = element.GetAttribute("data-id")
	}

	previouslySelected = selected
}

func centerOnTargetIfOffscreen(target dom.HTMLElement) {
	isOffscreen := int(target.OffsetTop()) < dom.GetWindow().ScrollY() ||
		int(target.OffsetTop()+target.OffsetHeight()) > dom.GetWindow().ScrollY()+dom.GetWindow().InnerHeight()

	if isOffscreen {
		windowHalfHeight := dom.GetWindow().InnerHeight() / 2

		dom.GetWindow().ScrollTo(dom.GetWindow().ScrollX(), int(target.OffsetTop()+target.OffsetHeight())-windowHalfHeight)
	}
}

var initialSelected int

func updateResults(init bool, overlay dom.HTMLElement) {
	started := time.Now()

	windowHalfHeight := dom.GetWindow().InnerHeight() / 2
	filter := document.GetElementByID("gts-command").(*dom.HTMLInputElement).Value
	lowerFilter := strings.ToLower(filter)
	//js.Global.Set("my-lower-filter", lowerFilter)

	results := document.GetElementByID("gts-results").(*dom.HTMLDivElement)

	var selectionPreserved = false

	//results.SetInnerHTML("")
	var ns []*html.Node
	var visibleIndex int
	/*for _, header := range headers {
		/*if filter != "" && !strings.Contains(strings.ToLower(header.TextContent()), lowerFilter) {
			continue
		}* /
		if filter != "" && header.Underlying().Get("textContent").Call("toLowerCase").Call("indexOf", lowerFilter).Int() == -1 {
			continue
		}
		/*if filter != "" && header.Underlying().Get("textContent").Call("toLowerCase").Call("indexOf", js.Global.Get("my-lower-filter")).Int() == -1 {
			continue
		}* /

		/*element := document.CreateElement("div")
		element.Class().Add("gts-entry")
		element.SetAttribute("data-id", header.ID())
		{
			entry := header.TextContent()
			index := strings.Index(strings.ToLower(entry), lowerFilter)
			element.SetInnerHTML(html.EscapeString(entry[:index]) + "<strong>" + html.EscapeString(entry[index:index+len(filter)]) + "</strong>" + html.EscapeString(entry[index+len(filter):]))
		}
		results.AppendChild(element)* /
		/*entry := header.TextContent()
		index := strings.Index(strings.ToLower(entry), lowerFilter)
		p1 := html_gen.Text(entry[:index])
		p2 := Strong(entry[index : index+len(filter)]) // This can be optimized out of loop?
		p3 := html_gen.Text(entry[index+len(filter):])
		n := CustomDiv(p1, p2, p3, "gts-entry", header.ID())
		ns = append(ns, n)* /

		if header.ID() == manuallyPicked {
			selectionPreserved = true

			selected = visibleIndex
			previouslySelected = visibleIndex
		}

		visibleIndex++

		/*if visibleIndex >= 1000 {
			break
		}* /
	}*/
	for _, header := range headers2 {
		if filter != "" && !strings.Contains(header.LowerTextContent, lowerFilter) {
			continue
		}
		/*if filter != "" && header.Underlying().Get("textContent").Call("toLowerCase").Call("indexOf", lowerFilter).Int() == -1 {
			continue
		}*/
		/*if filter != "" && header.Underlying().Get("textContent").Call("toLowerCase").Call("indexOf", js.Global.Get("my-lower-filter")).Int() == -1 {
			continue
		}*/

		/*element := document.CreateElement("div")
		element.Class().Add("gts-entry")
		element.SetAttribute("data-id", header.ID())
		{
			entry := header.TextContent()
			index := strings.Index(strings.ToLower(entry), lowerFilter)
			element.SetInnerHTML(html.EscapeString(entry[:index]) + "<strong>" + html.EscapeString(entry[index:index+len(filter)]) + "</strong>" + html.EscapeString(entry[index+len(filter):]))
		}
		results.AppendChild(element)*/
		entry := header.TextContent
		index := strings.Index(header.LowerTextContent, lowerFilter)
		p1 := html_gen.Text(entry[:index])
		p2 := Strong(entry[index : index+len(filter)]) // This can be optimized out of loop?
		p3 := html_gen.Text(entry[index+len(filter):])
		n := CustomDiv(p1, p2, p3, "gts-entry", header.Id)
		ns = append(ns, n)

		if header.Id == manuallyPicked {
			selectionPreserved = true

			selected = visibleIndex
			previouslySelected = visibleIndex
		}

		visibleIndex++

		if visibleIndex >= 1000 {
			break
		}
	}
	//results.SetInnerHTML(`<div class="gts-entry" data-id="bufio">stuff goes there</div><div class="gts-entry" data-id="strings">more stuff</div>`)
	innerHtml, err := html_gen.RenderNodes(ns...)
	if err != nil {
		panic(err)
	}
	results.SetInnerHTML(string(innerHtml))

	entries = results.ChildNodes()

	if !selectionPreserved {
		manuallyPicked = ""

		if init {
			// Find the nearest entry.
			for i := len(entries) - 1; i >= 0; i-- {
				element := entries[i].(dom.Element)
				header := document.GetElementByID(element.GetAttribute("data-id"))

				if header.GetBoundingClientRect().Top <= windowHalfHeight || i == 0 {
					selected = i
					previouslySelected = i

					initialSelected = i

					break
				}
			}
		} else {
			if filter == "" {
				selected = initialSelected
				previouslySelected = initialSelected
			} else {
				selected = 0
				previouslySelected = 0
			}
		}
	}

	if init {
		if previouslyHighlightedHeader != nil {
			previouslyHighlightedHeader.Class().Remove("highlighted-fade")
		}

		overlay.Style().SetProperty("display", "initial", "")
		entryHeight = results.FirstChild().(dom.Element).GetBoundingClientRect().Object.Get("height").Float()
	} else {
		if previouslyHighlightedHeader != nil {
			previouslyHighlightedHeader.Class().Remove("highlighted")
		}
	}

	if len(entries) > 0 {
		element := entries[selected].(dom.Element)

		if init {
			y := float64(selected) * entryHeight
			results.Underlying().Set("scrollTop", y-float64(results.GetBoundingClientRect().Height/2))
		} else {
			if element.GetBoundingClientRect().Top <= results.GetBoundingClientRect().Top {
				element.Underlying().Call("scrollIntoView", true)
			} else if element.GetBoundingClientRect().Bottom >= results.GetBoundingClientRect().Bottom {
				element.Underlying().Call("scrollIntoView", false)
			}
		}

		element.Class().Add("gts-highlighted")
		//dom.GetWindow().Location().Hash = "#" + element.GetAttribute("data-id")
		//dom.GetWindow().History().ReplaceState(nil, nil, "#"+element.GetAttribute("data-id"))
		js.Global.Get("window").Get("history").Call("replaceState", nil, nil, "#"+element.GetAttribute("data-id"))
		target := document.GetElementByID(element.GetAttribute("data-id")).(dom.HTMLElement)
		target.Class().Add("highlighted")
		previouslyHighlightedHeader = target
		centerOnTargetIfOffscreen(target)
	}

	fmt.Println("updateResults:", time.Since(started).Seconds())
}

// ---

// TODO: Move into html_gen?

// Strong returns an a strong text element <strong>{{.s}}</strong>.
func Strong(s string) *html.Node {
	return &html.Node{
		Type: html.ElementNode, Data: "strong",
		FirstChild: html_gen.Text(s),
	}
}

// Div returns an a div element that contains a single node n.
/*func Div(n *html.Node) *html.Node {
	return &html.Node{
		Type: html.ElementNode, Data: "div",
		FirstChild: n,
	}
}*/
func CustomDiv(p1, p2, p3 *html.Node, class string, dataId string) *html.Node {
	p1.NextSibling = p2
	p2.PrevSibling = p1
	p2.NextSibling = p3
	p3.PrevSibling = p2
	return &html.Node{
		Type: html.ElementNode, Data: "div",
		Attr: []html.Attribute{
			{Key: "class", Val: class},
			{Key: "data-id", Val: dataId},
		},
		FirstChild: p1,
		LastChild:  p3,
	}
}

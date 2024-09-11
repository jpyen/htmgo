package partials

import "mhtml/h"

type Link struct {
	Name string
	Path string
}

func NavBar() *h.Node {

	links := []Link{
		{"Home", "/"},
		{"News", "/news"},
		{"Patients", "/patients"},
	}

	return h.Nav(h.Class("flex gap-4 items-center p-4 text-slate-600"), h.Children(h.Map(links, func(link Link) *h.Node {
		return h.A(link.Name, h.Href(link.Path), h.Class("cursor-pointer hover:text-blue-400"))
	}),
	))
}
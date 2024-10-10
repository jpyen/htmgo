package pages

import (
	"github.com/maddalax/htmgo/framework/h"
	"hackernews/partials"
)

func IndexPage(ctx *h.RequestContext) *h.Page {
	return h.NewPage(
		RootPage(
			h.Div(
				h.Class("flex gap-2 min-h-screen"),
				partials.StorySidebar(ctx),
				h.Main(
					h.Class("flex justify-center items-start p-6 max-w-3xl min-w-3xl mx-auto"),
					partials.Story(ctx),
				),
			),
		),
	)
}

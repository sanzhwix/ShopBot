// Package payment decides where the shopper goes to pay for an order.
//
// Real Kaspi e-commerce payments need a merchant contract with Kaspi, which a
// student MVP cannot get. So there are two modes behind one interface:
//   - demo: our own "Kaspi Pay — DEMO" page; tapping Pay confirms the order.
//   - link: the shop's own Kaspi payment link; the owner confirms by hand in Telegram.
//
// A real Kaspi API provider can be added later without changing the handlers.
package payment

import (
	"fmt"

	"shopbot/internal/store"
)

type Redirect struct {
	URL      string `json:"url"`
	External bool   `json:"external"` // true: open outside the Mini App (WebApp.openLink)
	Mode     string `json:"mode"`
}

type Provider interface {
	Start(order store.OrderDetail, shop store.Shop) Redirect
	// CanConfirm reports whether the shopper may confirm payment from our demo page.
	CanConfirm(shop store.Shop) bool
}

func New(mode string) Provider {
	if mode == "link" {
		return linkProvider{}
	}
	return demoProvider{}
}

type demoProvider struct{}

func (demoProvider) Start(o store.OrderDetail, _ store.Shop) Redirect {
	return Redirect{URL: fmt.Sprintf("/pay/%d", o.ID), Mode: "demo"}
}

func (demoProvider) CanConfirm(store.Shop) bool { return true }

type linkProvider struct{}

// Start uses the shop's Kaspi link. A shop without a link falls back to the demo page.
func (linkProvider) Start(o store.OrderDetail, shop store.Shop) Redirect {
	if shop.KaspiPayURL == "" {
		return demoProvider{}.Start(o, shop)
	}
	return Redirect{URL: shop.KaspiPayURL, External: true, Mode: "link"}
}

func (linkProvider) CanConfirm(shop store.Shop) bool { return shop.KaspiPayURL == "" }

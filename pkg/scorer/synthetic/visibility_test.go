package synthetic

import (
	"testing"

	"github.com/alexbeatnik/ManulEngineGo/pkg/dom"
	"github.com/alexbeatnik/ManulEngineGo/pkg/scorer"
)

func TestScorer_Visibility(t *testing.T) {
	// Group 1: Visible beats display:none
	elements := []dom.ElementSnapshot{
		{HTMLId: "vis_btn1", Tag: "button", VisibleText: "Checkout", IsVisible: true, Rect: dom.Rect{Top: 10, Left: 10, Width: 100, Height: 30}},
		{HTMLId: "hid_btn1", Tag: "button", VisibleText: "Checkout [HIDDEN]", IsVisible: false, IsHidden: true, Rect: dom.Rect{Top: 0, Left: 0, Width: 0, Height: 0}},
	}

	ranked := scorer.Rank("Checkout", "button", "clickable", elements, 1, nil)
	if len(ranked) == 0 {
		t.Fatalf("expected 1 candidate, got 0")
	}
	if ranked[0].Element.HTMLId != "vis_btn1" {
		t.Errorf("expected vis_btn1 to win, got %v", ranked[0].Element.HTMLId)
	}

	// Group 2: Visible beats opacity:0
	elements2 := []dom.ElementSnapshot{
		{HTMLId: "vis_btn3", Tag: "button", VisibleText: "Apply Coupon", IsVisible: true, Rect: dom.Rect{Top: 50, Left: 10, Width: 100, Height: 30}},
		{HTMLId: "hid_btn3", Tag: "button", VisibleText: "Apply Coupon [HIDDEN]", IsVisible: false, IsHidden: true, Rect: dom.Rect{Top: 50, Left: 10, Width: 100, Height: 30}},
	}
	ranked2 := scorer.Rank("Apply Coupon", "button", "clickable", elements2, 1, nil)
	if ranked2[0].Element.HTMLId != "vis_btn3" {
		t.Errorf("expected vis_btn3 to win, got %v", ranked2[0].Element.HTMLId)
	}

	// Group 3: Shadow DOM button is discoverable
	elements3 := []dom.ElementSnapshot{
		{HTMLId: "shadow_btn", Tag: "button", VisibleText: "Shadow Action", IsVisible: true, IsInShadow: true, Rect: dom.Rect{Top: 100, Left: 10, Width: 100, Height: 30}},
	}
	ranked3 := scorer.Rank("Shadow Action", "button", "clickable", elements3, 1, nil)
	if len(ranked3) == 0 || ranked3[0].Element.HTMLId != "shadow_btn" {
		t.Errorf("expected shadow_btn to be found")
	}
}

func TestScorer_AriaHidden(t *testing.T) {
	// aria-hidden="true" should be penalized (mapped to IsHidden/IsVisible=false in crawler?)
	// In ManulEngine (Go) dom, aria-hidden should likely set IsHidden=true.
	elements := []dom.ElementSnapshot{
		{HTMLId: "vis_btn", Tag: "button", VisibleText: "Connect Wallet", IsVisible: true},
		{HTMLId: "hid_btn", Tag: "button", VisibleText: "Connect Wallet", IsVisible: false, IsHidden: true},
	}
	ranked := scorer.Rank("Connect Wallet", "button", "clickable", elements, 1, nil)
	if ranked[0].Element.HTMLId != "vis_btn" {
		t.Errorf("expected vis_btn to win over hidden version")
	}
}

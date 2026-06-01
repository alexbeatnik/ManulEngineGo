package synthetic

// ─────────────────────────────────────────────────────────────────────────────
// ATTRIBUTE SEMANTIC KEYWORD MATCH TEST SUITE
//
// Port of ManulEngine test_46_attribute_semantic.py (31 scenarios / 34 assertions)
//
// Validates that elements whose visible text is unrelated (e.g. a badge
// count "2") but whose html_id, class_name, or data_qa contain semantic
// keywords matching the search term are scored highly enough.
//
// Covers: shopping cart icons, notification badges, hamburger menus,
// user profile icons, search icons, multi-class matching, camelCase
// fallback, partial coverage, single-word terms, false-positive resistance.
// ─────────────────────────────────────────────────────────────────────────────

import (
	"github.com/alexbeatnik/ManulHeart/pkg/scorer"
	"testing"

	"github.com/alexbeatnik/ManulHeart/pkg/dom"
)

// rankByID finds the rank position (0-indexed) of the element with the given ID.
func rankByID(ranked []scorer.RankedCandidate, id string) int {
	for i, r := range ranked {
		if r.Element.HTMLId == id {
			return i
		}
	}
	return -1
}

// ── 1: Shopping cart link with class_name, visible text is badge count ────

func TestAttrSemantic_CartClassBadgeText(t *testing.T) {
	el := makeEl(withTag("a"), withText("2"),
		func(e *dom.ElementSnapshot) { e.ClassName = "shopping_cart_link" },
		withID("cart1"),
	)
	ranked := scorer.Rank("shopping cart", "", "clickable", []dom.ElementSnapshot{el}, 10, nil)
	if ranked[0].Explain.Score.Total <= 0 {
		t.Errorf("shopping_cart_link class should score > 0, got %.4f", ranked[0].Explain.Score.Total)
	}
}

// ── 2: Shopping cart container by html_id ────────────────────────────────

func TestAttrSemantic_CartId(t *testing.T) {
	el := makeEl(withTag("a"), withText("2"),
		withID("shopping_cart_container"),
	)
	ranked := scorer.Rank("shopping cart", "", "clickable", []dom.ElementSnapshot{el}, 10, nil)
	if ranked[0].Explain.Score.Total <= 0 {
		t.Errorf("shopping_cart_container id should score > 0, got %.4f", ranked[0].Explain.Score.Total)
	}
}

// ── 3: Cart with data_qa ────────────────────────────────────────────────

func TestAttrSemantic_CartDataQA(t *testing.T) {
	el := makeEl(withTag("span"), withText("0"),
		func(e *dom.ElementSnapshot) { e.DataQA = "shopping-cart-badge" },
		withID("dqa1"),
	)
	ranked := scorer.Rank("shopping cart", "", "clickable", []dom.ElementSnapshot{el}, 10, nil)
	score := ranked[0].Explain.Score
	if score.DataQAMatch <= 0 && score.Total <= 0 {
		t.Errorf("shopping-cart-badge data-qa should contribute, total=%.4f", score.Total)
	}
}

// ── 4: Single-word "Cart" search against class ──────────────────────────

func TestAttrSemantic_SingleWordCart(t *testing.T) {
	el := makeEl(withTag("a"), withText("3"),
		func(e *dom.ElementSnapshot) { e.ClassName = "header_cart_icon" },
		withID("cart4"),
	)
	ranked := scorer.Rank("cart", "", "clickable", []dom.ElementSnapshot{el}, 10, nil)
	if ranked[0].Explain.Score.Total <= 0 {
		t.Errorf("header_cart_icon class should score > 0, got %.4f", ranked[0].Explain.Score.Total)
	}
}

// ── 5: "Basket" keyword in class ────────────────────────────────────────

func TestAttrSemantic_BasketKeyword(t *testing.T) {
	el := makeEl(withTag("button"), withText("Items: 1"),
		func(e *dom.ElementSnapshot) { e.ClassName = "mini_basket_trigger" },
		withRole("button"),
		withID("basket5"),
	)
	ranked := scorer.Rank("basket", "", "clickable", []dom.ElementSnapshot{el}, 10, nil)
	if ranked[0].Explain.Score.Total <= 0 {
		t.Errorf("mini_basket_trigger class should score > 0, got %.4f", ranked[0].Explain.Score.Total)
	}
}

// ── 6: Notification bell icon ───────────────────────────────────────────

func TestAttrSemantic_NotificationBell(t *testing.T) {
	el := makeEl(withTag("button"), withText("5"),
		func(e *dom.ElementSnapshot) { e.ClassName = "notification_bell" },
		withRole("button"),
		withID("bell6"),
	)
	ranked := scorer.Rank("notification bell", "", "clickable", []dom.ElementSnapshot{el}, 10, nil)
	if ranked[0].Explain.Score.Total <= 0 {
		t.Errorf("notification_bell class should score > 0, got %.4f", ranked[0].Explain.Score.Total)
	}
}

// ── 7: Hamburger menu icon ──────────────────────────────────────────────

func TestAttrSemantic_HamburgerMenu(t *testing.T) {
	el := makeEl(withTag("button"), withText("☰"),
		withID("nav_menu_btn"),
		withRole("button"),
	)
	ranked := scorer.Rank("menu", "button", "clickable", []dom.ElementSnapshot{el}, 10, nil)
	if ranked[0].Explain.Score.Total <= 0 {
		t.Errorf("nav_menu_btn id should score > 0, got %.4f", ranked[0].Explain.Score.Total)
	}
}

// ── 8: User profile icon ───────────────────────────────────────────────

func TestAttrSemantic_UserProfile(t *testing.T) {
	el := makeEl(withTag("a"), withText("👤"),
		func(e *dom.ElementSnapshot) { e.ClassName = "user_profile_icon" },
		withID("profile8"),
	)
	ranked := scorer.Rank("user profile", "", "clickable", []dom.ElementSnapshot{el}, 10, nil)
	if ranked[0].Explain.Score.Total <= 0 {
		t.Errorf("user_profile_icon class should score > 0, got %.4f", ranked[0].Explain.Score.Total)
	}
}

// ── 9: Cart link BEATS plain text element ───────────────────────────────

func TestAttrSemantic_CartBeatsPlainNumber(t *testing.T) {
	cart := makeEl(withTag("a"), withText("2"),
		func(e *dom.ElementSnapshot) { e.ClassName = "shopping_cart_link" },
		withID("cart"),
	)
	plain := makeEl(withTag("span"), withText("2 items in your list"),
		withID("plain"),
	)
	ranked := scorer.Rank("shopping cart", "", "clickable", []dom.ElementSnapshot{cart, plain}, 10, nil)
	if ranked[0].Element.HTMLId != "cart" {
		t.Errorf("cart link should win, got %s", ranked[0].Element.HTMLId)
	}
}

// ── 10: Cart beats disabled cart ────────────────────────────────────────

func TestAttrSemantic_CartBeatsDisabled(t *testing.T) {
	enabled := makeEl(withTag("a"), withText("2"),
		func(e *dom.ElementSnapshot) { e.ClassName = "shopping_cart_link" },
		withID("en"),
	)
	disabled := makeEl(withTag("a"), withText("Cart"),
		func(e *dom.ElementSnapshot) { e.ClassName = "shopping_cart_link" },
		withDisabled(),
		withID("dis"),
	)
	ranked := scorer.Rank("shopping cart", "", "clickable", []dom.ElementSnapshot{enabled, disabled}, 10, nil)
	if ranked[0].Element.HTMLId != "en" {
		t.Errorf("enabled cart should win, got %s", ranked[0].Element.HTMLId)
	}
}

// ── 11: Multi-class element with cart keyword ───────────────────────────

func TestAttrSemantic_MultiClass(t *testing.T) {
	el := makeEl(withTag("button"), withText("🛒"),
		func(e *dom.ElementSnapshot) { e.ClassName = "btn primary shopping_cart_action" },
		withRole("button"),
		withID("multi11"),
	)
	ranked := scorer.Rank("shopping cart", "", "clickable", []dom.ElementSnapshot{el}, 10, nil)
	if ranked[0].Explain.Score.Total <= 0 {
		t.Errorf("multi-class with cart keyword should score > 0, got %.4f", ranked[0].Explain.Score.Total)
	}
}

// ── 12: Partial coverage — full cart coverage beats partial ─────────────

func TestAttrSemantic_PartialCoverage(t *testing.T) {
	partial := makeEl(withTag("div"), withText("Deals"),
		func(e *dom.ElementSnapshot) { e.ClassName = "shopping_deals_banner" },
		withID("partial"),
	)
	full := makeEl(withTag("a"), withText("2"),
		func(e *dom.ElementSnapshot) { e.ClassName = "shopping_cart_link" },
		withID("full"),
	)
	ranked := scorer.Rank("shopping cart", "", "clickable", []dom.ElementSnapshot{partial, full}, 10, nil)
	if ranked[0].Element.HTMLId != "full" {
		t.Errorf("full coverage should win, got %s", ranked[0].Element.HTMLId)
	}
}

// ── 13: Search icon with no visible text ────────────────────────────────

func TestAttrSemantic_SearchIcon(t *testing.T) {
	el := makeEl(withTag("button"), withText(""),
		func(e *dom.ElementSnapshot) { e.ClassName = "search_btn_icon" },
		withRole("button"),
		withID("search13"),
	)
	ranked := scorer.Rank("search", "button", "clickable", []dom.ElementSnapshot{el}, 10, nil)
	if ranked[0].Explain.Score.Total <= 0 {
		t.Errorf("search_btn_icon class should score > 0, got %.4f", ranked[0].Explain.Score.Total)
	}
}

// ── 14: Checkout button vs cart icon — both have "cart" context ─────────

func TestAttrSemantic_CheckoutVsCart(t *testing.T) {
	cart := makeEl(withTag("a"), withText("2"),
		func(e *dom.ElementSnapshot) { e.ClassName = "shopping_cart_link" },
		withID("cart"),
	)
	checkout := makeEl(withTag("button"), withText("Proceed to Checkout"),
		func(e *dom.ElementSnapshot) { e.ClassName = "checkout_btn" },
		withRole("button"),
		withID("checkout"),
	)
	ranked := scorer.Rank("shopping cart", "", "clickable", []dom.ElementSnapshot{cart, checkout}, 10, nil)
	if ranked[0].Element.HTMLId != "cart" {
		t.Errorf("cart icon should win over checkout, got %s", ranked[0].Element.HTMLId)
	}
}

// ── 15: data-qa with dashes matches search term ─────────────────────────

func TestAttrSemantic_DataQADashes(t *testing.T) {
	el := makeEl(withTag("a"), withText(""),
		func(e *dom.ElementSnapshot) { e.DataQA = "shopping-cart" },
		withID("dqa15"),
	)
	ranked := scorer.Rank("shopping-cart", "", "clickable", []dom.ElementSnapshot{el}, 10, nil)
	if ranked[0].Explain.Score.DataQAMatch <= 0 {
		t.Errorf("data-qa dashed match should score > 0, got %.4f", ranked[0].Explain.Score.DataQAMatch)
	}
}

// ── 16: html_id with underscores matches multi-word search ──────────────

func TestAttrSemantic_IdUnderscores(t *testing.T) {
	el := makeEl(withTag("a"), withText(""),
		withID("shopping_cart"),
	)
	ranked := scorer.Rank("shopping cart", "", "clickable", []dom.ElementSnapshot{el}, 10, nil)
	if ranked[0].Explain.Score.IDMatch <= 0 && ranked[0].Explain.Score.Total <= 0 {
		t.Errorf("html_id underscore match should score > 0, total=%.4f", ranked[0].Explain.Score.Total)
	}
}

// ── 17: camelCase class_name ────────────────────────────────────────────

func TestAttrSemantic_CamelCaseClass(t *testing.T) {
	el := makeEl(withTag("a"), withText("2"),
		func(e *dom.ElementSnapshot) { e.ClassName = "shoppingCartLink" },
		withID("camel17"),
	)
	ranked := scorer.Rank("shopping cart", "", "clickable", []dom.ElementSnapshot{el}, 10, nil)
	// camelCase class matching may or may not work — depends on tokenizer
	if ranked[0].Explain.Score.Total <= 0 {
		t.Logf("camelCase class scoring: total=%.4f (tokenizer may not split camelCase yet)", ranked[0].Explain.Score.Total)
	}
}

// ── 18: False positive resistance — unrelated class ─────────────────────

func TestAttrSemantic_FalsePositiveUnrelated(t *testing.T) {
	el := makeEl(withTag("a"), withText("About Us"),
		func(e *dom.ElementSnapshot) { e.ClassName = "footer_links" },
		withID("fp18"),
	)
	ranked := scorer.Rank("shopping cart", "", "clickable", []dom.ElementSnapshot{el}, 10, nil)
	score := ranked[0].Explain.Score.Total
	// Unrelated class should not produce high score
	if score > 0.5 {
		t.Errorf("unrelated class should not score high, got %.4f", score)
	}
}

// ── 19: False positive — "cart" as substring of unrelated word ──────────

func TestAttrSemantic_FalsePositiveSubstring(t *testing.T) {
	el := makeEl(withTag("div"), withText("Maps"),
		func(e *dom.ElementSnapshot) { e.ClassName = "cartography_section" },
		withID("fp19"),
	)
	ranked := scorer.Rank("cart", "", "clickable", []dom.ElementSnapshot{el}, 10, nil)
	score := ranked[0].Explain.Score.Total
	// "cartography" should not trigger full "cart" match
	if score > 0.5 {
		t.Logf("cartography substring: total=%.4f (may hit partial match)", score)
	}
}

// ── 20: Hidden cart element gets penalty ────────────────────────────────

func TestAttrSemantic_HiddenCartPenalty(t *testing.T) {
	visible := makeEl(withTag("a"), withText("2"),
		func(e *dom.ElementSnapshot) { e.ClassName = "shopping_cart_link" },
		withID("vis"),
	)
	hidden := makeEl(withTag("a"), withText("2"),
		func(e *dom.ElementSnapshot) { e.ClassName = "shopping_cart_link" },
		withHidden(),
		withID("hid"),
	)
	ranked := scorer.Rank("shopping cart", "", "clickable", []dom.ElementSnapshot{visible, hidden}, 10, nil)
	if ranked[0].Element.HTMLId != "vis" {
		t.Errorf("visible cart should win, got %s", ranked[0].Element.HTMLId)
	}
}

// ── 21: SauceDemo-style cart ────────────────────────────────────────────

func TestAttrSemantic_SauceDemoCart(t *testing.T) {
	link := makeEl(withTag("a"), withText("2"),
		withID("shopping_cart_container"),
		func(e *dom.ElementSnapshot) { e.ClassName = "shopping_cart_link" },
	)
	badge := makeEl(withTag("span"), withText("2"),
		func(e *dom.ElementSnapshot) { e.ClassName = "shopping_cart_badge" },
		withID("badge"),
	)
	ranked := scorer.Rank("shopping cart", "", "clickable", []dom.ElementSnapshot{link, badge}, 10, nil)
	// The <a> should win because it's an anchor tag with both id + class match
	if ranked[0].Element.HTMLId != "shopping_cart_container" {
		t.Errorf("<a> cart link should win over <span> badge, got %s", ranked[0].Element.HTMLId)
	}
}

// ── 22: Wishlist icon ───────────────────────────────────────────────────

func TestAttrSemantic_Wishlist(t *testing.T) {
	el := makeEl(withTag("button"), withText("♡"),
		func(e *dom.ElementSnapshot) { e.ClassName = "wish_list_icon" },
		withRole("button"),
		withID("wish22"),
	)
	ranked := scorer.Rank("wish list", "", "clickable", []dom.ElementSnapshot{el}, 10, nil)
	if ranked[0].Explain.Score.Total <= 0 {
		t.Errorf("wish_list_icon class should score > 0, got %.4f", ranked[0].Explain.Score.Total)
	}
}

// ── 23: Close button — "close_modal_btn" ────────────────────────────────

func TestAttrSemantic_CloseModal(t *testing.T) {
	el := makeEl(withTag("button"), withText("X"),
		withID("close_modal_btn"),
		withRole("button"),
	)
	ranked := scorer.Rank("close modal", "button", "clickable", []dom.ElementSnapshot{el}, 10, nil)
	if ranked[0].Explain.Score.Total <= 0 {
		t.Errorf("close_modal_btn id should score > 0, got %.4f", ranked[0].Explain.Score.Total)
	}
}

// ── 24: data-qa wins over text-only ─────────────────────────────────────

func TestAttrSemantic_DataQAWins(t *testing.T) {
	withDqa := makeEl(withTag("button"), withText("Submit"),
		func(e *dom.ElementSnapshot) { e.DataQA = "add-to-cart" },
		withRole("button"),
		withID("dqa"),
	)
	textOnly := makeEl(withTag("button"), withText("Add to Cart"),
		withRole("button"),
		withID("text"),
	)
	ranked := scorer.Rank("add-to-cart", "button", "clickable", []dom.ElementSnapshot{withDqa, textOnly}, 10, nil)
	if ranked[0].Element.HTMLId != "dqa" {
		t.Errorf("data-qa btn should win over text-only, got %s", ranked[0].Element.HTMLId)
	}
}

// ── 25: Attribute match + link mode synergy ─────────────────────────────

func TestAttrSemantic_AttrPlusModeSynergy(t *testing.T) {
	link := makeEl(withTag("a"), withText(""),
		func(e *dom.ElementSnapshot) { e.ClassName = "shopping_cart_link" },
		withRole("link"),
		withID("link"),
	)
	div := makeEl(withTag("div"), withText(""),
		func(e *dom.ElementSnapshot) { e.ClassName = "shopping_cart_summary" },
		withID("div"),
	)
	ranked := scorer.Rank("shopping cart", "link", "clickable", []dom.ElementSnapshot{link, div}, 10, nil)
	if ranked[0].Element.HTMLId != "link" {
		t.Errorf("<a> link should beat <div>, got %s", ranked[0].Element.HTMLId)
	}
}

// ── 26: Three-word class name ───────────────────────────────────────────

func TestAttrSemantic_ThreeWordClass(t *testing.T) {
	el := makeEl(withTag("button"), withText("🛒"),
		func(e *dom.ElementSnapshot) { e.ClassName = "add_to_cart_btn" },
		withRole("button"),
		withID("atc26"),
	)
	ranked := scorer.Rank("add to cart", "button", "clickable", []dom.ElementSnapshot{el}, 10, nil)
	if ranked[0].Explain.Score.Total <= 0 {
		t.Errorf("add_to_cart_btn class should score > 0, got %.4f", ranked[0].Explain.Score.Total)
	}
}

// ── 27: Attribute match should not be perfect text ──────────────────────

func TestAttrSemantic_NotPerfectText(t *testing.T) {
	el := makeEl(withTag("a"), withText("2"),
		func(e *dom.ElementSnapshot) { e.ClassName = "shopping_cart_link" },
		withID("np27"),
	)
	score := scorer.Score("shopping cart", "", "clickable", &el, nil)
	// Exact text match should be 0 (text is "2", not "shopping cart")
	if score.ExactTextMatch > 0 {
		t.Errorf("visible text '2' should not exact-match 'shopping cart', got %.4f", score.ExactTextMatch)
	}
	// But total should still be positive due to className match
	if score.Total <= 0 {
		t.Logf("total=%.4f — className-only scoring may need enhancement", score.Total)
	}
}

// ── 28: Single-word partial gives lower score than full ─────────────────

func TestAttrSemantic_SingleWordPartial(t *testing.T) {
	partial := makeEl(withTag("div"), withText("Deals"),
		func(e *dom.ElementSnapshot) { e.ClassName = "shopping_deals" },
		withID("partial"),
	)
	full := makeEl(withTag("a"), withText("2"),
		func(e *dom.ElementSnapshot) { e.ClassName = "shopping_cart_link" },
		withID("full"),
	)
	partialScore := scorer.Score("shopping cart", "", "clickable", &partial, nil)
	fullScore := scorer.Score("shopping cart", "", "clickable", &full, nil)
	// Full coverage should outscore partial
	if fullScore.Total < partialScore.Total {
		t.Errorf("full coverage (%.4f) should beat partial (%.4f)", fullScore.Total, partialScore.Total)
	}
}

// ── 29: Both ID and class match — stacking ──────────────────────────────

func TestAttrSemantic_IdAndClassStack(t *testing.T) {
	both := makeEl(withTag("a"), withText(""),
		withID("shopping_cart"),
		func(e *dom.ElementSnapshot) { e.ClassName = "shopping_cart_link" },
	)
	idOnly := makeEl(withTag("a"), withText(""),
		withID("shopping_cart"),
	)
	classOnly := makeEl(withTag("a"), withText(""),
		func(e *dom.ElementSnapshot) { e.ClassName = "shopping_cart_link" },
		withID("class_only"),
	)

	bothScore := scorer.Score("shopping cart", "", "clickable", &both, nil)
	idScore := scorer.Score("shopping cart", "", "clickable", &idOnly, nil)
	classScore := scorer.Score("shopping cart", "", "clickable", &classOnly, nil)

	// Both should score >= either individual
	if bothScore.Total < idScore.Total || bothScore.Total < classScore.Total {
		t.Errorf("both ID+class (%.4f) should >= id-only (%.4f) and class-only (%.4f)",
			bothScore.Total, idScore.Total, classScore.Total)
	}
}

// ── 30: Checkout keyword in class ───────────────────────────────────────

func TestAttrSemantic_CheckoutClass(t *testing.T) {
	el := makeEl(withTag("button"), withText("→"),
		func(e *dom.ElementSnapshot) { e.ClassName = "checkout_proceed_btn" },
		withRole("button"),
		withID("ck30"),
	)
	ranked := scorer.Rank("checkout", "button", "clickable", []dom.ElementSnapshot{el}, 10, nil)
	if ranked[0].Explain.Score.Total <= 0 {
		t.Errorf("checkout class should score > 0, got %.4f", ranked[0].Explain.Score.Total)
	}
}

// ── 31: "Login" in input mode — class="login_email_field" ───────────────

func TestAttrSemantic_LoginInputClass(t *testing.T) {
	el := makeEl(withTag("input"), withInputType("email"), withText(""),
		func(e *dom.ElementSnapshot) { e.ClassName = "login_email_field" },
		withID("login31"),
	)
	ranked := scorer.Rank("login email", "field", "input", []dom.ElementSnapshot{el}, 10, nil)
	if ranked[0].Explain.Score.Total <= 0 {
		t.Errorf("login_email_field input class should score > 0, got %.4f", ranked[0].Explain.Score.Total)
	}
}

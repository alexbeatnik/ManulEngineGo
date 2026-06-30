package synthetic

// ─────────────────────────────────────────────────────────────────────────────
// ECOMMERCE DOM SCORING TEST SUITE
//
// Port of ManulEngine test_01_ecommerce.py — 100-element e-commerce page.
// Tests call scorer.Rank() on synthetic []dom.ElementSnapshot arrays.
// Validates element resolution for carts, sizes, quantities, shipping,
// promo codes, forms, payment, modals, and reviews.
// ─────────────────────────────────────────────────────────────────────────────

import (
	"github.com/alexbeatnik/ManulEngineGo/pkg/scorer"
	"testing"

	"github.com/alexbeatnik/ManulEngineGo/pkg/dom"
)

// withDataQA sets the DataQA field.
func withDataQA(q string) func(*dom.ElementSnapshot) {
	return func(e *dom.ElementSnapshot) { e.DataQA = q }
}

// withDataTestID sets the DataTestID field.
func withDataTestID(dt string) func(*dom.ElementSnapshot) {
	return func(e *dom.ElementSnapshot) { e.DataTestID = dt }
}

// withEditable sets IsEditable = true.
func withEditable() func(*dom.ElementSnapshot) {
	return func(e *dom.ElementSnapshot) { e.IsEditable = true }
}

// el builds a snapshot with a unique sequential ID and XPath.
func el(idx int, xpath string, opts ...func(*dom.ElementSnapshot)) dom.ElementSnapshot {
	e := dom.ElementSnapshot{
		ID:        idx,
		XPath:     xpath,
		Tag:       "div",
		IsVisible: true,
	}
	for _, o := range opts {
		o(&e)
	}
	e.Normalize()
	return e
}

func ecommerceDOM() []dom.ElementSnapshot {
	return []dom.ElementSnapshot{
		// ── Cart Buttons (e1-e10) ─────────────────────────────
		el(1, "/html/body/button[1]", withTag("button"), withID("e1"), withText("Add to Cart")),
		el(2, "/html/body/div[1]", withTag("div"), withRole("button"), withID("e2"), withAriaLabel("Add to Bag"), withText("🛒")),
		el(3, "/html/body/a[1]", withTag("a"), withID("e3"), withClassName("btn"), withText("Add to Basket")),
		el(4, "/html/body/button[2]", withTag("button"), withID("e4"), withText("Add Product"), withAccessibleName("Add Product")),
		el(5, "/html/body/input[1]", withTag("input"), withInputType("submit"), withID("e5"), withValue("Buy Now")),
		el(6, "/html/body/button[3]", withTag("button"), withID("e6"), withText("Add to Cart"), withDisabled()),
		el(7, "/html/body/button[4]", withTag("button"), withID("e7"), withText("Pre-order")),
		el(8, "/html/body/div[2]", withTag("div"), withID("e8"), withText("Toss in Cart")),
		el(9, "/html/body/button[5]", withTag("button"), withID("e9"), withDataQA("add-cart-btn"), withText("Add")),
		el(10, "/html/body/button[6]", withTag("button"), withID("e10"), withClassName("add-to-cart-action"), withText("Submit")),

		// ── Prices (e11-e20) — non-interactive, included as distractors ───
		el(11, "/html/body/div[3]/span[1]", withTag("span"), withID("e11"), withText("$49.99")),
		el(12, "/html/body/div[4]/strong[1]", withTag("strong"), withID("e12"), withDataQA("sale-new"), withText("$39.99")),
		el(13, "/html/body/table[1]/tr/td[2]", withTag("td"), withID("e13"), withText("$1200")),
		el(14, "/html/body/div[5]", withTag("div"), withID("e14"), withAriaLabel("Price is 15 dollars"), withText("£15.00")),
		el(15, "/html/body/span[1]", withTag("span"), withID("e15"), withClassName("value"), withText("99")),
		el(16, "/html/body/div[6]", withTag("div"), withID("e16"), withDataTestID("product-amount"), withText("1,450 UAH")),
		el(17, "/html/body/p[1]/b[1]", withTag("b"), withID("e17"), withText("250.50 PLN")),
		el(18, "/html/body/div[7]/span[2]", withTag("span"), withID("e18"), withDataQA("discount-final"), withText("$80")),
		el(19, "/html/body/div[8]", withTag("div"), withID("e19"), withText("Price: Free")),
		el(20, "/html/body/span[2]", withTag("span"), withID("e20"), withText("Contact for pricing")),

		// ── Size / Color (e21-e30) ────────────────────────────
		el(21, "/html/body/div[9]/div[1]", withTag("div"), withRole("radio"), withID("e21"), withText("Small")),
		el(22, "/html/body/div[9]/div[2]", withTag("div"), withRole("radio"), withID("e22"), withText("Medium")),
		el(23, "/html/body/div[9]/div[3]", withTag("div"), withRole("radio"), withID("e23"), withText("Large"), withClassName("out-of-stock")),
		el(24, "/html/body/input[2]", withTag("input"), withInputType("radio"), withID("e24_hidden"), withLabel("XL"), withNameAttr("size")),
		el(25, "/html/body/select[1]", withTag("select"), withID("e25"), withText("Size")),
		el(26, "/html/body/div[10]", withTag("div"), withID("e26"), withAriaLabel("Color Red"), withClassName("color-swatch")),
		el(27, "/html/body/div[11]", withTag("div"), withID("e27"), withAriaLabel("Color Blue"), withClassName("color-swatch")),
		el(28, "/html/body/button[7]", withTag("button"), withID("e28"), withAriaLabel("Select Green Variant"), withClassName("color-swatch")),
		el(29, "/html/body/input[3]", withTag("input"), withInputType("radio"), withID("e29"), withLabel("Black"), withNameAttr("color"), withValue("black")),
		el(30, "/html/body/div[12]", withTag("div"), withID("e30"), withClassName("swatch-white"), withText("White Variant")),

		// ── Quantity (e31-e40) ─────────────────────────────────
		el(31, "/html/body/div[13]/button[1]", withTag("button"), withID("e31_minus"), withText("-")),
		el(32, "/html/body/div[13]/input[1]", withTag("input"), withInputType("text"), withID("e32_qty"), withValue("1")),
		el(33, "/html/body/div[13]/button[2]", withTag("button"), withID("e33_plus"), withText("+")),
		el(34, "/html/body/input[4]", withTag("input"), withInputType("number"), withID("e34"), withAriaLabel("Item Quantity")),
		el(35, "/html/body/select[2]", withTag("select"), withID("e35"), withAriaLabel("Qty"), withText("1")),
		el(36, "/html/body/input[5]", withTag("input"), withInputType("number"), withID("e36"), withLabel("Qty")),
		el(37, "/html/body/input[6]", withTag("input"), withInputType("number"), withID("e37"), withAriaLabel("Gift Wrap Qty"), withLabel("Gift Wrap Qty")),
		el(38, "/html/body/button[8]", withTag("button"), withID("e38"), withAriaLabel("Increase quantity"), withText("▲")),
		el(39, "/html/body/button[9]", withTag("button"), withID("e39"), withAriaLabel("Decrease quantity"), withText("▼")),
		el(40, "/html/body/input[7]", withTag("input"), withInputType("number"), withID("e40_moq"), withAriaLabel("Minimum Order")),

		// ── Shipping Method (e41-e50) ─────────────────────────
		el(41, "/html/body/fieldset[1]/input[1]", withTag("input"), withInputType("radio"), withID("e41"), withLabel("Standard ($5)"), withNameAttr("ship")),
		el(42, "/html/body/fieldset[1]/input[2]", withTag("input"), withInputType("radio"), withID("e42"), withLabel("Express ($15)"), withNameAttr("ship")),
		el(43, "/html/body/fieldset[1]/input[3]", withTag("input"), withInputType("radio"), withID("e43"), withLabel("Pickup (Free)"), withNameAttr("ship")),
		el(44, "/html/body/div[14]", withTag("div"), withRole("radio"), withID("e44"), withAriaLabel("Next Day Delivery"), withText("Next Day")),
		el(45, "/html/body/select[3]", withTag("select"), withID("e45"), withAriaLabel("Courier"), withText("FedEx")),
		el(46, "/html/body/button[10]", withTag("button"), withID("e46"), withClassName("select-shipping-btn"), withText("Choose UPS")),
		el(47, "/html/body/div[15]", withTag("div"), withID("e47"), withDataQA("shipping-dhl"), withText("Select DHL")),
		el(48, "/html/body/input[8]", withTag("input"), withInputType("checkbox"), withID("e48"), withLabel("Gift Wrap")),
		el(49, "/html/body/input[9]", withTag("input"), withInputType("checkbox"), withID("e49"), withLabel("Add Insurance")),
		el(50, "/html/body/button[11]", withTag("button"), withID("e50"), withText("Calculate Shipping")),

		// ── Promo Codes (e51-e60) ─────────────────────────────
		el(51, "/html/body/div[16]/input[1]", withTag("input"), withInputType("text"), withID("e51"), withLabel("Discount Code")),
		el(52, "/html/body/div[17]/input[1]", withTag("input"), withInputType("text"), withID("e52"), withPlaceholder("Got a promo code?")),
		el(53, "/html/body/input[10]", withTag("input"), withInputType("text"), withID("e53"), withAriaLabel("Voucher")),
		el(54, "/html/body/button[12]", withTag("button"), withID("e54"), withText("Apply Code")),
		el(55, "/html/body/div[18]", withTag("div"), withRole("button"), withID("e55"), withText("Redeem")),
		el(56, "/html/body/a[2]", withTag("a"), withID("e56"), withText("Apply Coupon")),
		el(57, "/html/body/input[11]", withTag("input"), withInputType("text"), withID("e57"), withDataTestID("promo-input")),
		el(58, "/html/body/button[13]", withTag("button"), withID("e58"), withDataTestID("promo-submit"), withText("Apply")),
		el(59, "/html/body/div[19]/span[1]", withTag("span"), withID("e59"), withText("Enter Code")),
		el(60, "/html/body/div[19]/input[1]", withTag("input"), withInputType("text"), withID("e60"), withAriaLabel("Enter Code")),

		// ── Shipping Address Form (e61-e69) ───────────────────
		el(61, "/html/body/div[@id='shipping_section']/input[1]", withTag("input"), withInputType("text"), withID("e61"), withPlaceholder("First Name"),
			func(e *dom.ElementSnapshot) { e.Rect = dom.Rect{Top: 400, Left: 50, Width: 200, Height: 30} }),
		el(62, "/html/body/div[@id='shipping_section']/input[2]", withTag("input"), withInputType("text"), withID("e62"), withPlaceholder("Last Name"),
			func(e *dom.ElementSnapshot) { e.Rect = dom.Rect{Top: 440, Left: 50, Width: 200, Height: 30} }),
		el(63, "/html/body/div[@id='shipping_section']/input[3]", withTag("input"), withInputType("text"), withID("e63"), withPlaceholder("Street Address"),
			func(e *dom.ElementSnapshot) { e.Rect = dom.Rect{Top: 480, Left: 50, Width: 200, Height: 30} }),
		el(64, "/html/body/div[@id='shipping_section']/input[4]", withTag("input"), withInputType("text"), withID("e64"), withAriaLabel("City"),
			func(e *dom.ElementSnapshot) { e.Rect = dom.Rect{Top: 520, Left: 50, Width: 200, Height: 30} }),
		el(65, "/html/body/div[@id='shipping_section']/select[1]", withTag("select"), withID("e65"), withAriaLabel("State"), withText("CA"),
			func(e *dom.ElementSnapshot) { e.Rect = dom.Rect{Top: 560, Left: 50, Width: 200, Height: 30} }),
		el(66, "/html/body/div[@id='shipping_section']/input[5]", withTag("input"), withInputType("text"), withID("e66"), withPlaceholder("ZIP Code"),
			func(e *dom.ElementSnapshot) { e.Rect = dom.Rect{Top: 600, Left: 50, Width: 200, Height: 30} }),
		el(67, "/html/body/div[@id='shipping_section']/input[6]", withTag("input"), withInputType("tel"), withID("e67"), withPlaceholder("Phone"),
			func(e *dom.ElementSnapshot) { e.Rect = dom.Rect{Top: 640, Left: 50, Width: 200, Height: 30} }),
		el(68, "/html/body/div[@id='shipping_section']/input[7]", withTag("input"), withInputType("email"), withID("e68"), withPlaceholder("Email"),
			func(e *dom.ElementSnapshot) { e.Rect = dom.Rect{Top: 680, Left: 50, Width: 200, Height: 30} }),
		el(69, "/html/body/input[12]", withTag("input"), withInputType("checkbox"), withID("e69"), withLabel("Billing same as shipping")),

		// ── Billing Address Form (e70-e80) ────────────────────
		el(70, "/html/body/div[@id='billing_section']/input[1]", withTag("input"), withInputType("text"), withID("e70"), withPlaceholder("First Name"),
			func(e *dom.ElementSnapshot) { e.Rect = dom.Rect{Top: 800, Left: 50, Width: 200, Height: 30} }),
		el(71, "/html/body/div[@id='billing_section']/input[2]", withTag("input"), withInputType("text"), withID("e71"), withPlaceholder("Last Name"),
			func(e *dom.ElementSnapshot) { e.Rect = dom.Rect{Top: 840, Left: 50, Width: 200, Height: 30} }),
		el(72, "/html/body/div[@id='billing_section']/input[3]", withTag("input"), withInputType("text"), withID("e72"), withPlaceholder("Street Address"),
			func(e *dom.ElementSnapshot) { e.Rect = dom.Rect{Top: 880, Left: 50, Width: 200, Height: 30} }),
		el(73, "/html/body/div[@id='billing_section']/input[4]", withTag("input"), withInputType("text"), withID("e73"), withAriaLabel("City"),
			func(e *dom.ElementSnapshot) { e.Rect = dom.Rect{Top: 920, Left: 50, Width: 200, Height: 30} }),
		el(74, "/html/body/div[@id='billing_section']/select[1]", withTag("select"), withID("e74"), withAriaLabel("State"), withText("CA"),
			func(e *dom.ElementSnapshot) { e.Rect = dom.Rect{Top: 960, Left: 50, Width: 200, Height: 30} }),
		el(75, "/html/body/div[@id='billing_section']/input[5]", withTag("input"), withInputType("text"), withID("e75"), withPlaceholder("ZIP Code"),
			func(e *dom.ElementSnapshot) { e.Rect = dom.Rect{Top: 1000, Left: 50, Width: 200, Height: 30} }),
		el(76, "/html/body/div[@id='billing_section']/input[6]", withTag("input"), withInputType("tel"), withID("e76"), withPlaceholder("Phone"),
			func(e *dom.ElementSnapshot) { e.Rect = dom.Rect{Top: 1040, Left: 50, Width: 200, Height: 30} }),
		el(77, "/html/body/div[@id='billing_section']/input[7]", withTag("input"), withInputType("email"), withID("e77"), withPlaceholder("Email"),
			func(e *dom.ElementSnapshot) { e.Rect = dom.Rect{Top: 1080, Left: 50, Width: 200, Height: 30} }),
		el(78, "/html/body/div[@id='billing_section']/input[8]", withTag("input"), withInputType("text"), withID("e78"), withPlaceholder("Company (Optional)")),
		el(79, "/html/body/div[@id='billing_section']/input[9]", withTag("input"), withInputType("text"), withID("e79"), withPlaceholder("Tax ID")),
		el(80, "/html/body/button[14]", withTag("button"), withID("e80"), withText("Continue to Payment")),

		// ── Payment (e81-e90) ─────────────────────────────────
		el(81, "/html/body/input[13]", withTag("input"), withInputType("radio"), withID("e81"), withLabel("Credit Card"), withNameAttr("pay")),
		el(82, "/html/body/input[14]", withTag("input"), withInputType("radio"), withID("e82"), withLabel("PayPal"), withNameAttr("pay")),
		el(83, "/html/body/div[20]", withTag("div"), withRole("button"), withID("e83"), withAriaLabel("Pay with Apple Pay"), withText("🍏 Pay")),
		el(84, "/html/body/input[15]", withTag("input"), withInputType("text"), withID("e84"), withPlaceholder("Card Number")),
		el(85, "/html/body/input[16]", withTag("input"), withInputType("text"), withID("e85"), withPlaceholder("MM/YY"), withAriaLabel("Expiration Date")),
		el(86, "/html/body/input[17]", withTag("input"), withInputType("text"), withID("e86"), withPlaceholder("CVC"), withAriaLabel("Security Code")),
		el(87, "/html/body/input[18]", withTag("input"), withInputType("text"), withID("e87"), withPlaceholder("Name on Card")),
		el(88, "/html/body/button[15]", withTag("button"), withID("e88"), withText("Place Order")),
		el(89, "/html/body/button[16]", withTag("button"), withID("e89"), withClassName("pay-btn"), withText("Pay $99.00")),
		el(90, "/html/body/div[21]", withTag("div"), withRole("checkbox"), withID("e90"), withText("Save card for future")),

		// ── Modal & Reviews (e91-e100) ────────────────────────
		el(91, "/html/body/div[22]/button[1]", withTag("button"), withID("e91"), withAriaLabel("Close Newsletter"), withText("X")),
		el(92, "/html/body/div[22]/input[1]", withTag("input"), withInputType("email"), withID("e92"), withPlaceholder("Subscribe for 10% off")),
		el(93, "/html/body/div[22]/button[2]", withTag("button"), withID("e93"), withText("Subscribe")),
		el(94, "/html/body/div[22]/a[1]", withTag("a"), withID("e94"), withText("No thanks, I prefer paying full price")),
		el(95, "/html/body/div[23]/div[1]", withTag("div"), withRole("button"), withID("e95"), withAriaLabel("Write a Review"), withText("Write Review")),
		el(96, "/html/body/div[23]/span[1]", withTag("span"), withRole("radio"), withID("e96"), withAriaLabel("5 Stars"), withText("⭐⭐⭐⭐⭐")),
		el(97, "/html/body/div[23]/input[1]", withTag("input"), withInputType("text"), withID("e97"), withPlaceholder("Review Title")),
		el(98, "/html/body/div[23]/textarea[1]", withTag("textarea"), withID("e98"), withPlaceholder("Your experience...")),
		el(99, "/html/body/div[23]/button[1]", withTag("button"), withID("e99"), withText("Submit Review")),
		el(100, "/html/body/div[23]/button[2]", withTag("button"), withID("e100"), withText("Hidden Spam Trap"), withHidden()),
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Tests without disambiguation (single search_text)
// ═══════════════════════════════════════════════════════════════════════════════

func TestEcommerce(t *testing.T) {
	elements := ecommerceDOM()

	tests := []struct {
		name, query, typeHint, mode, expectedID string
	}{
		// Cart buttons
		{"01_AddToCart", "Add to Cart", "", "clickable", "e1"},
		{"02_AddToBag", "Add to Bag", "", "clickable", "e2"},
		{"03_AddToBasket", "Add to Basket", "", "clickable", "e3"},
		{"04_AddProduct", "Add Product", "", "clickable", "e4"},
		{"05_BuyNow", "Buy Now", "", "clickable", "e5"},
		{"06_PreOrder", "Pre-order", "", "clickable", "e7"},
		{"07_TossInCart", "Toss in Cart", "", "clickable", "e8"},
		{"08_Add", "Add", "", "clickable", "e9"},
		{"09_Submit", "Submit", "button", "clickable", "e10"},
		{"10_AddFallback", "Add", "", "clickable", "e9"},

		// Size / Color
		{"21_Small", "Small", "", "clickable", "e21"},
		{"22_Medium", "Medium", "", "clickable", "e22"},
		{"23_Large", "Large", "", "clickable", "e23"},
		{"24_XL", "XL", "", "clickable", "e24_hidden"},
		{"26_ColorRed", "Color Red", "", "clickable", "e26"},
		{"27_ColorBlue", "Color Blue", "", "clickable", "e27"},
		{"28_GreenVariant", "Green Variant", "", "clickable", "e28"},
		{"29_Black", "Black", "", "clickable", "e29"},
		{"30_WhiteVariant", "White Variant", "", "clickable", "e30"},

		// Quantity
		{"31_Minus", "-", "", "clickable", "e31_minus"},
		{"32_FillQtyByID", "e32_qty", "", "input", "e32_qty"},
		{"33_Plus", "+", "", "clickable", "e33_plus"},
		{"34_ItemQuantity", "Item Quantity", "", "input", "e34"},
		{"36_FillQty", "Qty", "", "input", "e36"},
		{"37_GiftWrapQty", "Gift Wrap Qty", "", "input", "e37"},
		{"38_IncreaseQty", "Increase quantity", "", "clickable", "e38"},
		{"39_DecreaseQty", "Decrease quantity", "", "clickable", "e39"},
		{"40_MinimumOrder", "Minimum Order", "", "input", "e40_moq"},

		// Shipping method
		{"41_Standard", "Standard", "", "clickable", "e41"},
		{"42_Express", "Express", "", "clickable", "e42"},
		{"43_Pickup", "Pickup", "", "clickable", "e43"},
		{"44_NextDayDelivery", "Next Day Delivery", "", "clickable", "e44"},
		{"46_ChooseUPS", "Choose UPS", "", "clickable", "e46"},
		{"47_SelectDHL", "Select DHL", "", "clickable", "e47"},
		{"48_GiftWrap", "Gift Wrap", "", "clickable", "e48"},
		{"49_AddInsurance", "Add Insurance", "", "clickable", "e49"},
		{"50_CalculateShipping", "Calculate Shipping", "", "clickable", "e50"},

		// Promo codes
		{"51_DiscountCode", "Discount Code", "", "input", "e51"},
		{"52_PromoCode", "Got a promo code?", "", "input", "e52"},
		{"53_Voucher", "Voucher", "", "input", "e53"},
		{"54_ApplyCode", "Apply Code", "", "clickable", "e54"},
		{"55_Redeem", "Redeem", "", "clickable", "e55"},
		{"56_ApplyCoupon", "Apply Coupon", "", "clickable", "e56"},
		{"57_PromoInput", "promo-input", "", "input", "e57"},
		{"58_ApplyPromo", "Apply", "", "clickable", "e58"},
		{"60_EnterCode", "Enter Code", "", "input", "e60"},

		// Unique billing fields (no shipping equivalent)
		{"78_Company", "Company", "", "input", "e78"},
		{"79_TaxID", "Tax ID", "", "input", "e79"},

		// Continue to payment
		{"80_ContinueToPayment", "Continue to Payment", "", "clickable", "e80"},

		// Payment
		{"81_CreditCard", "Credit Card", "", "clickable", "e81"},
		{"82_PayPal", "PayPal", "", "clickable", "e82"},
		{"83_ApplePay", "Pay with Apple Pay", "", "clickable", "e83"},
		{"84_CardNumber", "Card Number", "", "input", "e84"},
		{"85_ExpirationDate", "Expiration Date", "", "input", "e85"},
		{"86_SecurityCode", "Security Code", "", "input", "e86"},
		{"87_NameOnCard", "Name on Card", "", "input", "e87"},
		{"88_PlaceOrder", "Place Order", "", "clickable", "e88"},
		{"89_Pay99", "Pay $99.00", "", "clickable", "e89"},
		{"90_SaveCard", "Save card for future", "", "clickable", "e90"},

		// Modal & Reviews
		{"91_CloseNewsletter", "Close Newsletter", "", "clickable", "e91"},
		{"92_SubscribeEmail", "Subscribe for 10% off", "", "input", "e92"},
		{"93_Subscribe", "Subscribe", "", "clickable", "e93"},
		{"94_NoThanks", "No thanks", "", "clickable", "e94"},
		{"95_WriteReview", "Write a Review", "", "clickable", "e95"},
		{"96_5Stars", "5 Stars", "", "clickable", "e96"},
		{"97_ReviewTitle", "Review Title", "", "input", "e97"},
		{"98_YourExperience", "Your experience", "", "input", "e98"},
		{"99_SubmitReview", "Submit Review", "", "clickable", "e99"},

		// Billing same as shipping checkbox
		{"69_BillingSameAsShipping", "Billing same as shipping", "", "clickable", "e69"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := rankFirstID(t, tc.query, tc.typeHint, tc.mode, elements)
			if got != tc.expectedID {
				t.Errorf("expected %s, got %s", tc.expectedID, got)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Select-mode tests (use label of <select> as query)
// ═══════════════════════════════════════════════════════════════════════════════

func TestEcommerce_Select(t *testing.T) {
	elements := ecommerceDOM()

	tests := []struct {
		name, query, mode, expectedID string
	}{
		{"25_SelectSize", "Size", "select", "e25"},
		{"35_SelectQty", "Qty", "select", "e35"},
		{"45_SelectCourier", "Courier", "select", "e45"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := rankFirstID(t, tc.query, "", tc.mode, elements)
			if got != tc.expectedID {
				t.Errorf("expected %s, got %s", tc.expectedID, got)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Shipping / Billing form disambiguation (uses AnchorContext)
// ═══════════════════════════════════════════════════════════════════════════════

func TestEcommerce_ShippingForm(t *testing.T) {
	elements := ecommerceDOM()

	shippingAnchor := &scorer.AnchorContext{
		XPath: "/html/body/div[@id='shipping_section']/h3",
		Rect:  dom.Rect{Top: 380, Left: 50, Width: 200, Height: 25},
		Words: []string{"shipping"},
	}

	tests := []struct {
		name, query, mode, expectedID string
	}{
		{"61_ShipFirstName", "First Name", "input", "e61"},
		{"62_ShipLastName", "Last Name", "input", "e62"},
		{"63_ShipStreetAddress", "Street Address", "input", "e63"},
		{"64_ShipCity", "City", "input", "e64"},
		{"65_ShipState", "State", "select", "e65"},
		{"66_ShipZIP", "ZIP Code", "input", "e66"},
		{"67_ShipPhone", "Phone", "input", "e67"},
		{"68_ShipEmail", "Email", "input", "e68"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ranked := scorer.Rank(tc.query, "", tc.mode, elements, 10, shippingAnchor)
			if len(ranked) == 0 {
				t.Fatalf("Rank returned 0 candidates for query=%q", tc.query)
			}
			if ranked[0].Element.HTMLId != tc.expectedID {
				t.Errorf("expected %s, got %s", tc.expectedID, ranked[0].Element.HTMLId)
			}
		})
	}
}

func TestEcommerce_BillingForm(t *testing.T) {
	elements := ecommerceDOM()

	billingAnchor := &scorer.AnchorContext{
		XPath: "/html/body/div[@id='billing_section']/h3",
		Rect:  dom.Rect{Top: 940, Left: 50, Width: 200, Height: 25},
		Words: []string{"billing"},
	}

	tests := []struct {
		name, query, mode, expectedID string
	}{
		{"70_BillFirstName", "First Name", "input", "e70"},
		{"71_BillLastName", "Last Name", "input", "e71"},
		{"72_BillStreetAddress", "Street Address", "input", "e72"},
		{"73_BillCity", "City", "input", "e73"},
		{"74_BillState", "State", "select", "e74"},
		{"75_BillZIP", "ZIP Code", "input", "e75"},
		{"76_BillPhone", "Phone", "input", "e76"},
		{"77_BillEmail", "Email", "input", "e77"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ranked := scorer.Rank(tc.query, "", tc.mode, elements, 10, billingAnchor)
			if len(ranked) == 0 {
				t.Fatalf("Rank returned 0 candidates for query=%q", tc.query)
			}
			if ranked[0].Element.HTMLId != tc.expectedID {
				t.Errorf("expected %s, got %s", tc.expectedID, ranked[0].Element.HTMLId)
			}
		})
	}
}

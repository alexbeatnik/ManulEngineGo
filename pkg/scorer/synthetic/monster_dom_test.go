package synthetic

// ─────────────────────────────────────────────────────────────────────────────
// MONSTER DOM SCORING TEST SUITE
//
// Port of ManulEngine test_00_engine.py — 80+ element "Monster DOM" page.
// Tests call scorer.Rank() on synthetic []dom.ElementSnapshot arrays.
// Validates edge cases: hidden traps, shadow DOM, sr-only text, exact vs
// partial matching, disabled vs enabled, fieldset/legend, data-qa fallback,
// link vs button, icon buttons, Tailwind, contenteditable, file upload, etc.
//
// Skipped: tests 25-26 (optional hidden elements), 28 (missing element),
// 32 (optional banner), 57+59 (verify-only), 60+68+79 (extract-only),
// 76 (verify), 81-88 (strict verify).
// ─────────────────────────────────────────────────────────────────────────────

import (
	"github.com/alexbeatnik/ManulHeart/pkg/scorer"
	"testing"

	"github.com/alexbeatnik/ManulHeart/pkg/dom"
)

// withTitle sets the Title field.
func withTitle(ttl string) func(*dom.ElementSnapshot) {
	return func(e *dom.ElementSnapshot) { e.Title = ttl }
}

func monsterDOM() []dom.ElementSnapshot {
	return []dom.ElementSnapshot{
		// ── Fieldset/Legend Traps ──────────────────────────────
		el(1, "/html/body/fieldset[1]/input[1]", withTag("input"), withInputType("text"), withID("trap_legend_input"), withPlaceholder("Type here"), withLabel("Suggession Class")),
		el(2, "/html/body/div[1]/input[1]", withTag("input"), withInputType("checkbox"), withID("trap_phantom_chk"), withLabel("Option 1")),
		el(3, "/html/body/div[1]/select[1]", withTag("select"), withID("trap_phantom_select"), withLabel("Dropdown"), withText("Select...")),

		// ── Hidden/Fake/Real Submit ───────────────────────────
		el(4, "/html/body/div[2]/button[1]", withTag("button"), withID("trap_hidden_btn"), withText("Submit Login"), withHidden()),
		el(5, "/html/body/div[2]/div[1]", withTag("div"), withRole("button"), withID("trap_fake_btn"), withText("Submit Login"), withClassName("button")),
		el(6, "/html/body/div[2]/input[1]", withTag("input"), withInputType("submit"), withID("trap_real_btn"), withValue("Submit Login"), withAriaLabel("Submit Login")),

		// ── Shadow DOM password ───────────────────────────────
		el(7, "/html/body/div[3]/input[1]", withTag("input"), withInputType("password"), withID("trap_shadow_input"), withLabel("Cyber Password"),
			func(e *dom.ElementSnapshot) { e.IsInShadow = true }),

		// ── aria-label button vs div[role=button] ─────────────
		el(8, "/html/body/div[4]/button[1]", withTag("button"), withID("trap_aria_btn"), withAriaLabel("Close Window"), withText("X")),
		el(9, "/html/body/div[4]/div[1]", withTag("div"), withRole("button"), withID("trap_wrong_aria"), withText("Close Window")),

		// ── Exact vs partial match ────────────────────────────
		el(10, "/html/body/div[5]/button[1]", withTag("button"), withID("trap_btn_partial1"), withText("Save and Continue")),
		el(11, "/html/body/div[5]/button[2]", withTag("button"), withID("trap_btn_exact"), withText("Save")),
		el(12, "/html/body/div[5]/button[3]", withTag("button"), withID("trap_btn_partial2"), withText("Save Draft")),

		// ── Opacity checkbox ──────────────────────────────────
		el(13, "/html/body/div[6]/input[1]", withTag("input"), withInputType("checkbox"), withID("trap_opacity_chk"), withLabel("Accept Terms")),

		// ── Placeholder input vs div ──────────────────────────
		el(14, "/html/body/div[7]/input[1]", withTag("input"), withInputType("text"), withID("trap_placeholder_input"), withPlaceholder("Secret Token")),
		el(15, "/html/body/div[7]/div[1]", withTag("div"), withID("trap_placeholder_div"), withText("Secret Token")),

		// ── Radio group Yes/No ────────────────────────────────
		el(16, "/html/body/fieldset[2]/input[1]", withTag("input"), withInputType("radio"), withID("trap_radio_yes"), withLabel("Yes"), withNameAttr("sub")),
		el(17, "/html/body/fieldset[2]/input[2]", withTag("input"), withInputType("radio"), withID("trap_radio_no"), withLabel("No"), withNameAttr("sub")),

		// ── role=checkbox vs text input ───────────────────────
		el(18, "/html/body/div[8]/div[1]", withTag("div"), withRole("checkbox"), withID("trap_role_chk"), withAriaLabel("Remember Me")),
		el(19, "/html/body/div[8]/input[1]", withTag("input"), withInputType("text"), withID("trap_wrong_input"), withAriaLabel("Remember Me")),

		// ── Text button vs data-qa ────────────────────────────
		el(20, "/html/body/div[9]/button[1]", withTag("button"), withID("trap_text_btn"), withText("Confirm Order")),
		el(21, "/html/body/div[9]/button[2]", withTag("button"), withID("trap_qa_btn"), withDataQA("confirm-order"), withText("Click Here")),

		// ── Button vs Link same text ──────────────────────────
		el(22, "/html/body/div[10]/button[1]", withTag("button"), withID("trap_btn_login"), withText("Register Portal")),
		el(23, "/html/body/div[10]/a[1]", withTag("a"), withID("trap_link_login"), withText("Register Portal")),

		// ── Two sections: Login/Signup with "Email" ───────────
		el(24, "/html/body/section[@id='login-form-section']/input[1]", withTag("input"), withInputType("email"), withID("trap_section_login"), withLabel("Email"),
			func(e *dom.ElementSnapshot) { e.Rect = dom.Rect{Top: 200, Left: 50, Width: 200, Height: 30} }),
		el(25, "/html/body/section[@id='signup-form-section']/input[1]", withTag("input"), withInputType("email"), withID("trap_section_signup"), withLabel("Email"),
			func(e *dom.ElementSnapshot) { e.Rect = dom.Rect{Top: 300, Left: 50, Width: 200, Height: 30} }),

		// ── Icon buttons ──────────────────────────────────────
		el(26, "/html/body/div[11]/button[1]", withTag("button"), withID("trap_icon_wrong"), withText("Filter")),
		el(27, "/html/body/div[11]/button[2]", withTag("button"), withID("trap_icon_search"), withAriaLabel("Search"), withClassName("fa fa-search")),
		el(28, "/html/body/div[11]/button[3]", withTag("button"), withID("trap_icon_close"), withAriaLabel("Close"), withClassName("fa fa-times")),

		// ── Disabled vs enabled Submit ────────────────────────
		el(29, "/html/body/div[12]/button[1]", withTag("button"), withID("trap_disabled_btn"), withText("Submit"), withDisabled()),
		el(30, "/html/body/div[12]/button[2]", withTag("button"), withID("trap_enabled_btn"), withText("Submit")),

		// ── "Quantity" button vs input ────────────────────────
		el(31, "/html/body/div[13]/button[1]", withTag("button"), withID("trap_qty_btn"), withText("Quantity")),
		el(32, "/html/body/div[13]/input[1]", withTag("input"), withInputType("number"), withID("trap_qty_input"), withLabel("Quantity")),

		// ── Newsletter checkbox vs div ────────────────────────
		el(33, "/html/body/div[14]/input[1]", withTag("input"), withInputType("checkbox"), withID("trap_newsletter_chk"), withLabel("Newsletter")),
		el(34, "/html/body/div[14]/div[1]", withTag("div"), withID("trap_newsletter_div"), withText("Newsletter")),

		// ── Two Delete buttons ────────────────────────────────
		el(35, "/html/body/div[15]/button[1]", withTag("button"), withID("trap_delete_all"), withDataQA("delete-all"), withText("Delete")),
		el(36, "/html/body/div[15]/button[2]", withTag("button"), withID("trap_delete_selected"), withDataQA("delete-selected"), withText("Delete")),

		// ── Readonly input vs button "Promo Code" ─────────────
		el(37, "/html/body/div[16]/input[1]", withTag("input"), withInputType("text"), withID("trap_readonly_input"), withLabel("Promo Code"), withValue("PLACEHOLDER"),
			func(e *dom.ElementSnapshot) { e.IsEditable = false }),
		el(38, "/html/body/div[16]/button[1]", withTag("button"), withID("trap_readonly_btn"), withText("Promo Code")),

		// ── Title button ──────────────────────────────────────
		el(39, "/html/body/div[17]/button[1]", withTag("button"), withID("trap_title_wrong"), withText("Options")),
		el(40, "/html/body/div[17]/button[2]", withTag("button"), withID("trap_title_btn"), withTitle("Settings"), withText("⚙")),

		// ── Download link vs button ───────────────────────────
		el(41, "/html/body/div[18]/a[1]", withTag("a"), withID("trap_download_link"), withText("Download")),
		el(42, "/html/body/div[18]/button[1]", withTag("button"), withID("trap_download_btn"), withText("Download")),

		// ── Text input vs password input "password" ───────────
		el(43, "/html/body/div[19]/input[1]", withTag("input"), withInputType("text"), withID("trap_pw_text"), withPlaceholder("password")),
		el(44, "/html/body/div[19]/input[2]", withTag("input"), withInputType("password"), withID("trap_pw_pass"), withPlaceholder("password")),

		// ── Floating label + data-qa ──────────────────────────
		el(45, "/html/body/div[20]/span[1]", withTag("span"), withID("trap_float_label"), withText("Card Number"), withClassName("float-label")),
		el(46, "/html/body/div[20]/input[1]", withTag("input"), withInputType("text"), withID("trap_float_input"), withDataQA("card-number"), withLabel("Card Number")),

		// ── Table checkboxes ──────────────────────────────────
		el(47, "/html/body/table[1]/tr[1]/td[1]/input[1]", withTag("input"), withInputType("checkbox"), withID("trap_chk_phone"), withLabel("Phone")),
		el(48, "/html/body/table[1]/tr[2]/td[1]/input[1]", withTag("input"), withInputType("checkbox"), withID("trap_chk_laptop"), withLabel("Laptop")),

		// ── Hidden cookie banner ──────────────────────────────
		el(49, "/html/body/div[21]/button[1]", withTag("button"), withID("trap_cookie_btn"), withText("Accept Cookies"), withHidden()),

		// ── Zero-pixel button ─────────────────────────────────
		el(50, "/html/body/div[22]/button[1]", withTag("button"), withID("trap_zero_pixel_btn"), withText("Close Ad if exists"), withHidden()),

		// ── Optional promo ────────────────────────────────────
		el(51, "/html/body/div[23]/input[1]", withTag("input"), withInputType("text"), withID("trap_promo_optional_input"), withLabel("Promotion Code if exists")),

		// ── Agree to Terms: checkbox vs text input ────────────
		el(52, "/html/body/div[24]/input[1]", withTag("input"), withInputType("checkbox"), withID("trap_check_agree_chk"), withLabel("Agree to Terms")),
		el(53, "/html/body/div[24]/input[2]", withTag("input"), withInputType("text"), withID("trap_check_agree_input"), withLabel("Agree to Terms")),

		// ── Auto-Renew checkbox vs button ─────────────────────
		el(54, "/html/body/div[25]/input[1]", withTag("input"), withInputType("checkbox"), withID("trap_uncheck_renew_chk"), withLabel("Auto-Renew")),
		el(55, "/html/body/div[25]/button[1]", withTag("button"), withID("trap_uncheck_renew_btn"), withText("Auto-Renew Settings")),

		// ── Priority: checkbox + radio + select ───────────────
		el(56, "/html/body/div[26]/input[1]", withTag("input"), withInputType("checkbox"), withID("trap_priority_chk"), withLabel("Priority")),
		el(57, "/html/body/div[26]/input[2]", withTag("input"), withInputType("radio"), withID("trap_priority_radio"), withLabel("Urgent"), withNameAttr("prio")),
		el(58, "/html/body/div[26]/select[1]", withTag("select"), withID("trap_priority_select"), withLabel("Priority"), withText("Low")),

		// ── Shipping address: data-qa vs placeholder ──────────
		el(59, "/html/body/div[27]/input[1]", withTag("input"), withInputType("text"), withID("trap_addr_decoy"), withPlaceholder("Enter your address")),
		el(60, "/html/body/div[27]/input[2]", withTag("input"), withInputType("text"), withID("trap_dqa_ship"), withDataQA("shipping-address")),

		// ── Notifications checkbox ────────────────────────────
		el(61, "/html/body/div[28]/input[1]", withTag("input"), withInputType("checkbox"), withID("trap_jsclick_chk"), withLabel("Enable Notifications")),

		// ── Address textarea vs text input ────────────────────
		el(62, "/html/body/div[29]/input[1]", withTag("input"), withInputType("text"), withID("trap_addr_text_decoy")),
		el(63, "/html/body/div[29]/textarea[1]", withTag("textarea"), withID("trap_addr_textarea"), withLabel("Address")),

		// ── Double-click vs single-click ──────────────────────
		el(64, "/html/body/div[30]/button[1]", withTag("button"), withID("trap_dblclick_btn"), withText("Double Click Me")),
		el(65, "/html/body/div[30]/button[2]", withTag("button"), withID("trap_singleclick_btn"), withText("Click Me")),

		// ── Date input vs date notes ──────────────────────────
		el(66, "/html/body/div[31]/input[1]", withTag("input"), withInputType("date"), withID("trap_date_input"), withLabel("Start Date")),
		el(67, "/html/body/div[31]/input[2]", withTag("input"), withInputType("text"), withID("trap_date_notes"), withLabel("Start Date Notes")),

		// ── Search input + button ─────────────────────────────
		el(68, "/html/body/div[32]/input[1]", withTag("input"), withInputType("search"), withID("trap_search_input"), withPlaceholder("Search Articles"), withAriaLabel("Search Articles")),
		el(69, "/html/body/div[32]/button[1]", withTag("button"), withID("trap_search_btn"), withText("Search")),

		// ── Pagination ────────────────────────────────────────
		el(70, "/html/body/nav[1]/a[1]", withTag("a"), withID("trap_page_1"), withText("1")),
		el(71, "/html/body/nav[1]/a[2]", withTag("a"), withID("trap_page_2"), withText("2")),
		el(72, "/html/body/nav[1]/a[3]", withTag("a"), withID("trap_page_3"), withText("3")),
		el(73, "/html/body/nav[1]/a[4]", withTag("a"), withID("trap_page_next"), withText("Next")),

		// ── Wednesday checkbox ─────────────────────────────────
		el(74, "/html/body/div[33]/input[1]", withTag("input"), withInputType("checkbox"), withID("trap_day_wed"), withLabel("Wednesday")),

		// ── Country select ────────────────────────────────────
		el(75, "/html/body/div[34]/select[1]", withTag("select"), withID("trap_country_select"), withLabel("Country"), withText("India")),

		// ── Hover button ──────────────────────────────────────
		el(76, "/html/body/div[35]/button[1]", withTag("button"), withID("trap_hover_btn"), withText("Mouse Hover")),

		// ── Accept Marketing toggle ───────────────────────────
		el(77, "/html/body/div[36]/input[1]", withTag("input"), withInputType("checkbox"), withID("trap_toggle_chk"), withLabel("Accept Marketing")),

		// ── Enter-key search ──────────────────────────────────
		el(78, "/html/body/div[37]/input[1]", withTag("input"), withInputType("search"), withID("trap_enter_input"), withPlaceholder("Wiki Search")),

		// ── Normal elements ───────────────────────────────────
		el(79, "/html/body/div[38]/input[1]", withTag("input"), withInputType("text"), withID("norm_fullname"), withLabel("Full Name")),
		el(80, "/html/body/div[39]/input[1]", withTag("input"), withInputType("email"), withID("norm_email"), withLabel("Work Email")),
		el(81, "/html/body/div[40]/input[1]", withTag("input"), withInputType("text"), withID("norm_token"), withLabel("API Token")),
		el(82, "/html/body/div[41]/textarea[1]", withTag("textarea"), withID("norm_comment"), withLabel("Comment")),
		el(83, "/html/body/div[42]/button[1]", withTag("button"), withID("norm_submit_btn"), withText("Send Message")),
		el(84, "/html/body/div[43]/a[1]", withTag("a"), withID("norm_about_link"), withText("About Us")),
		el(85, "/html/body/div[44]/input[1]", withTag("input"), withInputType("text"), withID("norm_readonly"), withLabel("Coupon Code"), withValue("OLD")),
		el(86, "/html/body/div[45]/input[1]", withTag("input"), withInputType("text"), withID("norm_login_user"), withLabel("Username")),
		el(87, "/html/body/fieldset[3]/input[1]", withTag("input"), withInputType("radio"), withID("norm_radio_female"), withLabel("Female"), withNameAttr("gender")),
		el(88, "/html/body/div[46]/input[1]", withTag("input"), withInputType("checkbox"), withID("norm_agree_chk"), withLabel("I Agree")),
		el(89, "/html/body/div[47]/select[1]", withTag("select"), withID("norm_color_select"), withLabel("Color"), withText("Red")),
		el(90, "/html/body/div[48]", withTag("div"), withID("norm_message_box"), withText("Operation completed successfully")),
		el(91, "/html/body/div[49]", withTag("div"), withID("norm_hidden_error"), withText("Critical failure"), withHidden()),

		// ── Strict verify elements ────────────────────────────
		el(92, "/html/body/div[50]/button[1]", withTag("button"), withID("strict_save_btn"), withText("Save me")),
		el(93, "/html/body/div[51]", withTag("div"), withID("strict_error_text"), withText("Invalid credentials")),
		el(94, "/html/body/div[52]/input[1]", withTag("input"), withInputType("text"), withID("strict_login_field"), withLabel("Login"), withPlaceholder("Login/Email")),
		el(95, "/html/body/div[53]/input[1]", withTag("input"), withInputType("search"), withID("strict_search_input"), withAriaLabel("Search"), withPlaceholder("Type to search...")),
		el(96, "/html/body/div[54]/input[1]", withTag("input"), withInputType("email"), withID("strict_email_value"), withLabel("Profile Email"), withValue("captain@manul.com")),
		el(97, "/html/body/div[55]/textarea[1]", withTag("textarea"), withID("strict_note_area"), withLabel("Notes"), withValue("treasure map")),
		el(98, "/html/body/table[2]/tr[1]/td[2]", withTag("td"), withID("norm_price_table"), withText("$299")),

		// ── Real World elements ───────────────────────────────
		el(99, "/html/body/button[1]", withTag("button"), withID("rw_tw_btn"), withText("Deploy Application"),
			withClassName("bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded")),
		el(100, "/html/body/button[2]", withTag("button"), withID("rw_svg_profile"), withAriaLabel("User Profile")),
		el(101, "/html/body/button[3]", withTag("button"), withID("rw_sr_bell"), withText("View Notifications"), withAccessibleName("View Notifications")),
		el(102, "/html/body/div[56]/div[1]", withTag("div"), withRole("switch"), withID("rw_custom_switch"), withAriaLabel("Dark Mode")),
		el(103, "/html/body/div[57]/div[1]", withTag("div"), withID("rw_wysiwyg"), withEditable(), withAriaLabel("Message Body"),
			func(e *dom.ElementSnapshot) { e.Tag = "div" }),
		el(104, "/html/body/label[1]", withTag("label"), withID("rw_file_label"), withDataQA("upload-resume"), withText("Upload Resume")),
		el(105, "/html/body/table[3]/tr[1]/td[2]/button[1]", withTag("button"), withID("rw_edit_profile"), withDataTestID("edit-user-btn"), withText("Edit Profile")),
		el(106, "/html/body/div[58]/button[1]", withTag("button"), withID("rw_modal_close"), withAriaLabel("Close dialog"), withText("✖")),
		el(107, "/html/body/button[4]", withTag("button"), withID("rw_hamburger"), withAriaLabel("Open Navigation"), withText("☰")),
		el(108, "/html/body/button[5]", withTag("button"), withID("rw_google_btn"), withText("Continue with Google"), withClassName("social-login")),
		el(109, "/html/body/a[1]", withTag("a"), withID("rw_terms_link"), withText("Terms of Service")),
		el(110, "/html/body/button[6]", withTag("button"), withID("rw_next_step"), withText("Next: Shipping Details →")),
		el(111, "/html/body/div[59]/div[1]", withTag("div"), withRole("radio"), withID("rw_star_1"), withAriaLabel("1 star"), withText("⭐")),
		el(112, "/html/body/div[59]/div[2]", withTag("div"), withRole("radio"), withID("rw_star_5"), withAriaLabel("5 stars"), withText("⭐⭐⭐⭐⭐")),
		el(113, "/html/body/button[7]", withTag("button"), withID("rw_load_more"), withText("Load More Articles"), withClassName("btn-ghost")),
		el(114, "/html/body/div[60]", withTag("div"), withID("rw_error_msg"), withText("Username is already taken."), withClassName("text-red-500")),
		el(115, "/html/body/button[8]", withTag("button"), withID("rw_fab_create"), withTitle("Create New Post"), withText("+")),
		el(116, "/html/body/button[9]", withTag("button"), withID("rw_complex_btn"), withText("Submit Order")),
		el(117, "/html/body/div[61]/span[1]", withTag("span"), withID("rw_cart_count"), withText("3"), withClassName("badge")),
		el(118, "/html/body/div[62]/button[1]", withTag("button"), withID("rw_play_btn"), withAriaLabel("Play Video"), withText("▶️")),
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Main test suite
// ═══════════════════════════════════════════════════════════════════════════════

func TestMonsterDOM(t *testing.T) {
	elements := monsterDOM()

	tests := []struct {
		name, query, typeHint, mode, expectedID string
	}{
		// ── Original traps (1-24) ────────────────────────────
		{"01_SuggessionClass", "Suggession Class", "", "input", "trap_legend_input"},
		{"03_SubmitLogin", "Submit Login", "button", "clickable", "trap_real_btn"},
		{"04_CyberPassword", "Cyber Password", "", "input", "trap_shadow_input"},
		{"05_CloseWindow", "Close Window", "button", "clickable", "trap_aria_btn"},
		{"06_Save", "Save", "", "clickable", "trap_btn_exact"},
		{"07_AcceptTerms", "Accept Terms", "checkbox", "clickable", "trap_opacity_chk"},
		{"08_SecretToken", "Secret Token", "", "input", "trap_placeholder_input"},
		{"09_RadioNo", "No", "", "clickable", "trap_radio_no"},
		{"10_RememberMe", "Remember Me", "checkbox", "clickable", "trap_role_chk"},
		{"11_ConfirmOrder", "Confirm Order", "", "clickable", "trap_text_btn"},
		{"12_RegisterPortalLink", "Register Portal", "link", "clickable", "trap_link_login"},
		{"15_Submit", "Submit", "", "clickable", "trap_enabled_btn"},
		{"16_QuantityInput", "Quantity", "", "input", "trap_qty_input"},
		{"17_NewsletterChk", "Newsletter", "checkbox", "clickable", "trap_newsletter_chk"},
		{"19_PromoCode", "Promo Code", "", "input", "trap_readonly_input"},
		{"20_Settings", "Settings", "", "clickable", "trap_title_btn"},
		{"21_DownloadBtn", "Download", "button", "clickable", "trap_download_btn"},
		{"22_Password", "password", "", "input", "trap_pw_pass"},
		{"23_CardNumber", "Card Number", "", "input", "trap_float_input"},
		{"24_LaptopChk", "Laptop", "checkbox", "clickable", "trap_chk_laptop"},

		// ── Test 27: Optional promo (exists, should find it) ─
		{"27_OptionalPromo", "Promotion Code if exists", "", "input", "trap_promo_optional_input"},

		// ── Integration bugs (29-34) ─────────────────────────
		{"29_AgreeToTerms", "Agree to Terms", "checkbox", "clickable", "trap_check_agree_chk"},
		{"30_AutoRenew", "Auto-Renew", "checkbox", "clickable", "trap_uncheck_renew_chk"},
		{"33_ShippingAddress", "Shipping Address", "", "input", "trap_dqa_ship"},
		{"34_EnableNotifications", "Enable Notifications", "checkbox", "clickable", "trap_jsclick_chk"},

		// ── DemoQA/Mega (35-46) ──────────────────────────────
		{"35_AddressTextarea", "Address", "", "input", "trap_addr_textarea"},
		{"36_ClickMe", "Click Me", "", "clickable", "trap_singleclick_btn"},
		{"37_StartDate", "Start Date", "", "input", "trap_date_input"},
		{"38_SearchArticles", "Search Articles", "", "input", "trap_search_input"},
		{"39_Page3", "3", "", "clickable", "trap_page_3"},
		{"40_Wednesday", "Wednesday", "checkbox", "clickable", "trap_day_wed"},
		{"42_DoubleClickMe", "Double Click Me", "", "clickable", "trap_dblclick_btn"},
		{"43_MouseHover", "Mouse Hover", "", "hover", "trap_hover_btn"},
		{"45_AcceptMarketing", "Accept Marketing", "checkbox", "clickable", "trap_toggle_chk"},

		// ── Normal elements (47-56) ──────────────────────────
		{"47_FullName", "Full Name", "", "input", "norm_fullname"},
		{"48_WorkEmail", "Work Email", "", "input", "norm_email"},
		{"49_APIToken", "API Token", "", "input", "norm_token"},
		{"50_Comment", "Comment", "", "input", "norm_comment"},
		{"51_SendMessage", "Send Message", "", "clickable", "norm_submit_btn"},
		{"52_AboutUs", "About Us", "", "clickable", "norm_about_link"},
		{"53_CouponCode", "Coupon Code", "", "input", "norm_readonly"},
		{"54_Username", "Username", "", "input", "norm_login_user"},
		{"55_Female", "Female", "", "clickable", "norm_radio_female"},
		{"56_IAgree", "I Agree", "checkbox", "clickable", "norm_agree_chk"},

		// ── Real-world (61-80) ───────────────────────────────
		{"61_DeployApp", "Deploy Application", "", "clickable", "rw_tw_btn"},
		{"62_UserProfile", "User Profile", "", "clickable", "rw_svg_profile"},
		{"63_ViewNotifications", "View Notifications", "", "clickable", "rw_sr_bell"},
		{"64_DarkMode", "Dark Mode", "", "clickable", "rw_custom_switch"},
		{"65_MessageBody", "Message Body", "", "input", "rw_wysiwyg"},
		{"66_UploadResume", "Upload Resume", "", "clickable", "rw_file_label"},
		{"67_EditProfile", "Edit Profile", "", "clickable", "rw_edit_profile"},
		{"69_CloseDialog", "Close dialog", "", "clickable", "rw_modal_close"},
		{"70_OpenNavigation", "Open Navigation", "", "clickable", "rw_hamburger"},
		{"71_ContinueWithGoogle", "Continue with Google", "", "clickable", "rw_google_btn"},
		{"72_TermsOfService", "Terms of Service", "", "clickable", "rw_terms_link"},
		{"73_NextShippingDetails", "Next: Shipping Details", "", "clickable", "rw_next_step"},
		{"74_5Stars", "5 stars", "", "clickable", "rw_star_5"},
		{"75_LoadMoreArticles", "Load More Articles", "", "clickable", "rw_load_more"},
		{"77_CreateNewPost", "Create New Post", "", "clickable", "rw_fab_create"},
		{"78_SubmitOrder", "Submit Order", "", "clickable", "rw_complex_btn"},
		{"80_PlayVideo", "Play Video", "", "clickable", "rw_play_btn"},
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
// Select-mode tests
// ═══════════════════════════════════════════════════════════════════════════════

func TestMonsterDOM_Select(t *testing.T) {
	elements := monsterDOM()

	tests := []struct {
		name, query, expectedID string
	}{
		{"02_Dropdown", "Dropdown", "trap_phantom_select"},
		{"31_Priority", "Priority", "trap_priority_select"},
		{"41_Country", "Country", "trap_country_select"},
		{"58_FavoriteColor", "Color", "norm_color_select"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := rankFirstID(t, tc.query, "", "select", elements)
			if got != tc.expectedID {
				t.Errorf("expected %s, got %s", tc.expectedID, got)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Section disambiguation with AnchorContext
// ═══════════════════════════════════════════════════════════════════════════════

func TestMonsterDOM_LoginFormEmail(t *testing.T) {
	elements := monsterDOM()

	loginAnchor := &scorer.AnchorContext{
		XPath: "/html/body/section[@id='login-form-section']/h3",
		Rect:  dom.Rect{Top: 190, Left: 50, Width: 200, Height: 25},
		Words: []string{"login", "form"},
	}

	ranked := scorer.Rank("Email", "", "input", elements, 10, loginAnchor)
	if len(ranked) == 0 {
		t.Fatal("Rank returned 0 candidates")
	}
	if ranked[0].Element.HTMLId != "trap_section_login" {
		t.Errorf("expected trap_section_login, got %s", ranked[0].Element.HTMLId)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Delete button disambiguation by data-qa
// ═══════════════════════════════════════════════════════════════════════════════

func TestMonsterDOM_DeleteSelected(t *testing.T) {
	elements := monsterDOM()

	// "Delete" for the selected item should match data-qa="delete-selected"
	// since both buttons have identical visible text "Delete".
	// The Python test expects "delete-selected" via context hint.
	anchor := &scorer.AnchorContext{
		Words: []string{"selected"},
		XPath: "/html/body/div[15]",
		Rect:  dom.Rect{Top: 100, Left: 50, Width: 200, Height: 25},
	}
	ranked := scorer.Rank("Delete", "", "clickable", elements, 10, anchor)
	if len(ranked) == 0 {
		t.Fatal("Rank returned 0 candidates")
	}
	if ranked[0].Element.HTMLId != "trap_delete_selected" {
		t.Errorf("expected trap_delete_selected, got %s", ranked[0].Element.HTMLId)
	}
}

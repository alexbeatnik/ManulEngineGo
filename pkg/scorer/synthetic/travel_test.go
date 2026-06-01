package synthetic

// ─────────────────────────────────────────────────────────────────────────────
// TRAVEL DOM SCORING TEST SUITE
//
// Port of ManulEngine test_04_travel.py — 100-element travel booking page.
// Validates: flight search, calendar, passengers, sidebar filters, flight cards,
// seat map, add-ons, passenger form, hotel search, checkout.
// Skipped: extract (14,23,26,41,44,61,89,91,92,93), verify (18,50,52,60),
//          optional (30).
// ─────────────────────────────────────────────────────────────────────────────

import (
	"testing"

	"github.com/alexbeatnik/ManulHeart/pkg/dom"
)

func travelDOM() []dom.ElementSnapshot {
	return []dom.ElementSnapshot{
		// Flight Search (t1-t10)
		el(1, "/html/body/div[1]/input[1]", withTag("input"), withInputType("radio"), withID("t1"), withLabel("Round Trip"), withNameAttr("trip")),
		el(2, "/html/body/div[1]/input[2]", withTag("input"), withInputType("radio"), withID("t2"), withLabel("One Way"), withNameAttr("trip")),
		el(3, "/html/body/div[1]/input[3]", withTag("input"), withInputType("radio"), withID("t3"), withLabel("Multi-city"), withNameAttr("trip")),
		el(4, "/html/body/input[1]", withTag("input"), withInputType("text"), withID("t4"), withPlaceholder("Flying from")),
		el(5, "/html/body/button[1]", withTag("button"), withID("t5"), withAriaLabel("Swap Origin and Destination"), withText("⇄")),
		el(6, "/html/body/input[2]", withTag("input"), withInputType("text"), withID("t6"), withPlaceholder("Flying to")),
		el(7, "/html/body/input[3]", withTag("input"), withInputType("checkbox"), withID("t7"), withLabel("Direct flights only")),
		el(8, "/html/body/input[4]", withTag("input"), withInputType("checkbox"), withID("t8"), withLabel("Add nearby airports")),
		el(9, "/html/body/button[2]", withTag("button"), withID("t9"), withClassName("btn-search"), withText("Search Flights")),
		el(10, "/html/body/a[1]", withTag("a"), withID("t10"), withText("Explore Deals")),

		// Calendar (t11-t20)
		el(11, "/html/body/input[5]", withTag("input"), withInputType("text"), withID("t11"), withPlaceholder("Depart")),
		el(12, "/html/body/input[6]", withTag("input"), withInputType("text"), withID("t12"), withPlaceholder("Return")),
		el(13, "/html/body/button[3]", withTag("button"), withID("t13"), withAriaLabel("Previous Month"), withText("◀")),
		el(14, "/html/body/span[1]", withTag("span"), withID("t14"), withDataQA("current-month"), withText("March 2026")),
		el(15, "/html/body/button[4]", withTag("button"), withID("t15"), withAriaLabel("Next Month"), withText("▶")),
		el(16, "/html/body/div[2]", withTag("div"), withRole("button"), withID("t16"), withAriaLabel("March 10, 2026"), withText("10")),
		el(17, "/html/body/div[3]", withTag("div"), withRole("button"), withID("t17"), withAriaLabel("March 15, 2026"), withText("15")),
		el(18, "/html/body/div[4]", withTag("div"), withRole("button"), withID("t18"), withAriaLabel("March 20, 2026"), withClassName("disabled"), withText("20")),
		el(19, "/html/body/button[5]", withTag("button"), withID("t19"), withText("I'm flexible (+/- 3 days)")),
		el(20, "/html/body/button[6]", withTag("button"), withID("t20"), withText("Done (Dates)")),

		// Passengers (t21-t30)
		el(21, "/html/body/div[5]", withTag("div"), withRole("button"), withID("t21"), withText("1 Adult, Economy ▼")),
		el(22, "/html/body/button[7]", withTag("button"), withID("t22"), withAriaLabel("Decrease Adults"), withText("-")),
		el(23, "/html/body/span[2]", withTag("span"), withID("t23"), withDataQA("adults"), withText("1")),
		el(24, "/html/body/button[8]", withTag("button"), withID("t24"), withAriaLabel("Increase Adults"), withText("+")),
		el(25, "/html/body/button[9]", withTag("button"), withID("t25"), withAriaLabel("Decrease Children"), withText("-")),
		el(26, "/html/body/span[3]", withTag("span"), withID("t26"), withDataQA("children"), withText("0")),
		el(27, "/html/body/button[10]", withTag("button"), withID("t27"), withAriaLabel("Increase Children"), withText("+")),
		el(28, "/html/body/select[1]", withTag("select"), withID("t28"), withAriaLabel("Cabin Class"), withText("Economy")),
		el(29, "/html/body/button[11]", withTag("button"), withID("t29"), withText("Apply Pax")),
		el(30, "/html/body/button[12]", withTag("button"), withID("t30"), withHidden(), withText("Add Infant")),

		// Sidebar Filters (t31-t40)
		el(31, "/html/body/input[7]", withTag("input"), withInputType("checkbox"), withID("t31"), withLabel("Non-stop")),
		el(32, "/html/body/input[8]", withTag("input"), withInputType("checkbox"), withID("t32"), withLabel("1 Stop")),
		el(33, "/html/body/input[9]", withTag("input"), withInputType("checkbox"), withID("t33"), withLabel("2+ Stops")),
		el(34, "/html/body/button[13]", withTag("button"), withID("t34"), withText("Clear All Airlines")),
		el(35, "/html/body/input[10]", withTag("input"), withInputType("checkbox"), withID("t35"), withLabel("Lufthansa")),
		el(36, "/html/body/input[11]", withTag("input"), withInputType("checkbox"), withID("t36"), withLabel("Ryanair")),
		el(37, "/html/body/input[12]", withTag("input"), withInputType("range"), withID("t37"), withAriaLabel("Departure Time")),
		el(38, "/html/body/select[2]", withTag("select"), withID("t38"), withAriaLabel("Sort by"), withText("Price (Lowest)")),
		el(39, "/html/body/button[14]", withTag("button"), withID("t39"), withText("Reset Filters")),
		el(40, "/html/body/div[6]", withTag("div"), withRole("switch"), withID("t40"), withText("Hide overnight flights")),

		// Flight Cards (t41-t50)
		el(41, "/html/body/div[7]", withTag("div"), withID("t41"), withClassName("price"), withText("€129.00")),
		el(42, "/html/body/button[15]", withTag("button"), withID("t42"), withAriaLabel("Select Lufthansa flight"), withText("Select")),
		el(43, "/html/body/button[16]", withTag("button"), withID("t43"), withText("View Details")),
		el(44, "/html/body/div[8]", withTag("div"), withID("t44"), withClassName("price"), withText("€19.99")),
		el(45, "/html/body/button[17]", withTag("button"), withID("t45"), withAriaLabel("Select Ryanair flight"), withText("Select")),
		el(46, "/html/body/button[18]", withTag("button"), withID("t46"), withText("Show more flights")),
		el(47, "/html/body/button[19]", withTag("button"), withID("t47"), withText("Track Prices")),
		el(48, "/html/body/input[13]", withTag("input"), withInputType("email"), withID("t48"), withPlaceholder("Email for price alerts")),
		el(49, "/html/body/button[20]", withTag("button"), withID("t49"), withText("Set Alert")),
		el(50, "/html/body/div[9]", withTag("div"), withID("t50"), withText("No flights found for selected dates")),

		// Seat Map (t51-t60)
		el(51, "/html/body/div[10]", withTag("div"), withRole("checkbox"), withID("t51"), withAriaLabel("Seat 1A"), withClassName("seat seat-avail"), withText("1A")),
		el(52, "/html/body/div[11]", withTag("div"), withRole("checkbox"), withID("t52"), withAriaLabel("Seat 1B"), withClassName("seat seat-taken"), withText("1B")),
		el(53, "/html/body/div[12]", withTag("div"), withRole("checkbox"), withID("t53"), withAriaLabel("Seat 1C"), withClassName("seat seat-avail"), withText("1C")),
		el(54, "/html/body/button[21]", withTag("button"), withID("t54"), withText("Skip Seat Selection")),
		el(55, "/html/body/button[22]", withTag("button"), withID("t55"), withText("Confirm Seats")),
		el(56, "/html/body/div[13]", withTag("div"), withRole("button"), withID("t56"), withText("Extra Legroom (+$20)")),
		el(57, "/html/body/div[14]", withTag("div"), withRole("button"), withID("t57"), withText("Window Seat")),
		el(58, "/html/body/div[15]", withTag("div"), withRole("button"), withID("t58"), withText("Aisle Seat")),
		el(59, "/html/body/button[23]", withTag("button"), withID("t59"), withText("View upper deck")),
		el(60, "/html/body/button[24]", withTag("button"), withID("t60"), withDisabled(), withText("Next Passenger")),

		// Add-ons (t61-t70)
		el(61, "/html/body/span[4]", withTag("span"), withID("t61"), withText("Included")),
		el(62, "/html/body/button[25]", withTag("button"), withID("t62"), withText("Add Cabin Bag (+€15)")),
		el(63, "/html/body/select[3]", withTag("select"), withID("t63"), withAriaLabel("Checked Bags"), withText("0")),
		el(64, "/html/body/input[14]", withTag("input"), withInputType("checkbox"), withID("t64"), withLabel("Priority Boarding")),
		el(65, "/html/body/input[15]", withTag("input"), withInputType("checkbox"), withID("t65"), withLabel("Fast Track Security")),
		el(66, "/html/body/input[16]", withTag("input"), withInputType("radio"), withID("t66"), withLabel("Yes, protect my trip"), withNameAttr("ins")),
		el(67, "/html/body/input[17]", withTag("input"), withInputType("radio"), withID("t67"), withLabel("No, I will risk it"), withNameAttr("ins")),
		el(68, "/html/body/button[26]", withTag("button"), withID("t68"), withText("Read Policy")),
		el(69, "/html/body/button[27]", withTag("button"), withID("t69"), withText("Add Rental Car")),
		el(70, "/html/body/button[28]", withTag("button"), withID("t70"), withText("Continue to Passenger Details")),

		// Passenger Form (t71-t80)
		el(71, "/html/body/select[4]", withTag("select"), withID("t71"), withAriaLabel("Title"), withText("Mr")),
		el(72, "/html/body/input[18]", withTag("input"), withInputType("text"), withID("t72"), withPlaceholder("First Name (as in passport)")),
		el(73, "/html/body/input[19]", withTag("input"), withInputType("text"), withID("t73"), withPlaceholder("Last Name (as in passport)")),
		el(74, "/html/body/input[20]", withTag("input"), withInputType("date"), withID("t74"), withAriaLabel("Date of Birth")),
		el(75, "/html/body/select[5]", withTag("select"), withID("t75"), withAriaLabel("Nationality"), withText("Ukraine")),
		el(76, "/html/body/input[21]", withTag("input"), withInputType("text"), withID("t76"), withPlaceholder("Passport Number")),
		el(77, "/html/body/input[22]", withTag("input"), withInputType("date"), withID("t77"), withAriaLabel("Passport Expiry")),
		el(78, "/html/body/input[23]", withTag("input"), withInputType("email"), withID("t78"), withPlaceholder("Contact Email")),
		el(79, "/html/body/input[24]", withTag("input"), withInputType("tel"), withID("t79"), withPlaceholder("Mobile Number")),
		el(80, "/html/body/button[29]", withTag("button"), withID("t80"), withText("Save Passenger")),

		// Hotel Search (t81-t90)
		el(81, "/html/body/input[25]", withTag("input"), withInputType("text"), withID("t81"), withPlaceholder("Where are you going?")),
		el(82, "/html/body/button[30]", withTag("button"), withID("t82"), withText("Search Hotels")),
		el(83, "/html/body/input[26]", withTag("input"), withInputType("checkbox"), withID("t83"), withLabel("I'm traveling for work")),
		el(84, "/html/body/button[31]", withTag("button"), withID("t84"), withAriaLabel("Show Map"), withText("🗺️ Map View")),
		el(85, "/html/body/select[6]", withTag("select"), withID("t85"), withAriaLabel("Star Rating"), withText("Any")),
		el(86, "/html/body/input[27]", withTag("input"), withInputType("checkbox"), withID("t86"), withLabel("Free Cancellation")),
		el(87, "/html/body/input[28]", withTag("input"), withInputType("checkbox"), withID("t87"), withLabel("Breakfast Included")),
		el(88, "/html/body/button[32]", withTag("button"), withID("t88"), withAriaLabel("Select Hilton Hotel"), withText("Book Hilton")),
		el(89, "/html/body/span[5]", withTag("span"), withID("t89"), withClassName("hotel-price"), withText("$150/night")),
		el(90, "/html/body/button[33]", withTag("button"), withID("t90"), withText("Read Guest Reviews")),

		// Checkout (t91-t100)
		el(91, "/html/body/span[6]", withTag("span"), withID("t91"), withText("$250.00")),
		el(92, "/html/body/span[7]", withTag("span"), withID("t92"), withText("$45.50")),
		el(93, "/html/body/span[8]", withTag("span"), withID("t93"), withText("$295.50")),
		el(94, "/html/body/input[29]", withTag("input"), withInputType("text"), withID("t94"), withPlaceholder("Promo / Voucher Code")),
		el(95, "/html/body/button[34]", withTag("button"), withID("t95"), withText("Apply Voucher")),
		el(96, "/html/body/input[30]", withTag("input"), withInputType("checkbox"), withID("t96"), withLabel("I accept the Terms & Conditions")),
		el(97, "/html/body/input[31]", withTag("input"), withInputType("checkbox"), withID("t97"), withLabel("I acknowledge the Hazmat Policy")),
		el(98, "/html/body/button[35]", withTag("button"), withID("t98"), withClassName("btn-pay"), withText("Pay Now")),
		el(99, "/html/body/button[36]", withTag("button"), withID("t99"), withText("Save Cart")),
		el(100, "/html/body/a[2]", withTag("a"), withID("t100"), withText("Back to Home")),
	}
}

func TestTravel(t *testing.T) {
	elements := travelDOM()

	tests := []struct {
		name, query, mode, expectedID string
	}{
		// Flight Search
		{"01_RoundTrip", "Round Trip", "clickable", "t1"},
		{"02_OneWay", "One Way", "clickable", "t2"},
		{"03_MultiCity", "Multi-city", "clickable", "t3"},
		{"04_FlyingFrom", "Flying from", "input", "t4"},
		{"05_SwapOriginDest", "Swap Origin and Destination", "clickable", "t5"},
		{"06_FlyingTo", "Flying to", "input", "t6"},
		{"07_DirectFlightsOnly", "Direct flights only", "clickable", "t7"},
		{"08_AddNearbyAirports", "Add nearby airports", "clickable", "t8"},
		{"09_SearchFlights", "Search Flights", "clickable", "t9"},
		{"10_ExploreDeals", "Explore Deals", "clickable", "t10"},

		// Calendar
		{"11_Depart", "Depart", "clickable", "t11"},
		{"12_Return", "Return", "clickable", "t12"},
		{"13_PreviousMonth", "Previous Month", "clickable", "t13"},
		// 14 = EXTRACT
		{"15_NextMonth", "Next Month", "clickable", "t15"},
		{"16_March10", "March 10, 2026", "clickable", "t16"},
		{"17_March15", "March 15, 2026", "clickable", "t17"},
		// 18 = VERIFY
		{"19_Flexible", "flexible", "clickable", "t19"},
		{"20_DoneDates", "Done (Dates)", "clickable", "t20"},

		// Passengers
		{"21_AdultEconomy", "Adult, Economy", "clickable", "t21"},
		{"22_DecreaseAdults", "Decrease Adults", "clickable", "t22"},
		// 23 = EXTRACT
		{"24_IncreaseAdults", "Increase Adults", "clickable", "t24"},
		{"25_DecreaseChildren", "Decrease Children", "clickable", "t25"},
		// 26 = EXTRACT
		{"27_IncreaseChildren", "Increase Children", "clickable", "t27"},
		{"29_ApplyPax", "Apply Pax", "clickable", "t29"},
		// 30 = optional hidden

		// Filters
		{"31_NonStop", "Non-stop", "clickable", "t31"},
		{"32_OneStop", "1 Stop", "clickable", "t32"},
		{"33_TwoPlusStops", "2+ Stops", "clickable", "t33"},
		{"34_ClearAllAirlines", "Clear All Airlines", "clickable", "t34"},
		{"35_Lufthansa", "Lufthansa", "clickable", "t35"},
		{"36_Ryanair", "Ryanair", "clickable", "t36"},
		{"37_DepartureTime", "Departure Time", "input", "t37"},
		{"39_ResetFilters", "Reset Filters", "clickable", "t39"},
		{"40_HideOvernightFlights", "Hide overnight flights", "clickable", "t40"},

		// Flight Cards
		// 41 = EXTRACT
		{"42_SelectLufthansa", "Select Lufthansa flight", "clickable", "t42"},
		{"43_ViewDetails", "View Details", "clickable", "t43"},
		// 44 = EXTRACT
		{"45_SelectRyanair", "Select Ryanair flight", "clickable", "t45"},
		{"46_ShowMoreFlights", "Show more flights", "clickable", "t46"},
		{"47_TrackPrices", "Track Prices", "clickable", "t47"},
		{"48_EmailPriceAlerts", "Email for price alerts", "input", "t48"},
		{"49_SetAlert", "Set Alert", "clickable", "t49"},
		// 50 = VERIFY

		// Seat Map
		{"51_Seat1A", "Seat 1A", "clickable", "t51"},
		// 52 = VERIFY
		{"53_Seat1C", "Seat 1C", "clickable", "t53"},
		{"54_SkipSeatSelection", "Skip Seat Selection", "clickable", "t54"},
		{"55_ConfirmSeats", "Confirm Seats", "clickable", "t55"},
		{"56_ExtraLegroom", "Extra Legroom", "clickable", "t56"},
		{"57_WindowSeat", "Window Seat", "clickable", "t57"},
		{"58_AisleSeat", "Aisle Seat", "clickable", "t58"},
		{"59_ViewUpperDeck", "View upper deck", "clickable", "t59"},
		// 60 = VERIFY disabled

		// Add-ons
		// 61 = EXTRACT
		{"62_AddCabinBag", "Add Cabin Bag", "clickable", "t62"},
		{"64_PriorityBoarding", "Priority Boarding", "clickable", "t64"},
		{"65_FastTrackSecurity", "Fast Track Security", "clickable", "t65"},
		{"66_ProtectTrip", "Yes, protect my trip", "clickable", "t66"},
		{"67_RiskIt", "No, I will risk it", "clickable", "t67"},
		{"68_ReadPolicy", "Read Policy", "clickable", "t68"},
		{"69_AddRentalCar", "Add Rental Car", "clickable", "t69"},
		{"70_ContinuePassenger", "Continue to Passenger Details", "clickable", "t70"},

		// Passenger Form
		{"72_FirstName", "First Name", "input", "t72"},
		{"73_LastName", "Last Name", "input", "t73"},
		{"74_DateOfBirth", "Date of Birth", "input", "t74"},
		{"76_PassportNumber", "Passport Number", "input", "t76"},
		{"77_PassportExpiry", "Passport Expiry", "input", "t77"},
		{"78_ContactEmail", "Contact Email", "input", "t78"},
		{"79_MobileNumber", "Mobile Number", "input", "t79"},
		{"80_SavePassenger", "Save Passenger", "clickable", "t80"},

		// Hotel Search
		{"81_WhereGoing", "Where are you going?", "input", "t81"},
		{"82_SearchHotels", "Search Hotels", "clickable", "t82"},
		{"83_TravelingForWork", "traveling for work", "clickable", "t83"},
		{"84_MapView", "Map View", "clickable", "t84"},
		{"86_FreeCancellation", "Free Cancellation", "clickable", "t86"},
		{"87_BreakfastIncluded", "Breakfast Included", "clickable", "t87"},
		{"88_BookHilton", "Book Hilton", "clickable", "t88"},
		// 89 = EXTRACT
		{"90_ReadGuestReviews", "Read Guest Reviews", "clickable", "t90"},

		// Checkout
		// 91-93 = EXTRACT
		{"94_VoucherCode", "Voucher Code", "input", "t94"},
		{"95_ApplyVoucher", "Apply Voucher", "clickable", "t95"},
		{"96_TermsConditions", "Terms & Conditions", "clickable", "t96"},
		{"97_HazmatPolicy", "Hazmat Policy", "clickable", "t97"},
		{"98_PayNow", "Pay Now", "clickable", "t98"},
		{"99_SaveCart", "Save Cart", "clickable", "t99"},
		{"100_BackToHome", "Back to Home", "clickable", "t100"},
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

func TestTravel_Select(t *testing.T) {
	elements := travelDOM()

	tests := []struct {
		name, query, expectedID string
	}{
		{"28_CabinClass", "Cabin Class", "t28"},
		{"38_SortBy", "Sort by", "t38"},
		{"63_CheckedBags", "Checked Bags", "t63"},
		{"71_Title", "Title", "t71"},
		{"75_Nationality", "Nationality", "t75"},
		{"85_StarRating", "Star Rating", "t85"},
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

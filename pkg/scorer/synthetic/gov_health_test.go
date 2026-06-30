package synthetic

// ─────────────────────────────────────────────────────────────────────────────
// GOV/HEALTHCARE DOM SCORING TEST SUITE
//
// Port of ManulEngine test_07_gov_health.py — 100-element healthcare/gov page.
// Validates: authentication, demographics, address, medical history, insurance,
// appointments, accessibility, consent/signatures, records, form navigation.
// Skipped: extract (47,89), verify (9,10,71,96,99).
// ─────────────────────────────────────────────────────────────────────────────

import (
	"testing"

	"github.com/alexbeatnik/ManulEngineGo/pkg/dom"
)

func govHealthDOM() []dom.ElementSnapshot {
	return []dom.ElementSnapshot{
		// Authentication (h1-h10)
		el(1, "/html/body/input[1]", withTag("input"), withInputType("password"), withID("h1"), withPlaceholder("XXX-XX-XXXX"), withLabel("Social Security Number")),
		el(2, "/html/body/input[2]", withTag("input"), withInputType("date"), withID("h2"), withLabel("Date of Birth")),
		el(3, "/html/body/button[1]", withTag("button"), withID("h3"), withClassName("btn-primary"), withText("Authenticate")),
		el(4, "/html/body/a[1]", withTag("a"), withID("h4"), withText("Forgot ID?")),
		el(5, "/html/body/select[1]", withTag("select"), withID("h5"), withAriaLabel("ID Type"), withText("State ID")),
		el(6, "/html/body/input[3]", withTag("input"), withInputType("text"), withID("h6"), withPlaceholder("Document Number")),
		el(7, "/html/body/input[4]", withTag("input"), withInputType("checkbox"), withID("h7"), withLabel("I am not a robot")),
		el(8, "/html/body/button[2]", withTag("button"), withID("h8"), withAriaLabel("Audio Captcha"), withText("🔊")),
		el(9, "/html/body/button[3]", withTag("button"), withID("h9"), withDisabled(), withText("Verify Captcha")),
		el(10, "/html/body/div[1]", withTag("div"), withRole("alert"), withID("h10"), withText("Session expires in 04:59")),

		// Demographics (h11-h20)
		el(11, "/html/body/input[5]", withTag("input"), withInputType("text"), withID("h11"), withPlaceholder("First Name")),
		el(12, "/html/body/input[6]", withTag("input"), withInputType("text"), withID("h12"), withPlaceholder("Middle Initial")),
		el(13, "/html/body/input[7]", withTag("input"), withInputType("text"), withID("h13"), withPlaceholder("Last Name")),
		el(14, "/html/body/select[2]", withTag("select"), withID("h14"), withLabel("Suffix"), withText("None")),
		el(15, "/html/body/input[8]", withTag("input"), withInputType("radio"), withID("h15"), withLabel("Male"), withNameAttr("sex")),
		el(16, "/html/body/input[9]", withTag("input"), withInputType("radio"), withID("h16"), withLabel("Female"), withNameAttr("sex")),
		el(17, "/html/body/select[3]", withTag("select"), withID("h17"), withAriaLabel("Marital Status"), withText("Single")),
		el(18, "/html/body/input[10]", withTag("input"), withInputType("text"), withID("h18"), withPlaceholder("Maiden Name (if applicable)")),
		el(19, "/html/body/input[11]", withTag("input"), withInputType("text"), withID("h19"), withPlaceholder("Place of Birth (City)")),
		el(20, "/html/body/button[4]", withTag("button"), withID("h20"), withText("Validate Identity")),

		// Address (h21-h30)
		el(21, "/html/body/input[12]", withTag("input"), withInputType("text"), withID("h21"), withLabel("Address Line 1")),
		el(22, "/html/body/input[13]", withTag("input"), withInputType("text"), withID("h22"), withLabel("Apt/Suite")),
		el(23, "/html/body/input[14]", withTag("input"), withInputType("text"), withID("h23"), withLabel("City")),
		el(24, "/html/body/select[4]", withTag("select"), withID("h24"), withLabel("State"), withText("TX")),
		el(25, "/html/body/input[15]", withTag("input"), withInputType("text"), withID("h25"), withLabel("ZIP Code")),
		el(26, "/html/body/input[16]", withTag("input"), withInputType("text"), withID("h26"), withLabel("County")),
		el(27, "/html/body/input[17]", withTag("input"), withInputType("checkbox"), withID("h27"), withLabel("Mailing address same as residential")),
		el(28, "/html/body/input[18]", withTag("input"), withInputType("tel"), withID("h28"), withPlaceholder("Primary Phone")),
		el(29, "/html/body/input[19]", withTag("input"), withInputType("email"), withID("h29"), withPlaceholder("Email Address")),
		el(30, "/html/body/button[5]", withTag("button"), withID("h30"), withText("Update Contact Info")),

		// Medical History (h31-h40)
		el(31, "/html/body/input[20]", withTag("input"), withInputType("checkbox"), withID("h31"), withLabel("Diabetes")),
		el(32, "/html/body/input[21]", withTag("input"), withInputType("checkbox"), withID("h32"), withLabel("Hypertension")),
		el(33, "/html/body/input[22]", withTag("input"), withInputType("checkbox"), withID("h33"), withLabel("Asthma")),
		el(34, "/html/body/input[23]", withTag("input"), withInputType("checkbox"), withID("h34"), withClassName("none-of-above"), withLabel("None of the above")),
		el(35, "/html/body/textarea[1]", withTag("textarea"), withID("h35"), withLabel("List any surgeries:"), withEditable()),
		el(36, "/html/body/input[24]", withTag("input"), withInputType("text"), withID("h36"), withLabel("Current Medications:")),
		el(37, "/html/body/button[6]", withTag("button"), withID("h37"), withText("Add Medication")),
		el(38, "/html/body/input[25]", withTag("input"), withInputType("text"), withID("h38"), withLabel("Allergies:")),
		el(39, "/html/body/button[7]", withTag("button"), withID("h39"), withText("Add Allergy")),
		el(40, "/html/body/input[26]", withTag("input"), withInputType("checkbox"), withID("h40"), withLabel("No known allergies")),

		// Insurance (h41-h50)
		el(41, "/html/body/select[5]", withTag("select"), withID("h41"), withAriaLabel("Insurance Provider"), withText("Medicare")),
		el(42, "/html/body/input[27]", withTag("input"), withInputType("text"), withID("h42"), withPlaceholder("Policy Number")),
		el(43, "/html/body/input[28]", withTag("input"), withInputType("text"), withID("h43"), withPlaceholder("Group ID")),
		el(44, "/html/body/input[29]", withTag("input"), withInputType("file"), withID("h44"), withLabel("Upload Insurance Card (Front)")),
		el(45, "/html/body/input[30]", withTag("input"), withInputType("file"), withID("h45"), withLabel("Upload Insurance Card (Back)")),
		el(46, "/html/body/button[8]", withTag("button"), withID("h46"), withText("Verify Coverage")),
		el(47, "/html/body/span[1]", withTag("span"), withID("h47"), withText("Status: Active")),
		el(48, "/html/body/input[31]", withTag("input"), withInputType("text"), withID("h48"), withPlaceholder("Primary Care Physician")),
		el(49, "/html/body/button[9]", withTag("button"), withID("h49"), withText("Search Providers")),
		el(50, "/html/body/input[32]", withTag("input"), withInputType("checkbox"), withID("h50"), withLabel("I am the primary policyholder")),

		// Appointments (h51-h60)
		el(51, "/html/body/button[10]", withTag("button"), withID("h51"), withClassName("btn-schedule"), withText("Schedule New Appointment")),
		el(52, "/html/body/select[6]", withTag("select"), withID("h52"), withAriaLabel("Reason for Visit"), withText("Annual Physical")),
		el(53, "/html/body/input[33]", withTag("input"), withInputType("date"), withID("h53"), withAriaLabel("Preferred Date")),
		el(54, "/html/body/select[7]", withTag("select"), withID("h54"), withAriaLabel("Preferred Time"), withText("Morning")),
		el(55, "/html/body/button[11]", withTag("button"), withID("h55"), withText("Find Available Slots")),
		el(56, "/html/body/div[2]", withTag("div"), withRole("button"), withID("h56"), withClassName("time-slot"), withText("09:00 AM")),
		el(57, "/html/body/div[3]", withTag("div"), withRole("button"), withID("h57"), withClassName("time-slot"), withText("10:30 AM")),
		el(58, "/html/body/button[12]", withTag("button"), withID("h58"), withText("Confirm Appointment")),
		el(59, "/html/body/button[13]", withTag("button"), withID("h59"), withText("Cancel Appointment")),
		el(60, "/html/body/button[14]", withTag("button"), withID("h60"), withText("Reschedule")),

		// Accessibility (h61-h70)
		el(61, "/html/body/button[15]", withTag("button"), withID("h61"), withAriaLabel("Toggle High Contrast"), withText("🌓")),
		el(62, "/html/body/button[16]", withTag("button"), withID("h62"), withAriaLabel("Increase Text Size"), withText("A+")),
		el(63, "/html/body/button[17]", withTag("button"), withID("h63"), withAriaLabel("Decrease Text Size"), withText("A-")),
		el(64, "/html/body/select[8]", withTag("select"), withID("h64"), withAriaLabel("Language Translation"), withText("English")),
		el(65, "/html/body/a[2]", withTag("a"), withID("h65"), withAriaLabel("Skip to main content"), withText("Skip to main content")),
		el(66, "/html/body/button[18]", withTag("button"), withID("h66"), withText("Print Page")),
		el(67, "/html/body/button[19]", withTag("button"), withID("h67"), withText("Download PDF")),
		el(68, "/html/body/button[20]", withTag("button"), withID("h68"), withText("Chat with Virtual Assistant")),
		el(69, "/html/body/button[21]", withTag("button"), withID("h69"), withText("Contact Live Agent")),
		el(70, "/html/body/button[22]", withTag("button"), withID("h70"), withText("Leave Feedback")),

		// Consent & Signatures (h71-h80)
		el(71, "/html/body/textarea[2]", withTag("textarea"), withID("h71"), withText("HIPAA Privacy Notice...")),
		el(72, "/html/body/input[34]", withTag("input"), withInputType("checkbox"), withID("h72"), withLabel("I acknowledge receipt of the Privacy Notice")),
		el(73, "/html/body/input[35]", withTag("input"), withInputType("checkbox"), withID("h73"), withLabel("I consent to telehealth services")),
		el(74, "/html/body/div[4]", withTag("div"), withRole("application"), withID("h74"), withClassName("signature-pad"), withAriaLabel("Sign here")),
		el(75, "/html/body/button[23]", withTag("button"), withID("h75"), withText("Clear Signature")),
		el(76, "/html/body/input[36]", withTag("input"), withInputType("text"), withID("h76"), withPlaceholder("Type name to sign")),
		el(77, "/html/body/input[37]", withTag("input"), withInputType("date"), withID("h77"), withAriaLabel("Signature Date")),
		el(78, "/html/body/input[38]", withTag("input"), withInputType("text"), withID("h78"), withPlaceholder("Relationship to patient")),
		el(79, "/html/body/button[24]", withTag("button"), withID("h79"), withText("Accept & Sign")),
		el(80, "/html/body/button[25]", withTag("button"), withID("h80"), withText("Decline")),

		// Records (h81-h90)
		el(81, "/html/body/button[26]", withTag("button"), withID("h81"), withText("Download W-2")),
		el(82, "/html/body/button[27]", withTag("button"), withID("h82"), withText("View Record")),
		el(83, "/html/body/a[3]", withTag("a"), withID("h83"), withText("Open PDF")),
		el(84, "/html/body/button[28]", withTag("button"), withID("h84"), withText("Request Medical Records")),
		el(85, "/html/body/button[29]", withTag("button"), withID("h85"), withText("Appeal Tax Decision")),
		el(86, "/html/body/input[39]", withTag("input"), withInputType("text"), withID("h86"), withPlaceholder("Search records by year")),
		el(87, "/html/body/button[30]", withTag("button"), withID("h87"), withText("Search Records")),
		el(88, "/html/body/select[9]", withTag("select"), withID("h88"), withAriaLabel("Filter Records"), withText("All")),
		el(89, "/html/body/div[5]", withTag("div"), withID("h89"), withText("Total documents: 14")),
		el(90, "/html/body/button[31]", withTag("button"), withID("h90"), withText("Upload New Document")),

		// Form Navigation (h91-h100)
		el(91, "/html/body/button[32]", withTag("button"), withID("h91"), withText("Save Draft")),
		el(92, "/html/body/button[33]", withTag("button"), withID("h92"), withText("Next Section >>")),
		el(93, "/html/body/button[34]", withTag("button"), withID("h93"), withText("<< Previous Section")),
		el(94, "/html/body/input[40]", withTag("input"), withInputType("submit"), withID("h94"), withValue("SUBMIT APPLICATION"), withClassName("btn-success")),
		el(95, "/html/body/input[41]", withTag("input"), withInputType("reset"), withID("h95"), withValue("CLEAR FORM"), withClassName("btn-danger")),
		el(96, "/html/body/div[6]", withTag("div"), withID("h96"), withHidden(), withText("Are you sure you want to submit?")),
		el(97, "/html/body/button[35]", withTag("button"), withID("h97"), withText("Yes, Submit")),
		el(98, "/html/body/button[36]", withTag("button"), withID("h98"), withText("No, Go Back")),
		el(99, "/html/body/button[37]", withTag("button"), withID("h99"), withDisabled(), withText("Processing...")),
		el(100, "/html/body/a[4]", withTag("a"), withID("h100"), withText("Secure Logout")),
	}
}

func TestGovHealth(t *testing.T) {
	elements := govHealthDOM()

	tests := []struct {
		name, query, mode, expectedID string
	}{
		// Authentication
		{"01_SSN", "Social Security Number", "input", "h1"},
		{"02_DOB", "Date of Birth", "input", "h2"},
		{"03_Authenticate", "Authenticate", "clickable", "h3"},
		{"04_ForgotID", "Forgot ID?", "clickable", "h4"},
		// test 5 → select
		{"06_DocumentNumber", "Document Number", "input", "h6"},
		{"07_NotARobot", "I am not a robot", "clickable", "h7"},
		{"08_AudioCaptcha", "Audio Captcha", "clickable", "h8"},
		// test 9 skipped — verify (disabled)
		// test 10 skipped — verify

		// Demographics
		{"11_FirstName", "First Name", "input", "h11"},
		{"12_MiddleInitial", "Middle Initial", "input", "h12"},
		{"13_LastName", "Last Name", "input", "h13"},
		// test 14 → select
		{"15_Male", "Male", "clickable", "h15"},
		{"16_Female", "Female", "clickable", "h16"},
		// test 17 → select
		{"18_MaidenName", "Maiden Name", "input", "h18"},
		{"19_PlaceOfBirth", "Place of Birth", "input", "h19"},
		{"20_ValidateIdentity", "Validate Identity", "clickable", "h20"},

		// Address
		{"21_AddressLine1", "Address Line 1", "input", "h21"},
		{"22_AptSuite", "Apt/Suite", "input", "h22"},
		{"23_City", "City", "input", "h23"},
		// test 24 → select
		{"25_ZIP", "ZIP Code", "input", "h25"},
		{"26_County", "County", "input", "h26"},
		{"27_MailingSame", "Mailing address same as residential", "clickable", "h27"},
		{"28_PrimaryPhone", "Primary Phone", "input", "h28"},
		{"29_EmailAddress", "Email Address", "input", "h29"},
		{"30_UpdateContact", "Update Contact Info", "clickable", "h30"},

		// Medical History
		{"31_Diabetes", "Diabetes", "clickable", "h31"},
		{"32_Hypertension", "Hypertension", "clickable", "h32"},
		{"33_Asthma", "Asthma", "clickable", "h33"},
		{"34_NoneOfAbove", "None of the above", "clickable", "h34"},
		{"35_Surgeries", "surgeries", "input", "h35"},
		{"36_Medications", "Current Medications", "input", "h36"},
		{"37_AddMedication", "Add Medication", "clickable", "h37"},
		{"38_Allergies", "Allergies", "input", "h38"},
		{"39_AddAllergy", "Add Allergy", "clickable", "h39"},
		{"40_NoAllergies", "No known allergies", "clickable", "h40"},

		// Insurance
		// test 41 → select
		{"42_PolicyNumber", "Policy Number", "input", "h42"},
		{"43_GroupID", "Group ID", "input", "h43"},
		{"44_UploadFront", "Upload Insurance Card (Front)", "clickable", "h44"},
		{"45_UploadBack", "Upload Insurance Card (Back)", "clickable", "h45"},
		{"46_VerifyCoverage", "Verify Coverage", "clickable", "h46"},
		// test 47 skipped — extract
		{"48_PrimaryCare", "Primary Care Physician", "input", "h48"},
		{"49_SearchProviders", "Search Providers", "clickable", "h49"},
		{"50_PrimaryPolicyholder", "primary policyholder", "clickable", "h50"},

		// Appointments
		{"51_ScheduleAppt", "Schedule New Appointment", "clickable", "h51"},
		// test 52 → select
		{"53_PreferredDate", "Preferred Date", "input", "h53"},
		// test 54 → select
		{"55_FindSlots", "Find Available Slots", "clickable", "h55"},
		{"56_0900AM", "09:00 AM", "clickable", "h56"},
		{"57_1030AM", "10:30 AM", "clickable", "h57"},
		{"58_ConfirmAppt", "Confirm Appointment", "clickable", "h58"},
		{"59_CancelAppt", "Cancel Appointment", "clickable", "h59"},
		{"60_Reschedule", "Reschedule", "clickable", "h60"},

		// Accessibility
		{"61_HighContrast", "Toggle High Contrast", "clickable", "h61"},
		{"62_IncTextSize", "Increase Text Size", "clickable", "h62"},
		{"63_DecTextSize", "Decrease Text Size", "clickable", "h63"},
		// test 64 → select
		{"65_SkipToMain", "Skip to main content", "clickable", "h65"},
		{"66_PrintPage", "Print Page", "clickable", "h66"},
		{"67_DownloadPDF", "Download PDF", "clickable", "h67"},
		{"68_ChatAssistant", "Chat with Virtual Assistant", "clickable", "h68"},
		{"69_LiveAgent", "Contact Live Agent", "clickable", "h69"},
		{"70_LeaveFeedback", "Leave Feedback", "clickable", "h70"},

		// Consent & Signatures
		// test 71 skipped — verify
		{"72_PrivacyNotice", "receipt of the Privacy Notice", "clickable", "h72"},
		{"73_Telehealth", "telehealth services", "clickable", "h73"},
		{"74_SignHere", "Sign here", "clickable", "h74"},
		{"75_ClearSignature", "Clear Signature", "clickable", "h75"},
		{"76_TypeNameSign", "Type name to sign", "input", "h76"},
		{"77_SignatureDate", "Signature Date", "input", "h77"},
		{"78_Relationship", "Relationship to patient", "input", "h78"},
		{"79_AcceptSign", "Accept & Sign", "clickable", "h79"},
		{"80_Decline", "Decline", "clickable", "h80"},

		// Records
		{"81_DownloadW2", "Download W-2", "clickable", "h81"},
		{"82_ViewRecord", "View Record", "clickable", "h82"},
		{"83_OpenPDF", "Open PDF", "clickable", "h83"},
		{"84_RequestRecords", "Request Medical Records", "clickable", "h84"},
		{"85_AppealTax", "Appeal Tax Decision", "clickable", "h85"},
		{"86_SearchRecords", "Search records by year", "input", "h86"},
		{"87_SearchRecordsBtn", "Search Records", "clickable", "h87"},
		// test 88 → select
		// test 89 skipped — extract
		{"90_UploadDoc", "Upload New Document", "clickable", "h90"},

		// Form Navigation
		{"91_SaveDraft", "Save Draft", "clickable", "h91"},
		{"92_NextSection", "Next Section", "clickable", "h92"},
		{"93_PreviousSection", "Previous Section", "clickable", "h93"},
		{"94_SubmitApp", "SUBMIT APPLICATION", "clickable", "h94"},
		{"95_ClearForm", "CLEAR FORM", "clickable", "h95"},
		// test 96 skipped — verify (hidden)
		{"97_YesSubmit", "Yes, Submit", "clickable", "h97"},
		{"98_NoGoBack", "No, Go Back", "clickable", "h98"},
		// test 99 skipped — verify (disabled)
		{"100_SecureLogout", "Secure Logout", "clickable", "h100"},
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

func TestGovHealth_Select(t *testing.T) {
	elements := govHealthDOM()

	tests := []struct {
		name, query, expectedID string
	}{
		{"05_IDType", "ID Type", "h5"},
		{"14_Suffix", "Suffix", "h14"},
		{"17_MaritalStatus", "Marital Status", "h17"},
		{"24_State", "State", "h24"},
		{"41_InsuranceProvider", "Insurance Provider", "h41"},
		{"52_ReasonForVisit", "Reason for Visit", "h52"},
		{"54_PreferredTime", "Preferred Time", "h54"},
		{"64_LanguageTranslation", "Language Translation", "h64"},
		{"88_FilterRecords", "Filter Records", "h88"},
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

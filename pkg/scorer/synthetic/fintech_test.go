package synthetic

// ─────────────────────────────────────────────────────────────────────────────
// FINTECH DOM SCORING TEST SUITE
//
// Port of ManulEngine test_05_fintech.py — 100-element banking/crypto page.
// Validates: account overview, transfers, crypto trading, transactions,
// security/PIN, card management, KYC, loans, investments, session.
// Skipped: extract (1,3,4,21,31,32,34,35,51,74,75,89), verify (30,72,80,93,100),
//          optional (20).
// ─────────────────────────────────────────────────────────────────────────────

import (
	"testing"

	"github.com/alexbeatnik/ManulHeart/pkg/dom"
)

func fintechDOM() []dom.ElementSnapshot {
	return []dom.ElementSnapshot{
		// Account Overview (f1-f10)
		el(1, "/html/body/h1[1]", withTag("h1"), withID("f1"), withClassName("blurred"), withText("$124,500.00")),
		el(2, "/html/body/button[1]", withTag("button"), withID("f2"), withAriaLabel("Reveal Balance"), withText("👁️")),
		el(3, "/html/body/span[1]", withTag("span"), withID("f3"), withText("$5,000.00")),
		el(4, "/html/body/span[2]", withTag("span"), withID("f4"), withText("$119,500.00")),
		el(5, "/html/body/button[2]", withTag("button"), withID("f5"), withText("Add Funds")),
		el(6, "/html/body/button[3]", withTag("button"), withID("f6"), withText("Withdraw")),
		el(7, "/html/body/button[4]", withTag("button"), withID("f7"), withDataTestID("quick-xfer"), withText("Quick Transfer")),
		el(8, "/html/body/a[1]", withTag("a"), withID("f8"), withText("Download Statements")),
		el(9, "/html/body/div[1]", withTag("div"), withRole("switch"), withID("f9"), withText("Hide zero balances")),
		el(10, "/html/body/button[5]", withTag("button"), withID("f10"), withAriaLabel("Account Settings"), withText("⚙️")),

		// Transfers (f11-f20)
		el(11, "/html/body/input[1]", withTag("input"), withInputType("text"), withID("f11"), withPlaceholder("Recipient Name or IBAN")),
		el(12, "/html/body/input[2]", withTag("input"), withInputType("number"), withID("f12"), withAriaLabel("Amount to Send")),
		el(13, "/html/body/select[1]", withTag("select"), withID("f13"), withAriaLabel("Currency"), withText("USD")),
		el(14, "/html/body/input[3]", withTag("input"), withInputType("text"), withID("f14"), withPlaceholder("Reference / Message")),
		el(15, "/html/body/input[4]", withTag("input"), withInputType("radio"), withID("f15"), withLabel("Standard (Free)"), withNameAttr("speed")),
		el(16, "/html/body/input[5]", withTag("input"), withInputType("radio"), withID("f16"), withLabel("Instant ($1.50)"), withNameAttr("speed")),
		el(17, "/html/body/button[6]", withTag("button"), withID("f17"), withClassName("btn-primary"), withText("Review Transfer")),
		el(18, "/html/body/button[7]", withTag("button"), withID("f18"), withClassName("btn-secondary"), withText("Save as Template")),
		el(19, "/html/body/button[8]", withTag("button"), withID("f19"), withText("Schedule for later")),
		el(20, "/html/body/button[9]", withTag("button"), withID("f20"), withHidden(), withText("Cancel Transfer")),

		// Crypto Trading (f21-f30)
		el(21, "/html/body/span[3]", withTag("span"), withID("f21"), withText("BTC/USDT")),
		el(22, "/html/body/button[10]", withTag("button"), withID("f22"), withText("Change Pair")),
		el(23, "/html/body/button[11]", withTag("button"), withID("f23"), withClassName("active"), withText("Buy")),
		el(24, "/html/body/button[12]", withTag("button"), withID("f24"), withText("Sell")),
		el(25, "/html/body/select[2]", withTag("select"), withID("f25"), withAriaLabel("Order Type"), withText("Market")),
		el(26, "/html/body/input[6]", withTag("input"), withInputType("number"), withID("f26"), withPlaceholder("Price (USDT)")),
		el(27, "/html/body/input[7]", withTag("input"), withInputType("number"), withID("f27"), withPlaceholder("Amount (BTC)")),
		el(28, "/html/body/input[8]", withTag("input"), withInputType("range"), withID("f28"), withAriaLabel("Leverage Slider")),
		el(29, "/html/body/button[13]", withTag("button"), withID("f29"), withText("Execute Buy")),
		el(30, "/html/body/div[2]", withTag("div"), withID("f30"), withClassName("fee-display"), withText("Est. Fee: 0.1%")),

		// Transactions (f31-f40)
		el(31, "/html/body/td[1]", withTag("td"), withID("f31"), withText("-$4.50")),
		el(32, "/html/body/td[2]", withTag("td"), withID("f32"), withText("Completed")),
		el(33, "/html/body/button[14]", withTag("button"), withID("f33"), withText("Dispute")),
		el(34, "/html/body/td[3]", withTag("td"), withID("f34"), withText("+$5,000.00")),
		el(35, "/html/body/td[4]", withTag("td"), withID("f35"), withText("Pending")),
		el(36, "/html/body/button[15]", withTag("button"), withID("f36"), withText("View Receipt")),
		el(37, "/html/body/button[16]", withTag("button"), withID("f37"), withText("Load More Transactions")),
		el(38, "/html/body/input[9]", withTag("input"), withInputType("text"), withID("f38"), withPlaceholder("Search transactions")),
		el(39, "/html/body/button[17]", withTag("button"), withID("f39"), withText("Export to CSV")),
		el(40, "/html/body/button[18]", withTag("button"), withID("f40"), withText("Export to PDF")),

		// Security (f41-f50)
		el(41, "/html/body/input[10]", withTag("input"), withInputType("password"), withID("f41"), withPlaceholder("Enter Current PIN")),
		el(42, "/html/body/input[11]", withTag("input"), withInputType("password"), withID("f42"), withPlaceholder("Enter New PIN")),
		el(43, "/html/body/input[12]", withTag("input"), withInputType("password"), withID("f43"), withPlaceholder("Confirm New PIN")),
		el(44, "/html/body/button[19]", withTag("button"), withID("f44"), withText("Update PIN")),
		el(45, "/html/body/input[13]", withTag("input"), withInputType("text"), withID("f45"), withPlaceholder("6-digit OTP Code")),
		el(46, "/html/body/button[20]", withTag("button"), withID("f46"), withText("Verify OTP")),
		el(47, "/html/body/button[21]", withTag("button"), withID("f47"), withText("Resend SMS")),
		el(48, "/html/body/div[3]", withTag("div"), withRole("switch"), withID("f48"), withText("Enable FaceID")),
		el(49, "/html/body/button[22]", withTag("button"), withID("f49"), withText("Register Hardware Key")),
		el(50, "/html/body/button[23]", withTag("button"), withID("f50"), withClassName("btn-danger"), withText("Lock Account")),

		// Card Management (f51-f60)
		el(51, "/html/body/span[4]", withTag("span"), withID("f51"), withText("1234")),
		el(52, "/html/body/button[24]", withTag("button"), withID("f52"), withText("Show Card Details")),
		el(53, "/html/body/button[25]", withTag("button"), withID("f53"), withAriaLabel("Copy Card Number"), withText("📋")),
		el(54, "/html/body/div[4]", withTag("div"), withRole("switch"), withID("f54"), withText("Freeze Card")),
		el(55, "/html/body/button[26]", withTag("button"), withID("f55"), withText("Report Lost/Stolen")),
		el(56, "/html/body/button[27]", withTag("button"), withID("f56"), withText("Change Limits")),
		el(57, "/html/body/input[14]", withTag("input"), withInputType("number"), withID("f57"), withPlaceholder("Daily Limit"), withValue("1000")),
		el(58, "/html/body/button[28]", withTag("button"), withID("f58"), withText("Save Limits")),
		el(59, "/html/body/button[29]", withTag("button"), withID("f59"), withText("Create Virtual Card")),
		el(60, "/html/body/div[5]", withTag("div"), withRole("switch"), withID("f60"), withText("Online Purchases")),

		// KYC (f61-f70)
		el(61, "/html/body/input[15]", withTag("input"), withInputType("text"), withID("f61"), withPlaceholder("Legal Name")),
		el(62, "/html/body/input[16]", withTag("input"), withInputType("date"), withID("f62"), withAriaLabel("Date of Birth")),
		el(63, "/html/body/input[17]", withTag("input"), withInputType("text"), withID("f63"), withPlaceholder("Residential Address")),
		el(64, "/html/body/button[30]", withTag("button"), withID("f64"), withText("Verify Identity (KYC)")),
		el(65, "/html/body/select[3]", withTag("select"), withID("f65"), withAriaLabel("Account Tier"), withText("Standard")),
		el(66, "/html/body/input[18]", withTag("input"), withInputType("radio"), withID("f66"), withLabel("Low Risk"), withNameAttr("risk")),
		el(67, "/html/body/input[19]", withTag("input"), withInputType("radio"), withID("f67"), withLabel("Medium Risk"), withNameAttr("risk")),
		el(68, "/html/body/input[20]", withTag("input"), withInputType("radio"), withID("f68"), withLabel("High Risk"), withNameAttr("risk")),
		el(69, "/html/body/button[31]", withTag("button"), withID("f69"), withText("Save Profile")),
		el(70, "/html/body/a[2]", withTag("a"), withID("f70"), withText("Tax Documents")),

		// Loans (f71-f80)
		el(71, "/html/body/input[21]", withTag("input"), withInputType("range"), withID("f71"), withAriaLabel("Loan Amount")),
		el(72, "/html/body/div[6]", withTag("div"), withID("f72"), withText("$10,000")),
		el(73, "/html/body/select[4]", withTag("select"), withID("f73"), withAriaLabel("Term"), withText("12 Months")),
		el(74, "/html/body/span[5]", withTag("span"), withID("f74"), withText("5.99%")),
		el(75, "/html/body/span[6]", withTag("span"), withID("f75"), withText("$860.00")),
		el(76, "/html/body/button[32]", withTag("button"), withID("f76"), withText("Apply for Loan")),
		el(77, "/html/body/button[33]", withTag("button"), withID("f77"), withText("View Amortization Schedule")),
		el(78, "/html/body/input[22]", withTag("input"), withInputType("checkbox"), withID("f78"), withLabel("Include Payment Protection")),
		el(79, "/html/body/button[34]", withTag("button"), withID("f79"), withText("Pay Early")),
		el(80, "/html/body/div[7]", withTag("div"), withID("f80"), withText("Credit Score: 750")),

		// Investments (f81-f90)
		el(81, "/html/body/button[35]", withTag("button"), withID("f81"), withText("Create Portfolio")),
		el(82, "/html/body/input[23]", withTag("input"), withInputType("text"), withID("f82"), withPlaceholder("Ticker (e.g. AAPL)")),
		el(83, "/html/body/button[36]", withTag("button"), withID("f83"), withText("Search Asset")),
		el(84, "/html/body/div[8]", withTag("div"), withRole("switch"), withID("f84"), withText("Auto-Invest")),
		el(85, "/html/body/input[24]", withTag("input"), withInputType("number"), withID("f85"), withPlaceholder("Auto-deposit amount")),
		el(86, "/html/body/select[5]", withTag("select"), withID("f86"), withAriaLabel("Frequency"), withText("Weekly")),
		el(87, "/html/body/button[37]", withTag("button"), withID("f87"), withText("Confirm Auto-Invest")),
		el(88, "/html/body/button[38]", withTag("button"), withID("f88"), withText("Rebalance Portfolio")),
		el(89, "/html/body/div[9]", withTag("div"), withID("f89"), withText("YTD Return: +12.4%")),
		el(90, "/html/body/button[39]", withTag("button"), withID("f90"), withText("Withdraw Funds")),

		// Session & Modals (f91-f100)
		el(91, "/html/body/button[40]", withTag("button"), withID("f91"), withText("Stay Logged In")),
		el(92, "/html/body/button[41]", withTag("button"), withID("f92"), withText("Log Out Now")),
		el(93, "/html/body/span[7]", withTag("span"), withID("f93"), withText("Transfer Successful")),
		el(94, "/html/body/button[42]", withTag("button"), withID("f94"), withAriaLabel("Dismiss Alert"), withText("X")),
		el(95, "/html/body/button[43]", withTag("button"), withID("f95"), withClassName("confirm-transfer"), withText("Confirm")),
		el(96, "/html/body/button[44]", withTag("button"), withID("f96"), withClassName("confirm-delete"), withText("Confirm")),
		el(97, "/html/body/button[45]", withTag("button"), withID("f97"), withClassName("confirm-generic"), withText("Confirm")),
		el(98, "/html/body/div[10]", withTag("div"), withRole("button"), withID("f98"), withText("Acknowledge Risk")),
		el(99, "/html/body/input[25]", withTag("input"), withInputType("checkbox"), withID("f99"), withLabel("I agree to the updated terms")),
		el(100, "/html/body/button[46]", withTag("button"), withID("f100"), withDisabled(), withText("Finalize")),
	}
}

func TestFintech(t *testing.T) {
	elements := fintechDOM()

	tests := []struct {
		name, query, mode, expectedID string
	}{
		// Account
		// 1,3,4 = EXTRACT
		{"02_RevealBalance", "Reveal Balance", "clickable", "f2"},
		{"05_AddFunds", "Add Funds", "clickable", "f5"},
		{"06_Withdraw", "Withdraw", "clickable", "f6"},
		{"07_QuickTransfer", "Quick Transfer", "clickable", "f7"},
		{"08_DownloadStatements", "Download Statements", "clickable", "f8"},
		{"09_HideZeroBalances", "Hide zero balances", "clickable", "f9"},
		{"10_AccountSettings", "Account Settings", "clickable", "f10"},

		// Transfers
		{"11_RecipientIBAN", "Recipient Name or IBAN", "input", "f11"},
		{"12_AmountToSend", "Amount to Send", "input", "f12"},
		{"14_Reference", "Reference", "input", "f14"},
		{"15_StandardFree", "Standard (Free)", "clickable", "f15"},
		{"16_Instant", "Instant", "clickable", "f16"},
		{"17_ReviewTransfer", "Review Transfer", "clickable", "f17"},
		{"18_SaveAsTemplate", "Save as Template", "clickable", "f18"},
		{"19_ScheduleForLater", "Schedule for later", "clickable", "f19"},
		// 20 = optional hidden

		// Crypto
		// 21 = EXTRACT
		{"22_ChangePair", "Change Pair", "clickable", "f22"},
		{"23_Buy", "Buy", "clickable", "f23"},
		{"24_Sell", "Sell", "clickable", "f24"},
		{"26_PriceUSDT", "Price (USDT)", "input", "f26"},
		{"27_AmountBTC", "Amount (BTC)", "input", "f27"},
		{"28_LeverageSlider", "Leverage Slider", "input", "f28"},
		{"29_ExecuteBuy", "Execute Buy", "clickable", "f29"},

		// Transactions
		// 31,32,34,35 = EXTRACT
		{"33_Dispute", "Dispute", "clickable", "f33"},
		{"36_ViewReceipt", "View Receipt", "clickable", "f36"},
		{"37_LoadMoreTransactions", "Load More Transactions", "clickable", "f37"},
		{"38_SearchTransactions", "Search transactions", "input", "f38"},
		{"39_ExportToCSV", "Export to CSV", "clickable", "f39"},
		{"40_ExportToPDF", "Export to PDF", "clickable", "f40"},

		// Security
		{"41_CurrentPIN", "Enter Current PIN", "input", "f41"},
		{"42_NewPIN", "Enter New PIN", "input", "f42"},
		{"43_ConfirmPIN", "Confirm New PIN", "input", "f43"},
		{"44_UpdatePIN", "Update PIN", "clickable", "f44"},
		{"45_OTPCode", "OTP Code", "input", "f45"},
		{"46_VerifyOTP", "Verify OTP", "clickable", "f46"},
		{"47_ResendSMS", "Resend SMS", "clickable", "f47"},
		{"48_EnableFaceID", "Enable FaceID", "clickable", "f48"},
		{"49_RegisterHardwareKey", "Register Hardware Key", "clickable", "f49"},
		{"50_LockAccount", "Lock Account", "clickable", "f50"},

		// Card Management
		// 51 = EXTRACT
		{"52_ShowCardDetails", "Show Card Details", "clickable", "f52"},
		{"53_CopyCardNumber", "Copy Card Number", "clickable", "f53"},
		{"54_FreezeCard", "Freeze Card", "clickable", "f54"},
		{"55_ReportLostStolen", "Report Lost/Stolen", "clickable", "f55"},
		{"56_ChangeLimits", "Change Limits", "clickable", "f56"},
		{"57_DailyLimit", "Daily Limit", "input", "f57"},
		{"58_SaveLimits", "Save Limits", "clickable", "f58"},
		{"59_CreateVirtualCard", "Create Virtual Card", "clickable", "f59"},
		{"60_OnlinePurchases", "Online Purchases", "clickable", "f60"},

		// KYC
		{"61_LegalName", "Legal Name", "input", "f61"},
		{"62_DateOfBirth", "Date of Birth", "input", "f62"},
		{"63_ResidentialAddress", "Residential Address", "input", "f63"},
		{"64_VerifyIdentityKYC", "Verify Identity (KYC)", "clickable", "f64"},
		{"66_LowRisk", "Low Risk", "clickable", "f66"},
		{"67_MediumRisk", "Medium Risk", "clickable", "f67"},
		{"68_HighRisk", "High Risk", "clickable", "f68"},
		{"69_SaveProfile", "Save Profile", "clickable", "f69"},
		{"70_TaxDocuments", "Tax Documents", "clickable", "f70"},

		// Loans
		{"71_LoanAmount", "Loan Amount", "input", "f71"},
		// 72 = VERIFY, 74,75 = EXTRACT
		{"76_ApplyForLoan", "Apply for Loan", "clickable", "f76"},
		{"77_ViewAmortizationSchedule", "View Amortization Schedule", "clickable", "f77"},
		{"78_IncludePaymentProtection", "Include Payment Protection", "clickable", "f78"},
		{"79_PayEarly", "Pay Early", "clickable", "f79"},

		// Investments
		{"81_CreatePortfolio", "Create Portfolio", "clickable", "f81"},
		{"82_Ticker", "Ticker", "input", "f82"},
		{"83_SearchAsset", "Search Asset", "clickable", "f83"},
		{"84_AutoInvest", "Auto-Invest", "clickable", "f84"},
		{"85_AutoDepositAmount", "Auto-deposit amount", "input", "f85"},
		{"87_ConfirmAutoInvest", "Confirm Auto-Invest", "clickable", "f87"},
		{"88_RebalancePortfolio", "Rebalance Portfolio", "clickable", "f88"},
		// 89 = EXTRACT
		{"90_WithdrawFunds", "Withdraw Funds", "clickable", "f90"},

		// Session
		{"91_StayLoggedIn", "Stay Logged In", "clickable", "f91"},
		{"92_LogOutNow", "Log Out Now", "clickable", "f92"},
		// 93 = VERIFY
		{"94_DismissAlert", "Dismiss Alert", "clickable", "f94"},
		{"98_AcknowledgeRisk", "Acknowledge Risk", "clickable", "f98"},
		{"99_AgreeTerms", "I agree to the updated terms", "clickable", "f99"},
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

func TestFintech_Select(t *testing.T) {
	elements := fintechDOM()

	tests := []struct {
		name, query, expectedID string
	}{
		{"13_Currency", "Currency", "f13"},
		{"25_OrderType", "Order Type", "f25"},
		{"65_AccountTier", "Account Tier", "f65"},
		{"73_Term", "Term", "f73"},
		{"86_Frequency", "Frequency", "f86"},
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

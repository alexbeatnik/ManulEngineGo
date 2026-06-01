package synthetic

// ─────────────────────────────────────────────────────────────────────────────
// SAAS ADMIN DOM SCORING TEST SUITE
//
// Port of ManulEngine test_03_saas.py — 100-element SaaS admin panel.
// Validates: sidebar nav, dashboard widgets, user table, pagination, invite modal,
// API keys, billing, workspace settings, danger zone.
// Skipped: extract (11,13,34,35,64), verify (20,39,41,42,55,93,95,99),
//          execute_step (94,96), optional (100).
// ─────────────────────────────────────────────────────────────────────────────

import (
	"testing"

	"github.com/alexbeatnik/ManulHeart/pkg/dom"
)

func saasDOM() []dom.ElementSnapshot {
	return []dom.ElementSnapshot{
		// Sidebar Navigation (a1-a10)
		el(1, "/html/body/nav/a[1]", withTag("a"), withID("a1"), withText("Dashboard")),
		el(2, "/html/body/nav/a[2]", withTag("a"), withID("a2"), withText("User Management")),
		el(3, "/html/body/nav/a[3]", withTag("a"), withID("a3"), withText("Billing & Plans")),
		el(4, "/html/body/nav/a[4]", withTag("a"), withID("a4"), withText("Workspace Settings")),
		el(5, "/html/body/nav/button[1]", withTag("button"), withID("a5"), withAriaLabel("Collapse Sidebar"), withText("◀")),
		el(6, "/html/body/nav/div[1]", withTag("div"), withRole("button"), withID("a6"), withText("Help & Support")),
		el(7, "/html/body/nav/a[5]", withTag("a"), withID("a7"), withText("API Keys")),
		el(8, "/html/body/nav/a[6]", withTag("a"), withID("a8"), withText("Audit Logs")),
		el(9, "/html/body/nav/button[2]", withTag("button"), withID("a9"), withDataQA("user-profile-menu"), withText("Profile")),
		el(10, "/html/body/nav/button[3]", withTag("button"), withID("a10"), withText("Sign Out")),

		// Dashboard Widgets (a11-a20, close_alert)
		el(11, "/html/body/div[1]", withTag("div"), withID("a11"), withClassName("metric-value"), withDataQA("mrr-value"), withText("$45,200")),
		el(12, "/html/body/button[1]", withTag("button"), withID("a12"), withText("Download Report")),
		el(13, "/html/body/span[1]", withTag("span"), withID("a13"), withText("1,240")),
		el(14, "/html/body/button[2]", withTag("button"), withID("a14"), withAriaLabel("Refresh Data"), withText("🔄")),
		el(15, "/html/body/button[3]", withTag("button"), withID("a15"), withText("Export to CSV")),
		el(16, "/html/body/button[4]", withTag("button"), withID("a16"), withText("Export to PDF")),
		el(17, "/html/body/div[2]", withTag("div"), withRole("button"), withID("a17"), withText("Customize Dashboard")),
		el(18, "/html/body/select[1]", withTag("select"), withID("a18"), withText("Last 7 days")),
		el(19, "/html/body/button[5]", withTag("button"), withID("a19"), withText("Add Widget")),
		el(20, "/html/body/div[3]", withTag("div"), withID("a20"), withClassName("alert alert-warning"), withText("Server load high")),
		el(101, "/html/body/button[6]", withTag("button"), withID("close_alert"), withText("x")),

		// Filters (a21-a30)
		el(22, "/html/body/input[1]", withTag("input"), withInputType("text"), withID("a21"), withPlaceholder("Search users by email...")),
		el(23, "/html/body/select[2]", withTag("select"), withID("a22"), withAriaLabel("Filter by Status"), withText("All")),
		el(24, "/html/body/select[3]", withTag("select"), withID("a23"), withAriaLabel("Filter by Role"), withText("All")),
		el(25, "/html/body/input[2]", withTag("input"), withInputType("date"), withID("a24"), withAriaLabel("Start Date")),
		el(26, "/html/body/input[3]", withTag("input"), withInputType("date"), withID("a25"), withAriaLabel("End Date")),
		el(27, "/html/body/button[7]", withTag("button"), withID("a26"), withClassName("btn-primary"), withText("Apply Filters")),
		el(28, "/html/body/button[8]", withTag("button"), withID("a27"), withClassName("btn-ghost"), withText("Clear Filters")),
		el(29, "/html/body/button[9]", withTag("button"), withID("a28"), withAriaLabel("Advanced Search"), withText("⚙️")),
		el(30, "/html/body/input[4]", withTag("input"), withInputType("checkbox"), withID("a29"), withLabel("Show deleted records")),
		el(31, "/html/body/button[10]", withTag("button"), withID("a30"), withText("Save View")),

		// User Table (a31-a50)
		el(32, "/html/body/input[5]", withTag("input"), withInputType("checkbox"), withID("a31"), withAriaLabel("Select All Rows")),
		el(33, "/html/body/button[11]", withTag("button"), withID("a32"), withAriaLabel("Sort Ascending"), withText("↑")),
		el(34, "/html/body/input[6]", withTag("input"), withInputType("checkbox"), withID("a33"), withAriaLabel("Select Alice")),
		el(35, "/html/body/td[1]", withTag("td"), withID("a34"), withText("Admin")),
		el(36, "/html/body/span[2]", withTag("span"), withID("a35"), withClassName("badge badge-success"), withText("Active")),
		el(37, "/html/body/button[12]", withTag("button"), withID("a36"), withAriaLabel("Edit Alice"), withText("✏️")),
		el(38, "/html/body/button[13]", withTag("button"), withID("a37"), withAriaLabel("Suspend Alice"), withText("🛑")),
		el(39, "/html/body/div[4]", withTag("div"), withRole("button"), withID("a38"), withAriaLabel("More actions for Alice"), withText("•••")),
		el(40, "/html/body/button[14]", withTag("button"), withID("a39"), withDisabled(), withText("Bulk Delete")),
		el(41, "/html/body/button[15]", withTag("button"), withID("a40"), withText("Bulk Export")),
		el(42, "/html/body/span[3]", withTag("span"), withID("a41"), withText("Showing 1 to 10 of 500 entries")),

		// Pagination (a42-a50)
		el(43, "/html/body/button[16]", withTag("button"), withID("a42"), withDisabled(), withText("Previous")),
		el(44, "/html/body/button[17]", withTag("button"), withID("a43"), withText("1")),
		el(45, "/html/body/button[18]", withTag("button"), withID("a44"), withText("2")),
		el(46, "/html/body/button[19]", withTag("button"), withID("a45"), withText("3")),
		el(47, "/html/body/button[20]", withTag("button"), withID("a46"), withText("Next")),
		el(48, "/html/body/button[21]", withTag("button"), withID("a47"), withText("Last")),
		el(49, "/html/body/select[4]", withTag("select"), withID("a48"), withAriaLabel("Rows per page"), withText("10")),
		el(50, "/html/body/input[7]", withTag("input"), withInputType("number"), withID("a49"), withPlaceholder("Go to page")),
		el(51, "/html/body/button[22]", withTag("button"), withID("a50"), withText("Go")),

		// Invite User Modal (a51-a60)
		el(52, "/html/body/button[23]", withTag("button"), withID("a51"), withText("Invite User")),
		el(53, "/html/body/input[8]", withTag("input"), withInputType("email"), withID("a52"), withPlaceholder("colleague@company.com")),
		el(54, "/html/body/input[9]", withTag("input"), withInputType("radio"), withID("a53"), withLabel("Admin"), withNameAttr("role")),
		el(55, "/html/body/input[10]", withTag("input"), withInputType("radio"), withID("a54"), withLabel("Editor"), withNameAttr("role")),
		el(56, "/html/body/input[11]", withTag("input"), withInputType("radio"), withID("a55"), withLabel("Viewer"), withNameAttr("role")),
		el(57, "/html/body/select[5]", withTag("select"), withID("a56"), withAriaLabel("Assign to Department"), withText("Engineering")),
		el(58, "/html/body/input[12]", withTag("input"), withInputType("checkbox"), withID("a57"), withLabel("Send welcome email")),
		el(59, "/html/body/button[24]", withTag("button"), withID("a58"), withClassName("btn-success"), withText("Send Invitation")),
		el(60, "/html/body/button[25]", withTag("button"), withID("a59"), withClassName("btn-cancel"), withText("Cancel")),
		el(61, "/html/body/button[26]", withTag("button"), withID("a60"), withAriaLabel("Close Modal"), withText("X")),

		// API Keys (a61-a70)
		el(62, "/html/body/button[27]", withTag("button"), withID("a61"), withText("Generate New API Key")),
		el(63, "/html/body/input[13]", withTag("input"), withInputType("text"), withID("a62"), withPlaceholder("Key Name (e.g. Production)")),
		el(64, "/html/body/button[28]", withTag("button"), withID("a63"), withText("Create Key")),
		el(65, "/html/body/span[4]", withTag("span"), withID("a64"), withDataQA("api-key-value"), withText("sk_live_123456789")),
		el(66, "/html/body/button[29]", withTag("button"), withID("a65"), withAriaLabel("Reveal API Key"), withText("👁️")),
		el(67, "/html/body/button[30]", withTag("button"), withID("a66"), withAriaLabel("Copy API Key"), withText("📋")),
		el(68, "/html/body/button[31]", withTag("button"), withID("a67"), withText("Revoke Key")),
		el(69, "/html/body/button[32]", withTag("button"), withID("a68"), withText("Add Webhook")),
		el(70, "/html/body/input[14]", withTag("input"), withInputType("url"), withID("a69"), withPlaceholder("https://your-domain.com/webhook")),
		el(71, "/html/body/div[5]", withTag("div"), withRole("switch"), withID("a70"), withAriaLabel("Enable Webhook")),

		// Billing (a71-a80)
		el(72, "/html/body/button[33]", withTag("button"), withID("a71"), withText("Upgrade to Enterprise")),
		el(73, "/html/body/button[34]", withTag("button"), withID("a72"), withText("Add Payment Method")),
		el(74, "/html/body/input[15]", withTag("input"), withInputType("text"), withID("a73"), withPlaceholder("Cardholder Name")),
		el(75, "/html/body/button[35]", withTag("button"), withID("a74"), withText("Save Card")),
		el(76, "/html/body/button[36]", withTag("button"), withID("a75"), withText("Download Invoices")),
		el(77, "/html/body/button[37]", withTag("button"), withID("a76"), withText("Update Billing Email")),
		el(78, "/html/body/input[16]", withTag("input"), withInputType("text"), withID("a77"), withPlaceholder("Tax ID / VAT")),
		el(79, "/html/body/button[38]", withTag("button"), withID("a78"), withClassName("btn-danger"), withText("Cancel Subscription")),
		el(80, "/html/body/a[7]", withTag("a"), withID("a79"), withText("Terms of Service")),
		el(81, "/html/body/div[6]", withTag("div"), withRole("button"), withID("a80"), withText("Contact Sales")),

		// Workspace Settings (a81-a92)
		el(82, "/html/body/input[17]", withTag("input"), withInputType("text"), withID("a81"), withAriaLabel("Workspace Name"), withValue("Acme Corp")),
		el(83, "/html/body/input[18]", withTag("input"), withInputType("file"), withID("a82"), withAriaLabel("Upload Logo")),
		el(84, "/html/body/select[6]", withTag("select"), withID("a83"), withAriaLabel("Timezone"), withText("UTC")),
		el(85, "/html/body/div[7]", withTag("div"), withRole("switch"), withID("a84"), withAriaLabel("Require MFA for all users")),
		el(86, "/html/body/button[39]", withTag("button"), withID("a85"), withText("Save Workspace Settings")),
		el(87, "/html/body/button[40]", withTag("button"), withID("a86"), withText("Transfer Ownership")),
		el(88, "/html/body/input[19]", withTag("input"), withInputType("text"), withID("a87"), withPlaceholder("Transfer to email")),
		el(89, "/html/body/select[7]", withTag("select"), withID("a88"), withAriaLabel("Language"), withText("English")),
		el(90, "/html/body/button[41]", withTag("button"), withID("a89"), withText("Sync Directory")),
		el(91, "/html/body/button[42]", withTag("button"), withID("a90"), withText("Clear Workspace Cache")),

		// Danger Zone (a91-a100)
		el(92, "/html/body/button[43]", withTag("button"), withID("a91"), withText("Delete Workspace")),
		el(93, "/html/body/input[20]", withTag("input"), withInputType("text"), withID("a92"), withPlaceholder("Type workspace name to confirm")),
		el(94, "/html/body/button[44]", withTag("button"), withID("a93"), withDisabled(), withText("Confirm Deletion")),
		el(95, "/html/body/div[8]", withTag("div"), withID("a94"), withClassName("tooltip"), withText("Hover me for info")),
		el(96, "/html/body/input[21]", withTag("input"), withInputType("text"), withID("a95"), withDisabled(), withValue("Cannot edit this")),
		el(97, "/html/body/input[22]", withTag("input"), withInputType("text"), withID("a96"), withValue("Readonly data")),
		el(98, "/html/body/div[9]", withTag("div"), withRole("menuitem"), withID("a97"), withText("Quick Action 1")),
		el(99, "/html/body/div[10]", withTag("div"), withRole("menuitem"), withID("a98"), withText("Quick Action 2")),
		el(100, "/html/body/div[11]", withTag("div"), withID("a99"), withRole("progressbar"), withAriaLabel("Progress: 75%")),
		el(102, "/html/body/button[45]", withTag("button"), withID("a100"), withHidden(), withText("Easter Egg Button")),
	}
}

func TestSaasAdmin(t *testing.T) {
	elements := saasDOM()

	tests := []struct {
		name, query, mode, expectedID string
	}{
		// Sidebar
		{"01_Dashboard", "Dashboard", "clickable", "a1"},
		{"02_UserManagement", "User Management", "clickable", "a2"},
		{"03_BillingPlans", "Billing & Plans", "clickable", "a3"},
		{"04_WorkspaceSettings", "Workspace Settings", "clickable", "a4"},
		{"05_CollapseSidebar", "Collapse Sidebar", "clickable", "a5"},
		{"06_HelpSupport", "Help & Support", "clickable", "a6"},
		{"07_APIKeys", "API Keys", "clickable", "a7"},
		{"08_AuditLogs", "Audit Logs", "clickable", "a8"},
		{"09_Profile", "Profile", "clickable", "a9"},
		{"10_SignOut", "Sign Out", "clickable", "a10"},

		// Dashboard
		// 11 = EXTRACT, 12:
		{"12_DownloadReport", "Download Report", "clickable", "a12"},
		// 13 = EXTRACT
		{"14_RefreshData", "Refresh Data", "clickable", "a14"},
		{"15_ExportToCSV", "Export to CSV", "clickable", "a15"},
		{"16_ExportToPDF", "Export to PDF", "clickable", "a16"},
		{"17_CustomizeDashboard", "Customize Dashboard", "clickable", "a17"},
		{"19_AddWidget", "Add Widget", "clickable", "a19"},

		// Filters
		{"21_SearchUsers", "Search users", "input", "a21"},
		{"26_ApplyFilters", "Apply Filters", "clickable", "a26"},
		{"27_ClearFilters", "Clear Filters", "clickable", "a27"},
		{"28_AdvancedSearch", "Advanced Search", "clickable", "a28"},
		{"29_ShowDeletedRecords", "Show deleted records", "clickable", "a29"},
		{"30_SaveView", "Save View", "clickable", "a30"},
		{"24_StartDate", "Start Date", "input", "a24"},
		{"25_EndDate", "End Date", "input", "a25"},

		// User Table
		{"31_SelectAllRows", "Select All Rows", "clickable", "a31"},
		{"32_SortAscending", "Sort Ascending", "clickable", "a32"},
		{"33_SelectAlice", "Select Alice", "clickable", "a33"},
		// 34, 35 = EXTRACT
		{"36_EditAlice", "Edit Alice", "clickable", "a36"},
		{"37_SuspendAlice", "Suspend Alice", "clickable", "a37"},
		{"38_MoreActionsAlice", "More actions for Alice", "clickable", "a38"},
		// 39 = VERIFY disabled
		{"40_BulkExport", "Bulk Export", "clickable", "a40"},

		// Pagination
		{"43_Page1", "1", "clickable", "a43"},
		{"44_Page2", "2", "clickable", "a44"},
		{"45_Page3", "3", "clickable", "a45"},
		{"46_Next", "Next", "clickable", "a46"},
		{"47_Last", "Last", "clickable", "a47"},
		{"49_GoToPage", "Go to page", "input", "a49"},
		{"50_Go", "Go", "clickable", "a50"},

		// Invite User
		{"51_InviteUser", "Invite User", "clickable", "a51"},
		{"52_ColleagueEmail", "colleague@", "input", "a52"},
		{"53_AdminRadio", "Admin", "clickable", "a53"},
		{"54_EditorRadio", "Editor", "clickable", "a54"},
		{"57_SendWelcomeEmail", "Send welcome email", "clickable", "a57"},
		{"58_SendInvitation", "Send Invitation", "clickable", "a58"},
		{"59_Cancel", "Cancel", "clickable", "a59"},
		{"60_CloseModal", "Close Modal", "clickable", "a60"},

		// API Keys
		{"61_GenerateNewAPIKey", "Generate New API Key", "clickable", "a61"},
		{"62_KeyName", "Key Name", "input", "a62"},
		{"63_CreateKey", "Create Key", "clickable", "a63"},
		// 64 = EXTRACT
		{"65_RevealAPIKey", "Reveal API Key", "clickable", "a65"},
		{"66_CopyAPIKey", "Copy API Key", "clickable", "a66"},
		{"67_RevokeKey", "Revoke Key", "clickable", "a67"},
		{"68_AddWebhook", "Add Webhook", "clickable", "a68"},
		{"69_WebhookURL", "https://", "input", "a69"},
		{"70_EnableWebhook", "Enable Webhook", "clickable", "a70"},

		// Billing
		{"71_UpgradeToEnterprise", "Upgrade to Enterprise", "clickable", "a71"},
		{"72_AddPaymentMethod", "Add Payment Method", "clickable", "a72"},
		{"73_CardholderName", "Cardholder Name", "input", "a73"},
		{"74_SaveCard", "Save Card", "clickable", "a74"},
		{"75_DownloadInvoices", "Download Invoices", "clickable", "a75"},
		{"76_UpdateBillingEmail", "Update Billing Email", "clickable", "a76"},
		{"77_TaxID", "Tax ID", "input", "a77"},
		{"78_CancelSubscription", "Cancel Subscription", "clickable", "a78"},
		{"79_TermsOfService", "Terms of Service", "clickable", "a79"},
		{"80_ContactSales", "Contact Sales", "clickable", "a80"},

		// Workspace Settings
		{"81_WorkspaceName", "Workspace Name", "input", "a81"},
		{"82_UploadLogo", "Upload Logo", "clickable", "a82"},
		{"84_RequireMFA", "Require MFA", "clickable", "a84"},
		{"85_SaveWorkspaceSettings", "Save Workspace Settings", "clickable", "a85"},
		{"86_TransferOwnership", "Transfer Ownership", "clickable", "a86"},
		{"87_TransferToEmail", "Transfer to email", "input", "a87"},
		{"89_SyncDirectory", "Sync Directory", "clickable", "a89"},
		{"90_ClearWorkspaceCache", "Clear Workspace Cache", "clickable", "a90"},

		// Danger Zone
		{"91_DeleteWorkspace", "Delete Workspace", "clickable", "a91"},
		{"92_TypeWorkspaceName", "Type workspace name", "input", "a92"},
		{"97_QuickAction1", "Quick Action 1", "clickable", "a97"},
		{"98_QuickAction2", "Quick Action 2", "clickable", "a98"},
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

func TestSaasAdmin_Select(t *testing.T) {
	elements := saasDOM()

	tests := []struct {
		name, query, expectedID string
	}{
		{"18_Last30Days", "Last 30 days", "a18"},
		{"22_FilterByStatus", "Status", "a22"},
		{"23_FilterByRole", "Filter by Role", "a23"},
		{"48_RowsPerPage", "Rows per page", "a48"},
		{"56_AssignDepartment", "Department", "a56"},
		{"83_Timezone", "Timezone", "a83"},
		{"88_Language", "Language", "a88"},
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

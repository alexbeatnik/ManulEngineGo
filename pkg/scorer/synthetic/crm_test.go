package synthetic

// ─────────────────────────────────────────────────────────────────────────────
// CRM / ATS / PM DOM SCORING TEST SUITE
//
// Port of ManulEngine test_08_crm.py — 100-element CRM/ATS/PM page.
// Validates: global nav, kanban board, lead form, activity log, tickets,
// candidate tracking, bulk operations, advanced filters, reports,
// workflow automation, custom fields.
// Skipped: extract (11,51,63,85), verify (40,70), optional/hidden (100).
// ─────────────────────────────────────────────────────────────────────────────

import (
	"testing"

	"github.com/alexbeatnik/ManulEngineGo/pkg/dom"
)

func crmDOM() []dom.ElementSnapshot {
	return []dom.ElementSnapshot{
		// Global Nav (c1-c10)
		el(1, "/html/body/input[1]", withTag("input"), withInputType("search"), withID("c1"), withPlaceholder("Search leads, contacts, tickets...")),
		el(2, "/html/body/button[1]", withTag("button"), withID("c2"), withAriaLabel("Global Create"), withText("➕ Create")),
		el(3, "/html/body/div[1]", withTag("div"), withRole("button"), withID("c3"), withAriaLabel("Notifications"), withText("🔔")),
		el(4, "/html/body/a[1]", withTag("a"), withID("c4"), withText("Pipelines")),
		el(5, "/html/body/a[2]", withTag("a"), withID("c5"), withText("Candidates (ATS)")),
		el(6, "/html/body/a[3]", withTag("a"), withID("c6"), withText("Reports")),
		el(7, "/html/body/button[2]", withTag("button"), withID("c7"), withText("Recent Items ▼")),
		el(8, "/html/body/button[3]", withTag("button"), withID("c8"), withAriaLabel("Settings"), withText("⚙️")),
		el(9, "/html/body/div[2]", withTag("div"), withRole("menuitem"), withID("c9"), withText("My Profile")),
		el(10, "/html/body/button[4]", withTag("button"), withID("c10"), withText("Log Out")),

		// Kanban Board (c11-c20)
		el(11, "/html/body/span[1]", withTag("span"), withID("c11"), withText("Acme Corp Deal")),
		el(12, "/html/body/button[5]", withTag("button"), withID("c12"), withText("Move to Contacted")),
		el(13, "/html/body/button[6]", withTag("button"), withID("c13"), withText("Move to Closed Won")),
		el(14, "/html/body/button[7]", withTag("button"), withID("c14"), withText("Move to Closed Lost")),
		el(15, "/html/body/button[8]", withTag("button"), withID("c15"), withText("Add Column")),
		el(16, "/html/body/input[2]", withTag("input"), withInputType("text"), withID("c16"), withPlaceholder("Column Name")),
		el(17, "/html/body/button[9]", withTag("button"), withID("c17"), withText("Save Column")),
		el(18, "/html/body/button[10]", withTag("button"), withID("c18"), withAriaLabel("Filter Board"), withText("Filter")),
		el(19, "/html/body/select[1]", withTag("select"), withID("c19"), withAriaLabel("Board View"), withText("Kanban")),
		el(20, "/html/body/button[11]", withTag("button"), withID("c20"), withText("Export Board")),

		// Lead Form (c21-c30)
		el(21, "/html/body/input[3]", withTag("input"), withInputType("text"), withID("c21"), withPlaceholder("Lead Name")),
		el(22, "/html/body/input[4]", withTag("input"), withInputType("text"), withID("c22"), withPlaceholder("Company Name")),
		el(23, "/html/body/input[5]", withTag("input"), withInputType("email"), withID("c23"), withPlaceholder("Contact Email")),
		el(24, "/html/body/input[6]", withTag("input"), withInputType("tel"), withID("c24"), withPlaceholder("Phone Number")),
		el(25, "/html/body/select[2]", withTag("select"), withID("c25"), withLabel("Lead Source"), withText("Inbound")),
		el(26, "/html/body/input[7]", withTag("input"), withInputType("number"), withID("c26"), withAriaLabel("Expected Revenue ($)")),
		el(27, "/html/body/input[8]", withTag("input"), withInputType("date"), withID("c27"), withAriaLabel("Estimated Close Date")),
		el(28, "/html/body/select[3]", withTag("select"), withID("c28"), withAriaLabel("Assignee"), withText("Unassigned")),
		el(29, "/html/body/button[12]", withTag("button"), withID("c29"), withClassName("btn-primary"), withText("Save Lead")),
		el(30, "/html/body/button[13]", withTag("button"), withID("c30"), withClassName("btn-cancel"), withText("Cancel")),

		// Activity Log (c31-c40)
		el(31, "/html/body/button[14]", withTag("button"), withID("c31"), withText("Log a Call")),
		el(32, "/html/body/button[15]", withTag("button"), withID("c32"), withText("Send Email")),
		el(33, "/html/body/button[16]", withTag("button"), withID("c33"), withText("Add Note")),
		el(34, "/html/body/button[17]", withTag("button"), withID("c34"), withText("Schedule Meeting")),
		el(35, "/html/body/button[18]", withTag("button"), withID("c35"), withText("Create Task")),
		el(36, "/html/body/textarea[1]", withTag("textarea"), withID("c36"), withPlaceholder("Write your note here..."), withEditable()),
		el(37, "/html/body/button[19]", withTag("button"), withID("c37"), withText("Attach File")),
		el(38, "/html/body/button[20]", withTag("button"), withID("c38"), withText("@ Mention")),
		el(39, "/html/body/button[21]", withTag("button"), withID("c39"), withText("Save Note")),
		el(40, "/html/body/div[3]", withTag("div"), withRole("alert"), withID("c40"), withText("Note saved successfully.")),

		// Tickets (c41-c50)
		el(41, "/html/body/h2[1]", withTag("h2"), withID("c41"), withAriaLabel("Ticket Title"), withText("Server Outage"), withEditable()),
		el(42, "/html/body/select[4]", withTag("select"), withID("c42"), withLabel("Status:"), withText("Open")),
		el(43, "/html/body/select[5]", withTag("select"), withID("c43"), withLabel("Priority:"), withText("Low")),
		el(44, "/html/body/button[22]", withTag("button"), withID("c44"), withAriaLabel("Remove Bug tag"), withText("x")),
		el(45, "/html/body/input[9]", withTag("input"), withInputType("text"), withID("c45"), withPlaceholder("Add tag...")),
		el(46, "/html/body/button[23]", withTag("button"), withID("c46"), withText("Add")),
		el(47, "/html/body/button[24]", withTag("button"), withID("c47"), withText("Link Issue")),
		el(48, "/html/body/button[25]", withTag("button"), withID("c48"), withText("Clone Ticket")),
		el(49, "/html/body/button[26]", withTag("button"), withID("c49"), withAriaLabel("Watch Ticket"), withText("👀 Watch")),
		el(50, "/html/body/button[27]", withTag("button"), withID("c50"), withAriaLabel("Vote on Ticket"), withText("👍 Vote")),

		// Candidate Tracking (c51-c60)
		el(51, "/html/body/h3[1]", withTag("h3"), withID("c51"), withDataQA("candidate-name"), withText("Sarah Connor")),
		el(52, "/html/body/a[4]", withTag("a"), withID("c52"), withText("View Resume")),
		el(53, "/html/body/a[5]", withTag("a"), withID("c53"), withText("LinkedIn Profile")),
		el(54, "/html/body/select[6]", withTag("select"), withID("c54"), withAriaLabel("Stage"), withText("Screening")),
		el(55, "/html/body/button[28]", withTag("button"), withID("c55"), withText("Move to Offer")),
		el(56, "/html/body/button[29]", withTag("button"), withID("c56"), withText("Reject Candidate")),
		el(57, "/html/body/select[7]", withTag("select"), withID("c57"), withAriaLabel("Rejection Reason"), withText("Not a fit")),
		el(58, "/html/body/button[30]", withTag("button"), withID("c58"), withText("Send Rejection Email")),
		el(59, "/html/body/button[31]", withTag("button"), withID("c59"), withText("Schedule Interview")),
		el(60, "/html/body/button[32]", withTag("button"), withID("c60"), withText("Request Feedback from Team")),

		// Bulk Operations (c61-c70)
		el(61, "/html/body/input[10]", withTag("input"), withInputType("checkbox"), withID("c61"), withAriaLabel("Select All Leads")),
		el(62, "/html/body/input[11]", withTag("input"), withInputType("checkbox"), withID("c62"), withAriaLabel("Select Lead 1")),
		el(63, "/html/body/td[1]", withTag("td"), withID("c63"), withText("85")),
		el(64, "/html/body/button[33]", withTag("button"), withID("c64"), withText("Edit")),
		el(65, "/html/body/button[34]", withTag("button"), withID("c65"), withText("Bulk Assign")),
		el(66, "/html/body/button[35]", withTag("button"), withID("c66"), withText("Bulk Delete")),
		el(67, "/html/body/button[36]", withTag("button"), withID("c67"), withText("Merge Duplicates")),
		el(68, "/html/body/button[37]", withTag("button"), withID("c68"), withText("Add to Campaign")),
		el(69, "/html/body/button[38]", withTag("button"), withID("c69"), withText("Change Status")),
		el(70, "/html/body/button[39]", withTag("button"), withID("c70"), withDisabled(), withText("Apply Bulk Action")),

		// Advanced Filters (c71-c80)
		el(71, "/html/body/select[8]", withTag("select"), withID("c71"), withAriaLabel("Filter Field"), withText("Status")),
		el(72, "/html/body/select[9]", withTag("select"), withID("c72"), withAriaLabel("Filter Operator"), withText("Equals")),
		el(73, "/html/body/input[12]", withTag("input"), withInputType("text"), withID("c73"), withPlaceholder("Filter Value")),
		el(74, "/html/body/button[40]", withTag("button"), withID("c74"), withText("Add Condition")),
		el(75, "/html/body/button[41]", withTag("button"), withID("c75"), withText("Apply Filter")),
		el(76, "/html/body/button[42]", withTag("button"), withID("c76"), withAriaLabel("Remove Filter"), withText("X")),
		el(77, "/html/body/input[13]", withTag("input"), withInputType("text"), withID("c77"), withPlaceholder("Save filter as...")),
		el(78, "/html/body/button[43]", withTag("button"), withID("c78"), withText("Save Filter")),
		el(79, "/html/body/select[10]", withTag("select"), withID("c79"), withAriaLabel("Saved Filters"), withText("My Open Leads")),
		el(80, "/html/body/button[44]", withTag("button"), withID("c80"), withText("Clear All")),

		// Reports (c81-c90)
		el(81, "/html/body/select[11]", withTag("select"), withID("c81"), withAriaLabel("Report Type"), withText("Sales Funnel")),
		el(82, "/html/body/input[14]", withTag("input"), withInputType("date"), withID("c82"), withAriaLabel("Report Start Date")),
		el(83, "/html/body/input[15]", withTag("input"), withInputType("date"), withID("c83"), withAriaLabel("Report End Date")),
		el(84, "/html/body/button[45]", withTag("button"), withID("c84"), withText("Generate Report")),
		el(85, "/html/body/div[4]", withTag("div"), withID("c85"), withClassName("chart-value"), withText("Total Deals: 145")),
		el(86, "/html/body/button[46]", withTag("button"), withID("c86"), withText("Download Excel")),
		el(87, "/html/body/button[47]", withTag("button"), withID("c87"), withText("Download PDF")),
		el(88, "/html/body/button[48]", withTag("button"), withID("c88"), withText("Share via Email")),
		el(89, "/html/body/button[49]", withTag("button"), withID("c89"), withText("Schedule Report")),
		el(90, "/html/body/div[5]", withTag("div"), withRole("switch"), withID("c90"), withAriaLabel("Include deleted records"), withText("Include deleted records")),

		// Workflow Automation (c91-c100)
		el(91, "/html/body/button[50]", withTag("button"), withID("c91"), withText("Create Workflow Rule")),
		el(92, "/html/body/input[16]", withTag("input"), withInputType("text"), withID("c92"), withPlaceholder("Rule Name")),
		el(93, "/html/body/select[12]", withTag("select"), withID("c93"), withAriaLabel("Trigger Event"), withText("On Create")),
		el(94, "/html/body/select[13]", withTag("select"), withID("c94"), withAriaLabel("Action"), withText("Send Email")),
		el(95, "/html/body/button[51]", withTag("button"), withID("c95"), withText("Save Workflow")),
		el(96, "/html/body/button[52]", withTag("button"), withID("c96"), withText("Create Custom Field")),
		el(97, "/html/body/input[17]", withTag("input"), withInputType("text"), withID("c97"), withPlaceholder("Field Label")),
		el(98, "/html/body/select[14]", withTag("select"), withID("c98"), withAriaLabel("Field Type"), withText("Text")),
		el(99, "/html/body/button[53]", withTag("button"), withID("c99"), withText("Save Field")),
		el(100, "/html/body/button[54]", withTag("button"), withID("c100"), withHidden(), withText("Unlock Admin Mode")),
	}
}

func TestCRM(t *testing.T) {
	elements := crmDOM()

	tests := []struct {
		name, query, mode, expectedID string
	}{
		// Global Nav
		{"01_SearchLeads", "Search leads", "input", "c1"},
		{"02_GlobalCreate", "Global Create", "clickable", "c2"},
		{"03_Notifications", "Notifications", "clickable", "c3"},
		{"04_Pipelines", "Pipelines", "clickable", "c4"},
		{"05_Candidates", "Candidates", "clickable", "c5"},
		{"06_Reports", "Reports", "clickable", "c6"},
		{"07_RecentItems", "Recent Items", "clickable", "c7"},
		{"08_Settings", "Settings", "clickable", "c8"},
		{"09_MyProfile", "My Profile", "clickable", "c9"},
		{"10_LogOut", "Log Out", "clickable", "c10"},

		// Kanban Board
		// test 11 skipped — extract
		{"12_MoveContacted", "Move to Contacted", "clickable", "c12"},
		{"13_MoveClosedWon", "Move to Closed Won", "clickable", "c13"},
		{"14_MoveClosedLost", "Move to Closed Lost", "clickable", "c14"},
		{"15_AddColumn", "Add Column", "clickable", "c15"},
		{"16_ColumnName", "Column Name", "input", "c16"},
		{"17_SaveColumn", "Save Column", "clickable", "c17"},
		{"18_FilterBoard", "Filter Board", "clickable", "c18"},
		// test 19 → select
		{"20_ExportBoard", "Export Board", "clickable", "c20"},

		// Lead Form
		{"21_LeadName", "Lead Name", "input", "c21"},
		{"22_CompanyName", "Company Name", "input", "c22"},
		{"23_ContactEmail", "Contact Email", "input", "c23"},
		{"24_PhoneNumber", "Phone Number", "input", "c24"},
		// test 25 → select
		{"26_ExpectedRevenue", "Expected Revenue", "input", "c26"},
		{"27_EstCloseDate", "Estimated Close Date", "input", "c27"},
		// test 28 → select
		{"29_SaveLead", "Save Lead", "clickable", "c29"},
		{"30_Cancel", "Cancel", "clickable", "c30"},

		// Activity Log
		{"31_LogCall", "Log a Call", "clickable", "c31"},
		{"32_SendEmail", "Send Email", "clickable", "c32"},
		{"33_AddNote", "Add Note", "clickable", "c33"},
		{"34_ScheduleMeeting", "Schedule Meeting", "clickable", "c34"},
		{"35_CreateTask", "Create Task", "clickable", "c35"},
		{"36_WriteNote", "Write your note", "input", "c36"},
		{"37_AttachFile", "Attach File", "clickable", "c37"},
		{"38_Mention", "@ Mention", "clickable", "c38"},
		{"39_SaveNote", "Save Note", "clickable", "c39"},
		// test 40 skipped — verify

		// Tickets
		{"41_TicketTitle", "Ticket Title", "input", "c41"},
		// tests 42,43 → select
		{"44_RemoveBugTag", "Remove Bug tag", "clickable", "c44"},
		{"45_AddTag", "Add tag...", "input", "c45"},
		{"46_Add", "Add", "clickable", "c46"},
		{"47_LinkIssue", "Link Issue", "clickable", "c47"},
		{"48_CloneTicket", "Clone Ticket", "clickable", "c48"},
		{"49_WatchTicket", "Watch Ticket", "clickable", "c49"},
		{"50_VoteTicket", "Vote on Ticket", "clickable", "c50"},

		// Candidate Tracking
		// test 51 skipped — extract
		{"52_ViewResume", "View Resume", "clickable", "c52"},
		{"53_LinkedInProfile", "LinkedIn Profile", "clickable", "c53"},
		// test 54 → select
		{"55_MoveToOffer", "Move to Offer", "clickable", "c55"},
		{"56_RejectCandidate", "Reject Candidate", "clickable", "c56"},
		// test 57 → select
		{"58_SendRejection", "Send Rejection Email", "clickable", "c58"},
		{"59_ScheduleInterview", "Schedule Interview", "clickable", "c59"},
		{"60_RequestFeedback", "Request Feedback", "clickable", "c60"},

		// Bulk Operations
		{"61_SelectAllLeads", "Select All Leads", "clickable", "c61"},
		{"62_SelectLead1", "Select Lead 1", "clickable", "c62"},
		// test 63 skipped — extract
		{"64_Edit", "Edit", "clickable", "c64"},
		{"65_BulkAssign", "Bulk Assign", "clickable", "c65"},
		{"66_BulkDelete", "Bulk Delete", "clickable", "c66"},
		{"67_MergeDuplicates", "Merge Duplicates", "clickable", "c67"},
		{"68_AddToCampaign", "Add to Campaign", "clickable", "c68"},
		{"69_ChangeStatus", "Change Status", "clickable", "c69"},
		// test 70 skipped — verify (disabled)

		// Advanced Filters
		// tests 71,72 → select
		{"73_FilterValue", "Filter Value", "input", "c73"},
		{"74_AddCondition", "Add Condition", "clickable", "c74"},
		{"75_ApplyFilter", "Apply Filter", "clickable", "c75"},
		{"76_RemoveFilter", "Remove Filter", "clickable", "c76"},
		{"77_SaveFilterAs", "Save filter as", "input", "c77"},
		{"78_SaveFilter", "Save Filter", "clickable", "c78"},
		// test 79 → select
		{"80_ClearAll", "Clear All", "clickable", "c80"},

		// Reports
		// test 81 → select
		{"82_ReportStartDate", "Report Start Date", "input", "c82"},
		{"83_ReportEndDate", "Report End Date", "input", "c83"},
		{"84_GenerateReport", "Generate Report", "clickable", "c84"},
		// test 85 skipped — extract
		{"86_DownloadExcel", "Download Excel", "clickable", "c86"},
		{"87_DownloadPDF", "Download PDF", "clickable", "c87"},
		{"88_ShareEmail", "Share via Email", "clickable", "c88"},
		{"89_ScheduleReport", "Schedule Report", "clickable", "c89"},
		{"90_IncludeDeleted", "Include deleted records", "clickable", "c90"},

		// Workflow Automation
		{"91_CreateWorkflow", "Create Workflow Rule", "clickable", "c91"},
		{"92_RuleName", "Rule Name", "input", "c92"},
		// tests 93,94 → select
		{"95_SaveWorkflow", "Save Workflow", "clickable", "c95"},
		{"96_CreateCustomField", "Create Custom Field", "clickable", "c96"},
		{"97_FieldLabel", "Field Label", "input", "c97"},
		// test 98 → select
		{"99_SaveField", "Save Field", "clickable", "c99"},
		// test 100 skipped — optional/hidden
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

func TestCRM_Select(t *testing.T) {
	elements := crmDOM()

	tests := []struct {
		name, query, expectedID string
	}{
		{"19_BoardView", "Board View", "c19"},
		{"25_LeadSource", "Lead Source", "c25"},
		{"28_Assignee", "Assignee", "c28"},
		{"42_Status", "Status:", "c42"},
		{"43_Priority", "Priority:", "c43"},
		{"54_Stage", "Stage", "c54"},
		{"57_RejectionReason", "Rejection Reason", "c57"},
		{"71_FilterField", "Filter Field", "c71"},
		{"72_FilterOperator", "Filter Operator", "c72"},
		{"79_SavedFilters", "Saved Filters", "c79"},
		{"81_ReportType", "Report Type", "c81"},
		{"93_TriggerEvent", "Trigger Event", "c93"},
		{"94_Action", "Action", "c94"},
		{"98_FieldType", "Field Type", "c98"},
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

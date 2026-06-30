package synthetic

// ─────────────────────────────────────────────────────────────────────────────
// CYBERSECURITY & DEVSECOPS DOM SCORING TEST SUITE
//
// Port of ManulEngine test_11_cyber.py — 100-element cybersecurity dashboard.
// Validates: authentication, VPN, terminal, firewall, threat map, key manager,
// malware sandbox, system config, self-destruct, incident response.
// Skipped: extract (12,15,19,28,49,56,64,68,80,87), verify (6,13,22,24,37,44,
//          58,71,79,80,83,89,95,97).
// ─────────────────────────────────────────────────────────────────────────────

import (
	"testing"

	"github.com/alexbeatnik/ManulEngineGo/pkg/dom"
)

func cyberDOM() []dom.ElementSnapshot {
	return []dom.ElementSnapshot{
		// Authentication (c1-c10)
		el(1, "/html/body/input[1]", withTag("input"), withInputType("text"), withID("c1"), withPlaceholder("Username")),
		el(2, "/html/body/input[2]", withTag("input"), withInputType("password"), withID("c2"), withClassName("secure-pass"), withAriaLabel("Password")),
		el(3, "/html/body/div[1]", withTag("div"), withRole("button"), withID("c3"), withDataQA("auth-btn"), withClassName("btn-primary"), withText("Authenticate")),
		el(4, "/html/body/input[3]", withTag("input"), withInputType("text"), withID("c4"), withPlaceholder("TOTP Code")),
		el(5, "/html/body/button[1]", withTag("button"), withID("c5"), withClassName("verify-totp"), withText("Verify")),
		el(6, "/html/body/div[2]", withTag("div"), withID("c6"), withClassName("alert success"), withText("Access Granted")),
		el(7, "/html/body/button[2]", withTag("button"), withID("c7"), withAriaLabel("Biometric Login"), withText("👁️")),
		el(8, "/html/body/a[1]", withTag("a"), withID("c8"), withClassName("cancel-btn"), withText("Cancel")),
		el(9, "/html/body/select[1]", withTag("select"), withID("c9"), withAriaLabel("Role"), withText("User")),
		el(10, "/html/body/input[4]", withTag("input"), withInputType("checkbox"), withID("c10"), withLabel("Remember Device")),

		// VPN (c11-c15)
		el(11, "/html/body/button[3]", withTag("button"), withID("c11"), withClassName("vpn-init"), withText("Initiate VPN")),
		el(12, "/html/body/span[1]", withTag("span"), withClassName("ip-addr"), withText("IP Address: 192.168.0.1")),
		el(13, "/html/body/div[3]", withTag("div"), withID("c13"), withText("Connection Secure")),
		el(14, "/html/body/button[4]", withTag("button"), withID("c14"), withClassName("btn-danger"), withText("Disconnect")),
		el(15, "/html/body/span[2]", withTag("span"), withText("Status: Offline")),

		// Terminal (c16-c29)
		el(16, "/html/body/button[5]", withTag("button"), withID("c16"), withText("Open Terminal")),
		el(17, "/html/body/input[5]", withTag("input"), withInputType("text"), withID("c17"), withPlaceholder("Command Line")),
		el(18, "/html/body/div[4]", withTag("div"), withRole("button"), withID("c18"), withClassName("btn-exec"), withText("Execute Command")),
		el(19, "/html/body/div[5]", withTag("div"), withClassName("ports-list"), withText("Open Ports: 22, 80, 443")),
		el(20, "/html/body/input[6]", withTag("input"), withInputType("text"), withID("c20"), withDataTestID("target-ip"), withPlaceholder("Target IP")),
		el(21, "/html/body/button[6]", withTag("button"), withID("c21"), withClassName("scan-btn"), withText("Scan Network")),
		el(22, "/html/body/div[6]", withTag("div"), withID("c22"), withText("Vulnerability Found")),
		el(23, "/html/body/button[7]", withTag("button"), withID("c23"), withText("Clear Terminal")),
		el(24, "/html/body/div[7]", withTag("div"), withID("c24"), withText("Terminal cleared")),
		el(25, "/html/body/button[8]", withTag("button"), withID("c25"), withClassName("load-script"), withText("Load Script")),
		el(26, "/html/body/select[2]", withTag("select"), withID("c26"), withAriaLabel("Scripts"), withText("Ping.sh")),
		el(27, "/html/body/button[9]", withTag("button"), withID("c27"), withClassName("run-script"), withText("Run Script")),
		el(28, "/html/body/span[3]", withTag("span"), withText("Root Password: hunter2")),
		el(29, "/html/body/button[10]", withTag("button"), withID("c29"), withClassName("btn-close"), withText("Close Terminal")),

		// Firewall (c30-c44)
		el(30, "/html/body/button[11]", withTag("button"), withID("c30"), withText("Firewall Settings")),
		el(31, "/html/body/button[12]", withTag("button"), withID("c31"), withText("Add New Rule")),
		el(32, "/html/body/input[7]", withTag("input"), withInputType("text"), withID("c32"), withPlaceholder("Rule Name")),
		el(33, "/html/body/select[3]", withTag("select"), withID("c33"), withAriaLabel("Action"), withText("ACCEPT")),
		el(34, "/html/body/select[4]", withTag("select"), withID("c34"), withAriaLabel("Protocol"), withText("TCP")),
		el(35, "/html/body/input[8]", withTag("input"), withInputType("checkbox"), withID("c35"), withLabel("Log packets")),
		el(36, "/html/body/button[13]", withTag("button"), withID("c36"), withClassName("save-fw-rule"), withText("Save Rule")),
		el(37, "/html/body/div[8]", withTag("div"), withID("c37"), withText("Rule saved")),
		el(38, "/html/body/button[14]", withTag("button"), withID("c38"), withText("Edit Rule 5")),
		el(39, "/html/body/button[15]", withTag("button"), withID("c40"), withText("Update Rule")),
		el(40, "/html/body/button[16]", withTag("button"), withID("c41"), withText("Delete Rule 2")),
		el(41, "/html/body/button[17]", withTag("button"), withID("c42"), withClassName("confirm-del"), withText("Confirm Deletion")),
		el(42, "/html/body/button[18]", withTag("button"), withID("c43"), withClassName("btn-mega-danger"), withText("Flush All Rules")),
		el(43, "/html/body/div[9]", withTag("div"), withID("c44"), withText("Firewall empty")),

		// Threat Map (c45-c59)
		el(44, "/html/body/button[19]", withTag("button"), withID("c45"), withText("Threat Map")),
		el(45, "/html/body/button[20]", withTag("button"), withID("c46"), withText("Region: Eastern Europe")),
		el(46, "/html/body/button[21]", withTag("button"), withID("c47"), withAriaLabel("Zoom In"), withText("+")),
		el(47, "/html/body/button[22]", withTag("button"), withID("c48"), withAriaLabel("Zoom Out"), withText("-")),
		el(48, "/html/body/span[4]", withTag("span"), withText("Active Threats: 9000")),
		el(49, "/html/body/input[9]", withTag("input"), withInputType("checkbox"), withID("c50"), withLabel("Show Botnets")),
		el(50, "/html/body/input[10]", withTag("input"), withInputType("checkbox"), withID("c51"), withLabel("Show Phishing")),
		el(51, "/html/body/button[23]", withTag("button"), withID("c52"), withClassName("export-btn"), withText("Export Threat Data")),
		el(52, "/html/body/select[5]", withTag("select"), withID("c53"), withAriaLabel("Export Format"), withText("CSV")),
		el(53, "/html/body/button[24]", withTag("button"), withID("c54"), withText("Download")),
		el(54, "/html/body/button[25]", withTag("button"), withID("c55"), withText("Node 404")),
		el(55, "/html/body/div[10]", withTag("div"), withText("Malware Family: Ransomware")),
		el(56, "/html/body/button[26]", withTag("button"), withID("c57"), withText("Isolate Node")),
		el(57, "/html/body/div[11]", withTag("div"), withID("c58"), withText("Node isolated")),
		el(58, "/html/body/button[27]", withTag("button"), withID("c59"), withText("Close Map")),

		// Key Manager (c60-c74)
		el(59, "/html/body/button[28]", withTag("button"), withID("c60"), withText("Key Manager")),
		el(60, "/html/body/button[29]", withTag("button"), withID("c61"), withText("Generate RSA Key")),
		el(61, "/html/body/select[6]", withTag("select"), withID("c62"), withAriaLabel("Key Size"), withText("2048 bit")),
		el(62, "/html/body/button[30]", withTag("button"), withID("c63"), withText("Generate")),
		el(63, "/html/body/div[12]", withTag("div"), withText("Public Key: ssh-rsa AAAAB3Nza...")),
		el(64, "/html/body/button[31]", withTag("button"), withID("c65"), withText("Copy Private Key")),
		el(65, "/html/body/input[11]", withTag("input"), withInputType("text"), withID("c66"), withPlaceholder("Encrypt Message")),
		el(66, "/html/body/button[32]", withTag("button"), withID("c67"), withText("Encrypt")),
		el(67, "/html/body/div[13]", withTag("div"), withText("Ciphertext: 0x8f7a6b5c")),
		el(68, "/html/body/input[12]", withTag("input"), withInputType("text"), withID("c69"), withPlaceholder("Decrypt Message")),
		el(69, "/html/body/button[33]", withTag("button"), withID("c70"), withText("Decrypt")),
		el(70, "/html/body/div[14]", withTag("div"), withID("c71"), withText("Invalid Padding")),
		el(71, "/html/body/button[34]", withTag("button"), withID("c72"), withText("Revoke Key")),
		el(72, "/html/body/input[13]", withTag("input"), withInputType("text"), withID("c73"), withPlaceholder("Reason")),
		el(73, "/html/body/button[35]", withTag("button"), withID("c74"), withText("Confirm Revocation")),

		// Malware Sandbox (c75-c83)
		el(74, "/html/body/button[36]", withTag("button"), withID("c75"), withText("Malware Sandbox")),
		el(75, "/html/body/button[37]", withTag("button"), withID("c76"), withText("Upload Executable")),
		el(76, "/html/body/input[14]", withTag("input"), withInputType("text"), withID("c77"), withPlaceholder("File URL")),
		el(77, "/html/body/button[38]", withTag("button"), withID("c78"), withText("Analyze")),
		el(78, "/html/body/div[15]", withTag("div"), withID("c79"), withText("Analysis in progress")),
		el(79, "/html/body/div[16]", withTag("div"), withText("Threat Score: 99/100")),
		el(80, "/html/body/button[39]", withTag("button"), withID("c81"), withText("View Process Tree")),
		el(81, "/html/body/button[40]", withTag("button"), withID("c82"), withText("Kill Process")),
		el(82, "/html/body/div[17]", withTag("div"), withID("c83"), withText("Process Terminated")),

		// System Config (c84-c88)
		el(83, "/html/body/button[41]", withTag("button"), withID("c84"), withText("System Config")),
		el(84, "/html/body/input[15]", withTag("input"), withInputType("checkbox"), withID("c85"), withLabel("Strict Mode")),
		el(85, "/html/body/button[42]", withTag("button"), withID("c86"), withText("Enable Honeypot")),
		el(86, "/html/body/div[18]", withTag("div"), withText("Decoy IP: 10.0.0.5")),
		el(87, "/html/body/button[43]", withTag("button"), withID("c88"), withText("Advanced Options")),

		// Self-destruct & safety (c89-c97)
		el(88, "/html/body/button[44]", withTag("button"), withID("c93"), withText("Initiate Self-Destruct")),
		el(89, "/html/body/button[45]", withTag("button"), withID("c90"), withText("Unlock Safety Protocol")),
		el(90, "/html/body/input[16]", withTag("input"), withInputType("text"), withID("c91"), withPlaceholder("Override Code")),
		el(91, "/html/body/input[17]", withTag("input"), withInputType("text"), withID("c92"), withPlaceholder("Confirm Destruction")),
		el(92, "/html/body/button[46]", withTag("button"), withID("c94"), withText("ABORT")),
		el(93, "/html/body/div[19]", withTag("div"), withID("c95"), withText("Sequence Aborted")),
		el(94, "/html/body/button[47]", withTag("button"), withID("c96"), withText("Wipe Logs")),
		el(95, "/html/body/div[20]", withTag("div"), withID("c97"), withText("Logs wiped")),

		// Final actions (c98-c100)
		el(96, "/html/body/button[48]", withTag("button"), withID("c98"), withText("Lockdown Network")),
		el(97, "/html/body/button[49]", withTag("button"), withID("c99"), withText("Contact Incident Response")),
		el(98, "/html/body/button[50]", withTag("button"), withID("c100"), withText("Logout")),
	}
}

func TestCyber(t *testing.T) {
	elements := cyberDOM()

	tests := []struct {
		name       string
		query      string
		mode       string
		expectedID string
	}{
		// Authentication
		{"Fill Username", "Username", "input", "c1"},
		{"Fill Password", "Password", "input", "c2"},
		{"Click Authenticate", "Authenticate", "clickable", "c3"},
		{"Fill TOTP Code", "TOTP Code", "input", "c4"},
		{"Click Verify", "Verify", "clickable", "c5"},
		{"Click Biometric Login", "Biometric Login", "clickable", "c7"},
		{"Click Cancel", "Cancel", "clickable", "c8"},
		{"Check Remember Device", "Remember Device", "clickable", "c10"},
		// VPN
		{"Click Initiate VPN", "Initiate VPN", "clickable", "c11"},
		{"Click Disconnect", "Disconnect", "clickable", "c14"},
		// Terminal
		{"Click Open Terminal", "Open Terminal", "clickable", "c16"},
		{"Fill Command Line", "Command Line", "input", "c17"},
		{"Click Execute Command", "Execute Command", "clickable", "c18"},
		{"Fill Target IP", "Target IP", "input", "c20"},
		{"Click Scan Network", "Scan Network", "clickable", "c21"},
		{"Click Clear Terminal", "Clear Terminal", "clickable", "c23"},
		{"Click Load Script", "Load Script", "clickable", "c25"},
		{"Click Run Script", "Run Script", "clickable", "c27"},
		{"Click Close Terminal", "Close Terminal", "clickable", "c29"},
		// Firewall
		{"Click Firewall Settings", "Firewall Settings", "clickable", "c30"},
		{"Click Add New Rule", "Add New Rule", "clickable", "c31"},
		{"Fill Rule Name", "Rule Name", "input", "c32"},
		{"Check Log packets", "Log packets", "clickable", "c35"},
		{"Click Save Rule", "Save Rule", "clickable", "c36"},
		{"Click Edit Rule 5", "Edit Rule 5", "clickable", "c38"},
		{"Click Update Rule", "Update Rule", "clickable", "c40"},
		{"Click Delete Rule 2", "Delete Rule 2", "clickable", "c41"},
		{"Click Confirm Deletion", "Confirm Deletion", "clickable", "c42"},
		{"Click Flush All Rules", "Flush All Rules", "clickable", "c43"},
		// Threat Map
		{"Click Threat Map", "Threat Map", "clickable", "c45"},
		{"Click Region Eastern Europe", "Region: Eastern Europe", "clickable", "c46"},
		{"Click Zoom In", "Zoom In", "clickable", "c47"},
		{"Click Zoom Out", "Zoom Out", "clickable", "c48"},
		{"Check Show Botnets", "Show Botnets", "clickable", "c50"},
		{"Uncheck Show Phishing", "Show Phishing", "clickable", "c51"},
		{"Click Export Threat Data", "Export Threat Data", "clickable", "c52"},
		{"Click Download", "Download", "clickable", "c54"},
		{"Click Node 404", "Node 404", "clickable", "c55"},
		{"Click Isolate Node", "Isolate Node", "clickable", "c57"},
		{"Click Close Map", "Close Map", "clickable", "c59"},
		// Key Manager
		{"Click Key Manager", "Key Manager", "clickable", "c60"},
		{"Click Generate RSA Key", "Generate RSA Key", "clickable", "c61"},
		{"Click Generate", "Generate", "clickable", "c63"},
		{"Click Copy Private Key", "Copy Private Key", "clickable", "c65"},
		{"Fill Encrypt Message", "Encrypt Message", "input", "c66"},
		{"Click Encrypt", "Encrypt", "clickable", "c67"},
		{"Fill Decrypt Message", "Decrypt Message", "input", "c69"},
		{"Click Decrypt", "Decrypt", "clickable", "c70"},
		{"Click Revoke Key", "Revoke Key", "clickable", "c72"},
		{"Fill Reason", "Reason", "input", "c73"},
		{"Click Confirm Revocation", "Confirm Revocation", "clickable", "c74"},
		// Malware Sandbox
		{"Click Malware Sandbox", "Malware Sandbox", "clickable", "c75"},
		{"Click Upload Executable", "Upload Executable", "clickable", "c76"},
		{"Fill File URL", "File URL", "input", "c77"},
		{"Click Analyze", "Analyze", "clickable", "c78"},
		{"Click View Process Tree", "View Process Tree", "clickable", "c81"},
		{"Click Kill Process", "Kill Process", "clickable", "c82"},
		// System Config
		{"Click System Config", "System Config", "clickable", "c84"},
		{"Check Strict Mode", "Strict Mode", "clickable", "c85"},
		{"Click Enable Honeypot", "Enable Honeypot", "clickable", "c86"},
		{"Click Advanced Options", "Advanced Options", "clickable", "c88"},
		// Self-destruct
		{"Click Unlock Safety Protocol", "Unlock Safety Protocol", "clickable", "c90"},
		{"Fill Override Code", "Override Code", "input", "c91"},
		{"Fill Confirm Destruction", "Confirm Destruction", "input", "c92"},
		{"Click Initiate Self-Destruct", "Initiate Self-Destruct", "clickable", "c93"},
		{"Click ABORT", "ABORT", "clickable", "c94"},
		{"Click Wipe Logs", "Wipe Logs", "clickable", "c96"},
		// Final
		{"Click Lockdown Network", "Lockdown Network", "clickable", "c98"},
		{"Click Contact Incident Response", "Contact Incident Response", "clickable", "c99"},
		{"Click Logout", "Logout", "clickable", "c100"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := rankFirstID(t, tc.query, "", tc.mode, elements)
			if got != tc.expectedID {
				t.Errorf("query=%q mode=%s → got %s, want %s", tc.query, tc.mode, got, tc.expectedID)
			}
		})
	}
}

func TestCyber_Select(t *testing.T) {
	elements := cyberDOM()

	tests := []struct {
		name       string
		query      string
		expectedID string
	}{
		{"Select Role", "Role", "c9"},
		{"Select Scripts", "Scripts", "c26"},
		{"Select Action", "Action", "c33"},
		{"Select Protocol", "Protocol", "c34"},
		{"Select Export Format", "Export Format", "c53"},
		{"Select Key Size", "Key Size", "c62"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := rankFirstID(t, tc.query, "", "select", elements)
			if got != tc.expectedID {
				t.Errorf("query=%q mode=select → got %s, want %s", tc.query, got, tc.expectedID)
			}
		})
	}
}

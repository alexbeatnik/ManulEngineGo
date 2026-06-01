package synthetic

// ─────────────────────────────────────────────────────────────────────────────
// EDTECH & E-LEARNING DOM SCORING TEST SUITE
//
// Port of ManulEngine test_09_edtech.py — 100-element EdTech/e-learning page.
// Validates: course browsing, lecture player, quizzes, code editor,
// discussion forum, assignments, grades, peer review, instructor tools,
// gamification.
// Skipped: extract (29,35,51,61,63,66,92,96), verify (28,60,80),
//          optional/hidden (100).
// ─────────────────────────────────────────────────────────────────────────────

import (
	"testing"

	"github.com/alexbeatnik/ManulHeart/pkg/dom"
)

func edtechDOM() []dom.ElementSnapshot {
	return []dom.ElementSnapshot{
		// Course Browsing (l1-l10)
		el(1, "/html/body/input[1]", withTag("input"), withInputType("search"), withID("l1"), withPlaceholder("Search courses, skills, and videos")),
		el(2, "/html/body/select[1]", withTag("select"), withID("l2"), withAriaLabel("Category"), withText("Computer Science")),
		el(3, "/html/body/button[1]", withTag("button"), withID("l3"), withClassName("btn-enroll"), withText("Enroll for Free")),
		el(4, "/html/body/a[1]", withTag("a"), withID("l4"), withText("View Syllabus")),
		el(5, "/html/body/button[2]", withTag("button"), withID("l5"), withAriaLabel("Bookmark Course"), withText("🔖")),
		el(6, "/html/body/input[2]", withTag("input"), withInputType("radio"), withID("l6"), withLabel("Beginner"), withNameAttr("lvl")),
		el(7, "/html/body/input[3]", withTag("input"), withInputType("radio"), withID("l7"), withLabel("Advanced"), withNameAttr("lvl")),
		el(8, "/html/body/button[3]", withTag("button"), withID("l8"), withText("Apply Filters")),
		el(9, "/html/body/button[4]", withTag("button"), withID("l9"), withText("Clear All")),
		el(10, "/html/body/div[1]", withTag("div"), withRole("button"), withID("l10"), withText("Instructor Bio ▼")),

		// Lecture Player (l11-l20)
		el(11, "/html/body/button[5]", withTag("button"), withID("l11"), withText("Previous Lecture")),
		el(12, "/html/body/button[6]", withTag("button"), withID("l12"), withClassName("next-btn"), withText("Next Lecture")),
		el(13, "/html/body/div[2]", withTag("div"), withRole("switch"), withID("l13"), withAriaLabel("Auto-advance"), withText("Auto-advance")),
		el(14, "/html/body/button[7]", withTag("button"), withID("l14"), withText("Show Transcript")),
		el(15, "/html/body/button[8]", withTag("button"), withID("l15"), withText("Take Notes")),
		el(16, "/html/body/select[2]", withTag("select"), withID("l16"), withAriaLabel("Speed"), withText("1x")),
		el(17, "/html/body/input[4]", withTag("input"), withInputType("checkbox"), withID("l17"), withLabel("Mark as Complete")),
		el(18, "/html/body/a[2]", withTag("a"), withID("l18"), withText("Download Slides")),
		el(19, "/html/body/button[9]", withTag("button"), withID("l19"), withText("Ask Question in Q&A")),
		el(20, "/html/body/button[10]", withTag("button"), withID("l20"), withText("Report Audio Issue")),

		// Quizzes (l21-l30)
		el(21, "/html/body/input[5]", withTag("input"), withInputType("radio"), withID("l21"), withLabel("A snake"), withNameAttr("q1")),
		el(22, "/html/body/input[6]", withTag("input"), withInputType("radio"), withID("l22"), withLabel("A programming language"), withNameAttr("q1")),
		el(23, "/html/body/input[7]", withTag("input"), withInputType("checkbox"), withID("l23"), withLabel("div")),
		el(24, "/html/body/input[8]", withTag("input"), withInputType("checkbox"), withID("l24"), withLabel("span")),
		el(25, "/html/body/input[9]", withTag("input"), withInputType("checkbox"), withID("l25"), withLabel("fakeTag")),
		el(26, "/html/body/button[11]", withTag("button"), withID("l26"), withText("Clear my choices")),
		el(27, "/html/body/button[12]", withTag("button"), withID("l27"), withClassName("btn-submit"), withText("Submit Quiz")),
		el(28, "/html/body/button[13]", withTag("button"), withID("l28"), withDisabled(), withText("Review Answers")),
		el(29, "/html/body/div[3]", withTag("div"), withID("l29"), withClassName("quiz-score"), withText("Score: 85%")),
		el(30, "/html/body/button[14]", withTag("button"), withID("l30"), withText("Retake Quiz")),

		// Code Editor (l31-l40)
		el(31, "/html/body/input[10]", withTag("input"), withInputType("text"), withID("l31"), withPlaceholder("def")),
		el(32, "/html/body/textarea[1]", withTag("textarea"), withID("l32"), withClassName("code-editor"), withAriaLabel("Code Editor"), withText("print(\"Hello\")"), withEditable()),
		el(33, "/html/body/button[15]", withTag("button"), withID("l33"), withText("Run Code")),
		el(34, "/html/body/button[16]", withTag("button"), withID("l34"), withClassName("btn-success"), withText("Submit Code")),
		el(35, "/html/body/div[4]", withTag("div"), withID("l35"), withClassName("console-output"), withText("SyntaxError: invalid syntax")),
		el(36, "/html/body/button[17]", withTag("button"), withID("l36"), withAriaLabel("Show Hint 1"), withText("💡 Hint")),
		el(37, "/html/body/button[18]", withTag("button"), withID("l37"), withText("View Solution")),
		el(38, "/html/body/div[5]", withTag("div"), withRole("button"), withID("l38"), withText("Reset Workspace")),
		el(39, "/html/body/input[11]", withTag("input"), withInputType("file"), withID("l39"), withAriaLabel("Upload Source File")),
		el(40, "/html/body/select[3]", withTag("select"), withID("l40"), withAriaLabel("Language"), withText("Python 3")),

		// Discussion Forum (l41-l50)
		el(41, "/html/body/input[12]", withTag("input"), withInputType("text"), withID("l41"), withPlaceholder("Search discussions...")),
		el(42, "/html/body/button[19]", withTag("button"), withID("l42"), withText("Create New Thread")),
		el(43, "/html/body/input[13]", withTag("input"), withInputType("text"), withID("l43"), withPlaceholder("Thread Title")),
		el(44, "/html/body/textarea[2]", withTag("textarea"), withID("l44"), withPlaceholder("Type your question here..."), withEditable()),
		el(45, "/html/body/button[20]", withTag("button"), withID("l45"), withText("Post Thread")),
		el(46, "/html/body/button[21]", withTag("button"), withID("l46"), withAriaLabel("Upvote Post"), withText("⬆")),
		el(47, "/html/body/button[22]", withTag("button"), withID("l47"), withAriaLabel("Downvote Post"), withText("⬇")),
		el(48, "/html/body/button[23]", withTag("button"), withID("l48"), withText("Reply to Thread")),
		el(49, "/html/body/button[24]", withTag("button"), withID("l49"), withText("Mark as Answer")),
		el(50, "/html/body/button[25]", withTag("button"), withID("l50"), withText("Subscribe to updates")),

		// Assignments (l51-l60)
		el(51, "/html/body/h2[1]", withTag("h2"), withID("l51"), withText("Final Project: Web Scraper")),
		el(52, "/html/body/input[14]", withTag("input"), withInputType("file"), withID("l52"), withAriaLabel("Upload Assignment")),
		el(53, "/html/body/input[15]", withTag("input"), withInputType("url"), withID("l53"), withPlaceholder("Paste Google Drive Link")),
		el(54, "/html/body/textarea[3]", withTag("textarea"), withID("l54"), withPlaceholder("Add comments for grader"), withEditable()),
		el(55, "/html/body/input[16]", withTag("input"), withInputType("checkbox"), withID("l55"), withLabel("I agree to the Honor Code")),
		el(56, "/html/body/button[26]", withTag("button"), withID("l56"), withClassName("btn-primary"), withText("Submit Assignment")),
		el(57, "/html/body/button[27]", withTag("button"), withID("l57"), withText("Withdraw Submission")),
		el(58, "/html/body/div[6]", withTag("div"), withRole("button"), withID("l58"), withText("View Grading Rubric")),
		el(59, "/html/body/button[28]", withTag("button"), withID("l59"), withText("Request Extension")),
		el(60, "/html/body/div[7]", withTag("div"), withID("l60"), withClassName("status-badge"), withText("Status: Not Submitted")),

		// Grades (l61-l70)
		el(61, "/html/body/div[8]", withTag("div"), withID("l61"), withClassName("final-grade"), withText("Final Grade: A-")),
		el(62, "/html/body/button[29]", withTag("button"), withID("l62"), withText("Download Transcript")),
		el(63, "/html/body/div[9]", withTag("div"), withID("l63"), withClassName("progress-text"), withText("Course Progress: 75%")),
		el(64, "/html/body/button[30]", withTag("button"), withID("l64"), withText("Request Regrade")),
		el(65, "/html/body/select[4]", withTag("select"), withID("l65"), withAriaLabel("Semester"), withText("Fall 2025")),
		el(66, "/html/body/div[10]", withTag("div"), withID("l66"), withText("GPA: 3.8")),
		el(67, "/html/body/button[31]", withTag("button"), withID("l67"), withText("Share Grades")),
		el(68, "/html/body/div[11]", withTag("div"), withRole("button"), withID("l68"), withText("View Weighting")),
		el(69, "/html/body/input[17]", withTag("input"), withInputType("text"), withID("l69"), withPlaceholder("Enter target grade to calculate needed score")),
		el(70, "/html/body/button[32]", withTag("button"), withID("l70"), withText("Calculate")),

		// Peer Review (l71-l80)
		el(71, "/html/body/button[33]", withTag("button"), withID("l71"), withText("Start Reviewing")),
		el(72, "/html/body/select[5]", withTag("select"), withID("l72"), withAriaLabel("Criterion 1 Score"), withText("0")),
		el(73, "/html/body/select[6]", withTag("select"), withID("l73"), withAriaLabel("Criterion 2 Score"), withText("0")),
		el(74, "/html/body/textarea[4]", withTag("textarea"), withID("l74"), withPlaceholder("Constructive Feedback"), withEditable()),
		el(75, "/html/body/button[34]", withTag("button"), withID("l75"), withText("Submit Review")),
		el(76, "/html/body/button[35]", withTag("button"), withID("l76"), withText("Flag for Plagiarism")),
		el(77, "/html/body/button[36]", withTag("button"), withID("l77"), withText("Save Draft Review")),
		el(78, "/html/body/div[12]", withTag("div"), withRole("button"), withID("l78"), withText("View Student Submission")),
		el(79, "/html/body/div[13]", withTag("div"), withRole("button"), withID("l79"), withText("Show Rubric Guidelines")),
		el(80, "/html/body/button[37]", withTag("button"), withID("l80"), withDisabled(), withText("Next Submission")),

		// Instructor Tools (l81-l90)
		el(81, "/html/body/button[38]", withTag("button"), withID("l81"), withText("Create New Course")),
		el(82, "/html/body/button[39]", withTag("button"), withID("l82"), withText("Edit Syllabus")),
		el(83, "/html/body/button[40]", withTag("button"), withID("l83"), withText("Post Announcement")),
		el(84, "/html/body/input[18]", withTag("input"), withInputType("text"), withID("l84"), withPlaceholder("Message to Students")),
		el(85, "/html/body/button[41]", withTag("button"), withID("l85"), withText("Send Message")),
		el(86, "/html/body/a[3]", withTag("a"), withID("l86"), withText("Open Gradebook")),
		el(87, "/html/body/button[42]", withTag("button"), withID("l87"), withClassName("publish-btn"), withText("Publish Grades")),
		el(88, "/html/body/button[43]", withTag("button"), withID("l88"), withText("Export Analytics")),
		el(89, "/html/body/div[14]", withTag("div"), withRole("switch"), withID("l89"), withAriaLabel("Accepting Enrollments"), withText("Accepting Enrollments")),
		el(90, "/html/body/button[44]", withTag("button"), withID("l90"), withText("Unpublish Course")),

		// Gamification (l91-l100)
		el(91, "/html/body/button[45]", withTag("button"), withID("l91"), withClassName("pulse"), withText("Claim Daily Reward")),
		el(92, "/html/body/div[15]", withTag("div"), withID("l92"), withText("Current Streak: 14 Days")),
		el(93, "/html/body/button[46]", withTag("button"), withID("l93"), withText("View Certificate")),
		el(94, "/html/body/button[47]", withTag("button"), withID("l94"), withText("Share to LinkedIn")),
		el(95, "/html/body/button[48]", withTag("button"), withID("l95"), withText("Download PDF Certificate")),
		el(96, "/html/body/div[16]", withTag("div"), withID("l96"), withText("Rank: #4 in Class")),
		el(97, "/html/body/button[49]", withTag("button"), withID("l97"), withAriaLabel("View Leaderboard"), withText("🏆")),
		el(98, "/html/body/button[50]", withTag("button"), withID("l98"), withText("Equip Avatar Frame")),
		el(99, "/html/body/button[51]", withTag("button"), withID("l99"), withText("Spend XP Points")),
		el(100, "/html/body/button[52]", withTag("button"), withID("l100"), withHidden(), withText("Secret Dev Mode")),
	}
}

func TestEdtech(t *testing.T) {
	elements := edtechDOM()

	tests := []struct {
		name, query, mode, expectedID string
	}{
		// Course Browsing
		{"01_SearchCourses", "Search courses", "input", "l1"},
		// test 2 → select
		{"03_EnrollFree", "Enroll for Free", "clickable", "l3"},
		{"04_ViewSyllabus", "View Syllabus", "clickable", "l4"},
		{"05_BookmarkCourse", "Bookmark Course", "clickable", "l5"},
		{"06_Beginner", "Beginner", "clickable", "l6"},
		{"07_Advanced", "Advanced", "clickable", "l7"},
		{"08_ApplyFilters", "Apply Filters", "clickable", "l8"},
		{"09_ClearAll", "Clear All", "clickable", "l9"},
		{"10_InstructorBio", "Instructor Bio", "clickable", "l10"},

		// Lecture Player
		{"11_PrevLecture", "Previous Lecture", "clickable", "l11"},
		{"12_NextLecture", "Next Lecture", "clickable", "l12"},
		{"13_AutoAdvance", "Auto-advance", "clickable", "l13"},
		{"14_ShowTranscript", "Show Transcript", "clickable", "l14"},
		{"15_TakeNotes", "Take Notes", "clickable", "l15"},
		// test 16 → select
		{"17_MarkComplete", "Mark as Complete", "clickable", "l17"},
		{"18_DownloadSlides", "Download Slides", "clickable", "l18"},
		{"19_AskQuestion", "Ask Question in Q&A", "clickable", "l19"},
		{"20_ReportAudio", "Report Audio Issue", "clickable", "l20"},

		// Quizzes
		{"21_ASnake", "A snake", "clickable", "l21"},
		{"22_AProgrammingLang", "A programming language", "clickable", "l22"},
		{"23_DivCheckbox", "div", "clickable", "l23"},
		{"24_SpanCheckbox", "span", "clickable", "l24"},
		{"25_FakeTag", "fakeTag", "clickable", "l25"},
		{"26_ClearChoices", "Clear my choices", "clickable", "l26"},
		{"27_SubmitQuiz", "Submit Quiz", "clickable", "l27"},
		// test 28 skipped — verify (disabled)
		// test 29 skipped — extract
		{"30_RetakeQuiz", "Retake Quiz", "clickable", "l30"},

		// Code Editor
		{"31_DefField", "def", "input", "l31"},
		{"32_CodeEditor", "Code Editor", "input", "l32"},
		{"33_RunCode", "Run Code", "clickable", "l33"},
		{"34_SubmitCode", "Submit Code", "clickable", "l34"},
		// test 35 skipped — extract
		{"36_Hint", "Hint", "clickable", "l36"},
		{"37_ViewSolution", "View Solution", "clickable", "l37"},
		{"38_ResetWorkspace", "Reset Workspace", "clickable", "l38"},
		{"39_UploadSource", "Upload Source File", "clickable", "l39"},
		// test 40 → select

		// Discussion Forum
		{"41_SearchDiscussions", "Search discussions", "input", "l41"},
		{"42_CreateThread", "Create New Thread", "clickable", "l42"},
		{"43_ThreadTitle", "Thread Title", "input", "l43"},
		{"44_TypeQuestion", "Type your question here", "input", "l44"},
		{"45_PostThread", "Post Thread", "clickable", "l45"},
		{"46_UpvotePost", "Upvote Post", "clickable", "l46"},
		{"47_DownvotePost", "Downvote Post", "clickable", "l47"},
		{"48_ReplyThread", "Reply to Thread", "clickable", "l48"},
		{"49_MarkAnswer", "Mark as Answer", "clickable", "l49"},
		{"50_Subscribe", "Subscribe to updates", "clickable", "l50"},

		// Assignments
		// test 51 skipped — extract
		{"52_UploadAssignment", "Upload Assignment", "clickable", "l52"},
		{"53_GoogleDriveLink", "Google Drive Link", "input", "l53"},
		{"54_CommentsGrader", "Add comments for grader", "input", "l54"},
		{"55_HonorCode", "Honor Code", "clickable", "l55"},
		{"56_SubmitAssignment", "Submit Assignment", "clickable", "l56"},
		{"57_WithdrawSubmission", "Withdraw Submission", "clickable", "l57"},
		{"58_ViewGradingRubric", "View Grading Rubric", "clickable", "l58"},
		{"59_RequestExtension", "Request Extension", "clickable", "l59"},
		// test 60 skipped — verify

		// Grades
		// test 61 skipped — extract
		{"62_DownloadTranscript", "Download Transcript", "clickable", "l62"},
		// test 63 skipped — extract
		{"64_RequestRegrade", "Request Regrade", "clickable", "l64"},
		// test 65 → select
		// test 66 skipped — extract
		{"67_ShareGrades", "Share Grades", "clickable", "l67"},
		{"68_ViewWeighting", "View Weighting", "clickable", "l68"},
		{"69_TargetGrade", "Enter target grade", "input", "l69"},
		{"70_Calculate", "Calculate", "clickable", "l70"},

		// Peer Review
		{"71_StartReviewing", "Start Reviewing", "clickable", "l71"},
		// tests 72,73 → select
		{"74_ConstructiveFeedback", "Constructive Feedback", "input", "l74"},
		{"75_SubmitReview", "Submit Review", "clickable", "l75"},
		{"76_FlagPlagiarism", "Flag for Plagiarism", "clickable", "l76"},
		{"77_SaveDraftReview", "Save Draft Review", "clickable", "l77"},
		{"78_ViewStudentSubmission", "View Student Submission", "clickable", "l78"},
		{"79_ShowRubricGuidelines", "Show Rubric Guidelines", "clickable", "l79"},
		// test 80 skipped — verify (disabled)

		// Instructor Tools
		{"81_CreateNewCourse", "Create New Course", "clickable", "l81"},
		{"82_EditSyllabus", "Edit Syllabus", "clickable", "l82"},
		{"83_PostAnnouncement", "Post Announcement", "clickable", "l83"},
		{"84_MessageStudents", "Message to Students", "input", "l84"},
		{"85_SendMessage", "Send Message", "clickable", "l85"},
		{"86_OpenGradebook", "Open Gradebook", "clickable", "l86"},
		{"87_PublishGrades", "Publish Grades", "clickable", "l87"},
		{"88_ExportAnalytics", "Export Analytics", "clickable", "l88"},
		{"89_AcceptingEnrollments", "Accepting Enrollments", "clickable", "l89"},
		{"90_UnpublishCourse", "Unpublish Course", "clickable", "l90"},

		// Gamification
		{"91_ClaimDailyReward", "Claim Daily Reward", "clickable", "l91"},
		// test 92 skipped — extract
		{"93_ViewCertificate", "View Certificate", "clickable", "l93"},
		{"94_ShareLinkedIn", "Share to LinkedIn", "clickable", "l94"},
		{"95_DownloadPDFCert", "Download PDF Certificate", "clickable", "l95"},
		// test 96 skipped — extract
		{"97_ViewLeaderboard", "View Leaderboard", "clickable", "l97"},
		{"98_EquipAvatarFrame", "Equip Avatar Frame", "clickable", "l98"},
		{"99_SpendXP", "Spend XP Points", "clickable", "l99"},
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

func TestEdtech_Select(t *testing.T) {
	elements := edtechDOM()

	tests := []struct {
		name, query, expectedID string
	}{
		{"02_Category", "Category", "l2"},
		{"16_Speed", "Speed", "l16"},
		{"40_Language", "Language", "l40"},
		{"65_Semester", "Semester", "l65"},
		{"72_Criterion1", "Criterion 1 Score", "l72"},
		{"73_Criterion2", "Criterion 2 Score", "l73"},
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

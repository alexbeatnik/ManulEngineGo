package synthetic

// ─────────────────────────────────────────────────────────────────────────────
// SOCIAL MEDIA DOM SCORING TEST SUITE
//
// Port of ManulEngine test_02_social_media.py — 100-element social/messenger page.
// Validates: feed interactions, commenting, DMs, network, profile editing,
// privacy settings, notifications, search, groups/events, stories.
// Skipped: tests 38,39,62 (extract).
// ─────────────────────────────────────────────────────────────────────────────

import (
	"testing"

	"github.com/alexbeatnik/ManulEngineGo/pkg/dom"
)

func socialDOM() []dom.ElementSnapshot {
	return []dom.ElementSnapshot{
		// Feed Interactions (s1-s10)
		el(1, "/html/body/button[1]", withTag("button"), withID("s1"), withAriaLabel("Like Post")),
		el(2, "/html/body/div[1]", withTag("div"), withRole("button"), withID("s2"), withAriaLabel("Unlike"), withText("❤️")),
		el(3, "/html/body/button[2]", withTag("button"), withID("s3"), withClassName("repost-btn"), withText("Repost")),
		el(4, "/html/body/button[3]", withTag("button"), withID("s4"), withAccessibleName("Share via Direct Message"), withText("Share via Direct Message")),
		el(5, "/html/body/div[2]", withTag("div"), withRole("button"), withID("s5"), withAriaLabel("Save to Bookmarks"), withText("🔖")),
		el(6, "/html/body/a[1]", withTag("a"), withID("s6"), withText("Report this post")),
		el(7, "/html/body/button[4]", withTag("button"), withID("s7"), withText("Hide")),
		el(8, "/html/body/span[1]", withTag("span"), withRole("button"), withID("s8"), withAriaLabel("More Options"), withText("•••")),
		el(9, "/html/body/button[5]", withTag("button"), withID("s9"), withDataTestID("translate-tweet"), withText("Translate post")),
		el(10, "/html/body/div[3]", withTag("div"), withID("s10"), withText("View 15 hidden replies")),

		// Commenting (s11-s20)
		el(11, "/html/body/div[4]", withTag("div"), withID("s11"), withEditable(), withAriaLabel("Write a comment..."), withClassName("chat-input")),
		el(12, "/html/body/button[6]", withTag("button"), withID("s12"), withText("Post Comment"), withDisabled()),
		el(13, "/html/body/button[7]", withTag("button"), withID("s13"), withText("Reply")),
		el(14, "/html/body/input[1]", withTag("input"), withInputType("text"), withID("s14"), withPlaceholder("Mention someone (@)")),
		el(15, "/html/body/div[5]/div[1]", withTag("div"), withRole("option"), withID("s15"), withText("@alex_dev")),
		el(16, "/html/body/div[5]/div[2]", withTag("div"), withRole("option"), withID("s16"), withText("@manul_qa")),
		el(17, "/html/body/button[8]", withTag("button"), withID("s17"), withAriaLabel("Insert Emoji"), withText("😀")),
		el(18, "/html/body/input[2]", withTag("input"), withInputType("file"), withID("s18_upload"), withAccessibleName("Attach Photo"), withLabel("Attach Photo")),
		el(19, "/html/body/button[9]", withTag("button"), withID("s19"), withText("Sort by: Top")),
		el(20, "/html/body/div[6]", withTag("div"), withRole("button"), withID("s20"), withText("Load more comments")),

		// Direct Messages (s21-s30)
		el(21, "/html/body/input[3]", withTag("input"), withInputType("text"), withID("s21"), withPlaceholder("Search chats")),
		el(22, "/html/body/div[7]", withTag("div"), withRole("button"), withID("s22"), withAriaLabel("New Message"), withText("📝")),
		el(23, "/html/body/div[8]", withTag("div"), withRole("textbox"), withID("s23"), withEditable(), withPlaceholder("Type a message")),
		el(24, "/html/body/button[10]", withTag("button"), withID("s24"), withAriaLabel("Send Message")),
		el(25, "/html/body/button[11]", withTag("button"), withID("s25"), withAriaLabel("Record Voice Memo"), withText("🎤")),
		el(26, "/html/body/div[9]", withTag("div"), withRole("button"), withID("s26"), withAriaLabel("Video Call"), withText("📹")),
		el(27, "/html/body/div[10]", withTag("div"), withRole("button"), withID("s27"), withAriaLabel("Audio Call"), withText("📞")),
		el(28, "/html/body/button[12]", withTag("button"), withID("s28"), withText("Mute Chat")),
		el(29, "/html/body/button[13]", withTag("button"), withID("s29"), withText("Block User")),
		el(30, "/html/body/a[2]", withTag("a"), withID("s30"), withText("Message Requests")),

		// Network (s31-s40)
		el(31, "/html/body/button[14]", withTag("button"), withID("s31"), withClassName("btn-primary"), withText("Follow")),
		el(32, "/html/body/button[15]", withTag("button"), withID("s32"), withClassName("btn-secondary"), withText("Following")),
		el(33, "/html/body/button[16]", withTag("button"), withID("s33"), withText("Unfollow")),
		el(34, "/html/body/button[17]", withTag("button"), withID("s34"), withAriaLabel("Remove follower"), withText("Remove")),
		el(35, "/html/body/button[18]", withTag("button"), withID("s35"), withText("Connect")),
		el(36, "/html/body/button[19]", withTag("button"), withID("s36"), withText("Accept Request")),
		el(37, "/html/body/button[20]", withTag("button"), withID("s37"), withText("Decline")),
		el(38, "/html/body/a[3]", withTag("a"), withID("s38"), withText("Followers 1,200")),
		el(39, "/html/body/a[4]", withTag("a"), withID("s39"), withText("Following 350")),
		el(40, "/html/body/button[21]", withTag("button"), withID("s40"), withDataQA("subscribe"), withText("Subscribe")),

		// Profile Editing (s41-s50)
		el(41, "/html/body/button[22]", withTag("button"), withID("s41"), withText("Edit Profile")),
		el(42, "/html/body/input[4]", withTag("input"), withInputType("text"), withID("s42"), withAriaLabel("Display Name"), withValue("Manul QA")),
		el(43, "/html/body/textarea[1]", withTag("textarea"), withID("s43"), withAriaLabel("Bio"), withValue("Automating the web.")),
		el(44, "/html/body/input[5]", withTag("input"), withInputType("text"), withID("s44"), withPlaceholder("Add Location")),
		el(45, "/html/body/input[6]", withTag("input"), withInputType("url"), withID("s45"), withPlaceholder("Website link")),
		el(46, "/html/body/select[1]", withTag("select"), withID("s46"), withAriaLabel("Pronouns"), withText("They/Them")),
		el(47, "/html/body/button[23]", withTag("button"), withID("s47"), withAriaLabel("Change Avatar"), withText("🖼️")),
		el(48, "/html/body/button[24]", withTag("button"), withID("s48"), withAriaLabel("Change Cover Photo"), withText("🌄")),
		el(49, "/html/body/button[25]", withTag("button"), withID("s49"), withClassName("save-profile"), withText("Save Changes")),
		el(50, "/html/body/button[26]", withTag("button"), withID("s50"), withText("Cancel")),

		// Privacy & Settings (s51-s60)
		el(51, "/html/body/div[11]", withTag("div"), withRole("switch"), withID("s51"), withAriaLabel("Private Account")),
		el(52, "/html/body/div[12]", withTag("div"), withRole("switch"), withID("s52"), withAriaLabel("Show Activity Status")),
		el(53, "/html/body/input[7]", withTag("input"), withInputType("radio"), withID("s53"), withLabel("Everyone"), withNameAttr("tags")),
		el(54, "/html/body/input[8]", withTag("input"), withInputType("radio"), withID("s54"), withLabel("Friends Only"), withNameAttr("tags")),
		el(55, "/html/body/input[9]", withTag("input"), withInputType("radio"), withID("s55"), withLabel("Nobody"), withNameAttr("tags")),
		el(56, "/html/body/button[27]", withTag("button"), withID("s56"), withText("Change Password")),
		el(57, "/html/body/button[28]", withTag("button"), withID("s57"), withText("Enable 2FA")),
		el(58, "/html/body/a[5]", withTag("a"), withID("s58"), withText("Active Sessions")),
		el(59, "/html/body/button[29]", withTag("button"), withID("s59"), withText("Deactivate Account")),
		el(60, "/html/body/button[30]", withTag("button"), withID("s60"), withText("Delete Account Permanently")),

		// Notifications (s61-s70)
		el(61, "/html/body/button[31]", withTag("button"), withID("s61"), withAriaLabel("Notifications"), withText("🔔 5")),
		el(62, "/html/body/button[32]", withTag("button"), withID("s62"), withText("Mark all as read")),
		el(63, "/html/body/button[33]", withTag("button"), withID("s63"), withText("Filter by Mentions")),
		el(64, "/html/body/div[13]", withTag("div"), withRole("button"), withID("s64"), withAriaLabel("Notification Settings"), withText("⚙️")),
		el(65, "/html/body/a[6]", withTag("a"), withID("s65"), withDataQA("notif-alex"), withText("Alex liked your photo")),
		el(66, "/html/body/button[34]", withTag("button"), withID("s66"), withDataQA("notif-mute"), withText("Turn off notifications for this post")),
		el(67, "/html/body/div[14]", withTag("div"), withID("s67"), withClassName("tab"), withText("All")),
		el(68, "/html/body/div[15]", withTag("div"), withID("s68"), withClassName("tab"), withText("Verified")),
		el(69, "/html/body/div[16]", withTag("div"), withID("s69"), withClassName("tab"), withText("Mentions")),
		el(70, "/html/body/button[35]", withTag("button"), withID("s70"), withAriaLabel("Clear Notifications"), withText("Clear")),

		// Search & Explore (s71-s80)
		el(71, "/html/body/input[10]", withTag("input"), withInputType("search"), withID("s71"), withPlaceholder("Search Twitter"), withAriaLabel("Search query")),
		el(72, "/html/body/button[36]", withTag("button"), withID("s72"), withAriaLabel("Clear search"), withText("✖")),
		el(73, "/html/body/div[17]", withTag("div"), withRole("button"), withID("s73"), withText("Trending")),
		el(74, "/html/body/div[18]", withTag("div"), withRole("button"), withID("s74"), withText("News")),
		el(75, "/html/body/div[19]", withTag("div"), withRole("button"), withID("s75"), withText("Sports")),
		el(76, "/html/body/a[7]", withTag("a"), withID("s76"), withText("#Manul")),
		el(77, "/html/body/button[37]", withTag("button"), withID("s77"), withAriaLabel("Search Settings"), withText("⚙️")),
		el(78, "/html/body/div[20]", withTag("div"), withRole("button"), withID("s78"), withText("Clear recent searches")),
		el(79, "/html/body/button[38]", withTag("button"), withID("s79"), withText("Show more trends")),
		el(80, "/html/body/input[11]", withTag("input"), withInputType("text"), withID("s80"), withPlaceholder("Search messages...")),

		// Groups & Events (s81-s90)
		el(81, "/html/body/button[39]", withTag("button"), withID("s81"), withText("Join Group")),
		el(82, "/html/body/button[40]", withTag("button"), withID("s82"), withText("Leave Group")),
		el(83, "/html/body/button[41]", withTag("button"), withID("s83"), withText("Invite Friends")),
		el(84, "/html/body/button[42]", withTag("button"), withID("s84"), withText("Going")),
		el(85, "/html/body/button[43]", withTag("button"), withID("s85"), withText("Maybe")),
		el(86, "/html/body/button[44]", withTag("button"), withID("s86"), withText("Can't Go")),
		el(87, "/html/body/a[8]", withTag("a"), withID("s87"), withText("Create Event")),
		el(88, "/html/body/input[12]", withTag("input"), withInputType("text"), withID("s88"), withPlaceholder("Event Name")),
		el(89, "/html/body/textarea[2]", withTag("textarea"), withID("s89"), withPlaceholder("Event Description")),
		el(90, "/html/body/button[45]", withTag("button"), withID("s90"), withText("Publish Event")),

		// Stories & Modals (s91-s100)
		el(91, "/html/body/div[21]", withTag("div"), withRole("button"), withID("s91"), withAriaLabel("Next Story"), withText("▶")),
		el(92, "/html/body/div[22]", withTag("div"), withRole("button"), withID("s92"), withAriaLabel("Previous Story"), withText("◀")),
		el(93, "/html/body/div[23]", withTag("div"), withRole("button"), withID("s93"), withAriaLabel("Pause Story"), withText("⏸")),
		el(94, "/html/body/input[13]", withTag("input"), withInputType("text"), withID("s94"), withPlaceholder("Reply to story...")),
		el(95, "/html/body/button[46]", withTag("button"), withID("s95"), withAriaLabel("Send Reaction 💖"), withText("💖")),
		el(96, "/html/body/button[47]", withTag("button"), withID("s96"), withAriaLabel("Send Reaction 🔥"), withText("🔥")),
		el(97, "/html/body/button[48]", withTag("button"), withID("s97"), withAriaLabel("Close Modal"), withText("Close")),
		el(98, "/html/body/button[49]", withTag("button"), withID("s98"), withText("Copy Link to Tweet")),
		el(99, "/html/body/button[50]", withTag("button"), withID("s99"), withText("Pin to profile")),
		el(100, "/html/body/button[51]", withTag("button"), withID("s100"), withText("Log Out")),
	}
}

func TestSocialMedia(t *testing.T) {
	elements := socialDOM()

	tests := []struct {
		name, query, mode, expectedID string
	}{
		// Feed
		{"01_LikePost", "Like Post", "clickable", "s1"},
		{"02_Unlike", "Unlike", "clickable", "s2"},
		{"03_Repost", "Repost", "clickable", "s3"},
		{"04_ShareViaDM", "Share via Direct Message", "clickable", "s4"},
		{"05_SaveBookmarks", "Save to Bookmarks", "clickable", "s5"},
		{"06_ReportPost", "Report this post", "clickable", "s6"},
		{"07_Hide", "Hide", "clickable", "s7"},
		{"08_MoreOptions", "More Options", "clickable", "s8"},
		{"09_TranslatePost", "Translate post", "clickable", "s9"},
		{"10_ViewHiddenReplies", "View 15 hidden replies", "clickable", "s10"},

		// Commenting
		{"11_WriteComment", "Write a comment...", "input", "s11"},
		{"12_Reply", "Reply", "clickable", "s13"},
		{"14_MentionSomeone", "Mention someone", "input", "s14"},
		{"15_AlexDev", "@alex_dev", "clickable", "s15"},
		{"16_ManulQA", "@manul_qa", "clickable", "s16"},
		{"17_InsertEmoji", "Insert Emoji", "clickable", "s17"},
		{"18_AttachPhoto", "Attach Photo", "clickable", "s18_upload"},
		{"19_SortByTop", "Sort by: Top", "clickable", "s19"},
		{"20_LoadMoreComments", "Load more comments", "clickable", "s20"},

		// Direct Messages
		{"21_SearchChats", "Search chats", "input", "s21"},
		{"22_NewMessage", "New Message", "clickable", "s22"},
		{"23_TypeMessage", "Type a message", "input", "s23"},
		{"24_SendMessage", "Send Message", "clickable", "s24"},
		{"25_RecordVoiceMemo", "Record Voice Memo", "clickable", "s25"},
		{"26_VideoCall", "Video Call", "clickable", "s26"},
		{"27_AudioCall", "Audio Call", "clickable", "s27"},
		{"28_MuteChat", "Mute Chat", "clickable", "s28"},
		{"29_BlockUser", "Block User", "clickable", "s29"},
		{"30_MessageRequests", "Message Requests", "clickable", "s30"},

		// Network
		{"31_Follow", "Follow", "clickable", "s31"},
		{"32_Following", "Following", "clickable", "s32"},
		{"33_Unfollow", "Unfollow", "clickable", "s33"},
		{"34_RemoveFollower", "Remove follower", "clickable", "s34"},
		{"35_Connect", "Connect", "clickable", "s35"},
		{"36_AcceptRequest", "Accept Request", "clickable", "s36"},
		{"37_Decline", "Decline", "clickable", "s37"},
		// 38,39 = EXTRACT (skip)
		{"40_Subscribe", "Subscribe", "clickable", "s40"},

		// Profile
		{"41_EditProfile", "Edit Profile", "clickable", "s41"},
		{"42_DisplayName", "Display Name", "input", "s42"},
		{"43_Bio", "Bio", "input", "s43"},
		{"44_AddLocation", "Add Location", "input", "s44"},
		{"45_WebsiteLink", "Website link", "input", "s45"},
		{"47_ChangeAvatar", "Change Avatar", "clickable", "s47"},
		{"48_ChangeCoverPhoto", "Change Cover Photo", "clickable", "s48"},
		{"49_SaveChanges", "Save Changes", "clickable", "s49"},
		{"50_Cancel", "Cancel", "clickable", "s50"},

		// Privacy
		{"51_PrivateAccount", "Private Account", "clickable", "s51"},
		{"52_ShowActivityStatus", "Show Activity Status", "clickable", "s52"},
		{"53_Everyone", "Everyone", "clickable", "s53"},
		{"54_FriendsOnly", "Friends Only", "clickable", "s54"},
		{"55_Nobody", "Nobody", "clickable", "s55"},
		{"56_ChangePassword", "Change Password", "clickable", "s56"},
		{"57_Enable2FA", "Enable 2FA", "clickable", "s57"},
		{"58_ActiveSessions", "Active Sessions", "clickable", "s58"},
		{"59_DeactivateAccount", "Deactivate Account", "clickable", "s59"},
		{"60_DeleteAccountPermanently", "Delete Account Permanently", "clickable", "s60"},

		// Notifications
		{"61_Notifications", "Notifications", "clickable", "s61"},
		// 62 = EXTRACT (skip)
		{"63_MarkAllAsRead", "Mark all as read", "clickable", "s62"},
		{"64_FilterByMentions", "Filter by Mentions", "clickable", "s63"},
		{"65_NotificationSettings", "Notification Settings", "clickable", "s64"},
		{"66_AlexLikedPhoto", "Alex liked your photo", "clickable", "s65"},
		{"67_TurnOffNotifications", "Turn off notifications for this post", "clickable", "s66"},
		{"68_Verified", "Verified", "clickable", "s68"},
		{"69_Mentions", "Mentions", "clickable", "s69"},
		{"70_ClearNotifications", "Clear Notifications", "clickable", "s70"},

		// Search
		{"71_SearchQuery", "Search query", "input", "s71"},
		{"72_ClearSearch", "Clear search", "clickable", "s72"},
		{"73_Trending", "Trending", "clickable", "s73"},
		{"74_News", "News", "clickable", "s74"},
		{"75_Sports", "Sports", "clickable", "s75"},
		{"76_Manul", "#Manul", "clickable", "s76"},
		{"77_SearchSettings", "Search Settings", "clickable", "s77"},
		{"78_ClearRecentSearches", "Clear recent searches", "clickable", "s78"},
		{"79_ShowMoreTrends", "Show more trends", "clickable", "s79"},
		{"80_SearchMessages", "Search messages...", "input", "s80"},

		// Groups & Events
		{"81_JoinGroup", "Join Group", "clickable", "s81"},
		{"82_LeaveGroup", "Leave Group", "clickable", "s82"},
		{"83_InviteFriends", "Invite Friends", "clickable", "s83"},
		{"84_Going", "Going", "clickable", "s84"},
		{"85_Maybe", "Maybe", "clickable", "s85"},
		{"86_CantGo", "Can't Go", "clickable", "s86"},
		{"87_CreateEvent", "Create Event", "clickable", "s87"},
		{"88_EventName", "Event Name", "input", "s88"},
		{"89_EventDescription", "Event Description", "input", "s89"},
		{"90_PublishEvent", "Publish Event", "clickable", "s90"},

		// Stories & Modals
		{"91_NextStory", "Next Story", "clickable", "s91"},
		{"92_PreviousStory", "Previous Story", "clickable", "s92"},
		{"93_PauseStory", "Pause Story", "clickable", "s93"},
		{"94_ReplyToStory", "Reply to story", "input", "s94"},
		{"95_SendReactionHeart", "Send Reaction 💖", "clickable", "s95"},
		{"96_SendReactionFire", "Send Reaction 🔥", "clickable", "s96"},
		{"97_CloseModal", "Close Modal", "clickable", "s97"},
		{"98_CopyLinkToTweet", "Copy Link to Tweet", "clickable", "s98"},
		{"99_PinToProfile", "Pin to profile", "clickable", "s99"},
		{"100_LogOut", "Log Out", "clickable", "s100"},
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

func TestSocialMedia_Select(t *testing.T) {
	elements := socialDOM()

	tests := []struct {
		name, query, expectedID string
	}{
		{"46_Pronouns", "Pronouns", "s46"},
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

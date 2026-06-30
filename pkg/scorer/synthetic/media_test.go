package synthetic

// ─────────────────────────────────────────────────────────────────────────────
// MEDIA & VIDEO STREAMING DOM SCORING TEST SUITE
//
// Port of ManulEngine test_06_media.py — 100-element media/streaming page.
// Validates: video player controls, metadata/engagement, comments, autoplay,
// Netflix-style hero/carousel/episodes, Spotify music player, search/genre,
// ads, age gate.
// Skipped: extract (8,21,22), verify (31,36,46,51,55,62,65,91,94,97),
//          optional/hidden (2,100), absent tests (50,60,80).
// ─────────────────────────────────────────────────────────────────────────────

import (
	"testing"

	"github.com/alexbeatnik/ManulEngineGo/pkg/dom"
)

func mediaDOM() []dom.ElementSnapshot {
	return []dom.ElementSnapshot{
		// Video Player Controls (m1-m20)
		el(1, "/html/body/button[1]", withTag("button"), withID("m1"), withAriaLabel("Play Video"), withText("▶")),
		el(2, "/html/body/button[2]", withTag("button"), withID("m2"), withAriaLabel("Pause Video"), withHidden(), withText("⏸")),
		el(3, "/html/body/button[3]", withTag("button"), withID("m3"), withAriaLabel("Rewind 10 seconds"), withText("⏪")),
		el(4, "/html/body/button[4]", withTag("button"), withID("m4"), withAriaLabel("Forward 10 seconds"), withText("⏩")),
		el(5, "/html/body/button[5]", withTag("button"), withID("m5"), withAriaLabel("Mute")),
		el(6, "/html/body/input[1]", withTag("input"), withInputType("range"), withID("m6"), withAriaLabel("Volume"), withValue("50")),
		el(7, "/html/body/input[2]", withTag("input"), withInputType("range"), withID("m7"), withAriaLabel("Timeline"), withValue("120")),
		el(8, "/html/body/span[1]", withTag("span"), withID("m8"), withDataQA("timestamp"), withText("12:34 / 45:00")),
		el(9, "/html/body/button[6]", withTag("button"), withID("m9"), withAriaLabel("Next Video"), withText("⏭")),
		el(10, "/html/body/div[1]", withTag("div"), withRole("button"), withID("m10"), withAriaLabel("Miniplayer"), withText("🔲")),
		el(11, "/html/body/button[7]", withTag("button"), withID("m11"), withAriaLabel("Settings"), withText("⚙️")),
		el(12, "/html/body/button[8]", withTag("button"), withID("m12"), withAriaLabel("Subtitles/closed captions (c)"), withText("CC")),
		el(13, "/html/body/button[9]", withTag("button"), withID("m13"), withAriaLabel("Theater mode (t)"), withText("🖵")),
		el(14, "/html/body/button[10]", withTag("button"), withID("m14"), withAriaLabel("Fullscreen (f)"), withText("🔲")),
		el(15, "/html/body/select[1]", withTag("select"), withID("m15"), withAriaLabel("Playback Quality"), withText("Auto")),
		el(16, "/html/body/select[2]", withTag("select"), withID("m16"), withAriaLabel("Playback Speed"), withText("Normal")),
		el(17, "/html/body/div[2]", withTag("div"), withRole("menuitem"), withID("m17"), withText("Report playback issue")),
		el(18, "/html/body/div[3]", withTag("div"), withRole("switch"), withID("m18"), withAriaLabel("Annotations"), withText("Annotations")),
		el(19, "/html/body/div[4]", withTag("div"), withRole("switch"), withID("m19"), withAriaLabel("Ambient Mode"), withText("Ambient Mode")),
		el(20, "/html/body/button[11]", withTag("button"), withID("m20"), withAriaLabel("Loop video"), withText("🔁")),

		// Video Metadata & Engagement (m21-m30)
		el(21, "/html/body/h1[1]", withTag("h1"), withID("m21"), withDataQA("video-title"), withText("10 Hours of Manul Cats Meowing")),
		el(22, "/html/body/span[2]", withTag("span"), withID("m22"), withText("1,500,000 views")),
		el(23, "/html/body/button[12]", withTag("button"), withID("m23"), withAriaLabel("Like this video"), withText("👍 50K")),
		el(24, "/html/body/button[13]", withTag("button"), withID("m24"), withAriaLabel("Dislike this video"), withText("👎")),
		el(25, "/html/body/button[14]", withTag("button"), withID("m25"), withText("Share")),
		el(26, "/html/body/button[15]", withTag("button"), withID("m26"), withText("Download")),
		el(27, "/html/body/button[16]", withTag("button"), withID("m27"), withAriaLabel("Save to playlist"), withText("➕ Save")),
		el(28, "/html/body/button[17]", withTag("button"), withID("m28"), withClassName("btn-subscribe"), withText("Subscribe")),
		el(29, "/html/body/button[18]", withTag("button"), withID("m29"), withAriaLabel("Join channel"), withText("Join")),
		el(30, "/html/body/button[19]", withTag("button"), withID("m30"), withAriaLabel("More actions"), withText("•••")),

		// Comments (m31-m40)
		el(31, "/html/body/h2[1]", withTag("h2"), withID("m31"), withText("1,204 Comments")),
		el(32, "/html/body/button[20]", withTag("button"), withID("m32"), withText("Sort by")),
		el(33, "/html/body/div[5]", withTag("div"), withRole("menuitem"), withID("m33"), withText("Top comments")),
		el(34, "/html/body/div[6]", withTag("div"), withRole("menuitem"), withID("m34"), withText("Newest first")),
		el(35, "/html/body/input[3]", withTag("input"), withInputType("text"), withID("m35"), withPlaceholder("Add a comment...")),
		el(36, "/html/body/button[21]", withTag("button"), withID("m36"), withDisabled(), withText("Comment")),
		el(37, "/html/body/button[22]", withTag("button"), withID("m37"), withText("Cancel")),
		el(38, "/html/body/button[23]", withTag("button"), withID("m38"), withAriaLabel("Like comment"), withText("👍")),
		el(39, "/html/body/button[24]", withTag("button"), withID("m39"), withText("Reply")),
		el(40, "/html/body/div[7]", withTag("div"), withRole("button"), withID("m40"), withText("View 5 replies")),

		// Autoplay & Recommendations (m41-m50)
		el(41, "/html/body/div[8]", withTag("div"), withRole("switch"), withID("m41"), withText("Autoplay")),
		el(42, "/html/body/a[1]", withTag("a"), withID("m42"), withText("Funny Dogs Compilation")),
		el(43, "/html/body/button[25]", withTag("button"), withID("m43"), withAriaLabel("Add to queue"), withText("🕒")),
		el(44, "/html/body/a[2]", withTag("a"), withID("m44"), withText("Nature Documentary")),
		el(45, "/html/body/button[26]", withTag("button"), withID("m45"), withAriaLabel("Add to queue"), withText("🕒")),
		el(46, "/html/body/button[27]", withTag("button"), withID("m46"), withText("Show more")),
		el(47, "/html/body/div[9]", withTag("div"), withID("m47"), withText("Playlist: Favorites (12/50)")),
		el(48, "/html/body/button[28]", withTag("button"), withID("m48"), withAriaLabel("Shuffle playlist"), withText("🔀")),
		el(49, "/html/body/button[29]", withTag("button"), withID("m49"), withAriaLabel("Save playlist"), withText("➕")),
		el(50, "/html/body/button[30]", withTag("button"), withID("m50"), withAriaLabel("Hide playlist"), withText("❌")),

		// Netflix-style Hero & Carousel (m51-m60)
		el(51, "/html/body/h1[2]", withTag("h1"), withID("m51"), withText("Stranger Manuls")),
		el(52, "/html/body/button[31]", withTag("button"), withID("m52"), withClassName("btn-play"), withDataQA("hero-play"), withText("▶ Play")),
		el(53, "/html/body/button[32]", withTag("button"), withID("m53"), withClassName("btn-info"), withText("ℹ More Info")),
		el(54, "/html/body/button[33]", withTag("button"), withID("m54"), withAriaLabel("Mute trailer"), withText("🔇")),
		el(55, "/html/body/h3[1]", withTag("h3"), withID("m55"), withText("Trending Now")),
		el(56, "/html/body/button[34]", withTag("button"), withID("m56"), withAriaLabel("Scroll Left"), withText("◀")),
		el(57, "/html/body/div[10]", withTag("div"), withID("m57"), withClassName("movie-card"), withText("Movie 1")),
		el(58, "/html/body/div[11]", withTag("div"), withID("m58"), withClassName("movie-card"), withText("Movie 2")),
		el(59, "/html/body/button[35]", withTag("button"), withID("m59"), withAriaLabel("Scroll Right"), withText("▶")),
		el(60, "/html/body/button[36]", withTag("button"), withID("m60"), withAriaLabel("Add to My List"), withText("✚")),

		// Episodes (m61-m70)
		el(61, "/html/body/select[3]", withTag("select"), withID("m61"), withAriaLabel("Season Selector"), withText("Season 1")),
		el(62, "/html/body/div[12]", withTag("div"), withID("m62"), withClassName("episode"), withText("1. The Beginning")),
		el(63, "/html/body/button[37]", withTag("button"), withID("m63"), withText("Play Episode 1")),
		el(64, "/html/body/button[38]", withTag("button"), withID("m64"), withText("Download Episode 1")),
		el(65, "/html/body/div[13]", withTag("div"), withID("m65"), withClassName("episode"), withText("2. The Middle")),
		el(66, "/html/body/button[39]", withTag("button"), withID("m66"), withText("Play Episode 2")),
		el(67, "/html/body/button[40]", withTag("button"), withID("m67"), withText("Resume from 15:00")),
		el(68, "/html/body/button[41]", withTag("button"), withID("m68"), withText("Play from beginning")),
		el(69, "/html/body/button[42]", withTag("button"), withID("m69"), withText("Rate this title")),
		el(70, "/html/body/button[43]", withTag("button"), withID("m70"), withAriaLabel("Remove from My List"), withText("✔")),

		// Spotify-style Music Player (m71-m80)
		el(71, "/html/body/img[1]", withTag("img"), withID("m71"), withText("Album Cover")),
		el(72, "/html/body/button[44]", withTag("button"), withID("m72"), withAriaLabel("Save to Your Library"), withText("💚")),
		el(73, "/html/body/button[45]", withTag("button"), withID("m73"), withAriaLabel("Enable shuffle"), withText("🔀")),
		el(74, "/html/body/button[46]", withTag("button"), withID("m74"), withAriaLabel("Previous track"), withText("⏮")),
		el(75, "/html/body/button[47]", withTag("button"), withID("m75"), withClassName("play-btn"), withAriaLabel("Pause/Play"), withText("▶")),
		el(76, "/html/body/button[48]", withTag("button"), withID("m76"), withAriaLabel("Next track"), withText("⏭")),
		el(77, "/html/body/button[49]", withTag("button"), withID("m77"), withAriaLabel("Enable repeat"), withText("🔁")),
		el(78, "/html/body/button[50]", withTag("button"), withID("m78"), withAriaLabel("Lyrics"), withText("🎤")),
		el(79, "/html/body/button[51]", withTag("button"), withID("m79"), withAriaLabel("Queue"), withText("🎶")),
		el(80, "/html/body/button[52]", withTag("button"), withID("m80"), withAriaLabel("Connect to a device"), withText("💻")),

		// Search & Filters (m81-m90)
		el(81, "/html/body/input[4]", withTag("input"), withInputType("search"), withID("m81"), withPlaceholder("Movies, shows, and more")),
		el(82, "/html/body/button[53]", withTag("button"), withID("m82"), withAriaLabel("Clear Search"), withText("X")),
		el(83, "/html/body/button[54]", withTag("button"), withID("m83"), withText("Filter by Genre")),
		el(84, "/html/body/div[14]", withTag("div"), withRole("checkbox"), withID("m84"), withText("Action")),
		el(85, "/html/body/div[15]", withTag("div"), withRole("checkbox"), withID("m85"), withText("Comedy")),
		el(86, "/html/body/div[16]", withTag("div"), withRole("checkbox"), withID("m86"), withText("Sci-Fi")),
		el(87, "/html/body/button[55]", withTag("button"), withID("m87"), withText("Apply Filters")),
		el(88, "/html/body/a[3]", withTag("a"), withID("m88"), withText("Podcasts")),
		el(89, "/html/body/a[4]", withTag("a"), withID("m89"), withText("Audiobooks")),
		el(90, "/html/body/a[5]", withTag("a"), withID("m90"), withText("Live Radio")),

		// Ads & Prompts (m91-m100)
		el(91, "/html/body/div[17]", withTag("div"), withID("m91"), withText("Ad ends in 5")),
		el(92, "/html/body/button[56]", withTag("button"), withID("m92"), withText("Skip Ad")),
		el(93, "/html/body/button[57]", withTag("button"), withID("m93"), withAriaLabel("Learn more about this ad"), withText("Learn More")),
		el(94, "/html/body/h2[2]", withTag("h2"), withID("m94"), withText("Are you still watching?")),
		el(95, "/html/body/button[58]", withTag("button"), withID("m95"), withText("Continue Watching")),
		el(96, "/html/body/button[59]", withTag("button"), withID("m96"), withText("Go to Homepage")),
		el(97, "/html/body/h2[3]", withTag("h2"), withID("m97"), withText("This video may be inappropriate for some users.")),
		el(98, "/html/body/button[60]", withTag("button"), withID("m98"), withText("I understand and wish to proceed")),
		el(99, "/html/body/button[61]", withTag("button"), withID("m99"), withText("Go back")),
		el(100, "/html/body/button[62]", withTag("button"), withID("m100"), withHidden(), withText("Claim Free Premium")),
	}
}

func TestMedia(t *testing.T) {
	elements := mediaDOM()

	tests := []struct {
		name    string
		query   string
		mode    string
		wantID  string
	}{
		// Video Player Controls
		{"01 Play Video", "Play Video", "clickable", "m1"},
		// test 2 skipped — optional/hidden (Pause Video)
		{"03 Rewind 10 seconds", "Rewind 10 seconds", "clickable", "m3"},
		{"04 Forward 10 seconds", "Forward 10 seconds", "clickable", "m4"},
		{"05 Mute", "Mute", "clickable", "m5"},
		{"06 Volume", "Volume", "input", "m6"},
		{"07 Timeline", "Timeline", "input", "m7"},
		// test 8 skipped — extract
		{"09 Next Video", "Next Video", "clickable", "m9"},
		{"10 Miniplayer", "Miniplayer", "clickable", "m10"},
		{"11 Settings", "Settings", "clickable", "m11"},
		{"12 Subtitles/closed captions", "Subtitles/closed captions", "clickable", "m12"},
		{"13 Theater mode", "Theater mode", "clickable", "m13"},
		{"14 Fullscreen", "Fullscreen", "clickable", "m14"},
		// tests 15,16 → select tests below
		{"17 Report playback issue", "Report playback issue", "clickable", "m17"},
		{"18 Annotations", "Annotations", "clickable", "m18"},
		{"19 Ambient Mode", "Ambient Mode", "clickable", "m19"},
		{"20 Loop video", "Loop video", "clickable", "m20"},

		// Video Metadata & Engagement
		// tests 21,22 skipped — extract
		{"23 Like this video", "Like this video", "clickable", "m23"},
		{"24 Dislike this video", "Dislike this video", "clickable", "m24"},
		{"25 Share", "Share", "clickable", "m25"},
		{"26 Download", "Download", "clickable", "m26"},
		{"27 Save to playlist", "Save to playlist", "clickable", "m27"},
		{"28 Subscribe", "Subscribe", "clickable", "m28"},
		{"29 Join channel", "Join channel", "clickable", "m29"},
		{"30 More actions", "More actions", "clickable", "m30"},

		// Comments
		// test 31 skipped — verify
		{"32 Sort by", "Sort by", "clickable", "m32"},
		{"33 Top comments", "Top comments", "clickable", "m33"},
		{"34 Newest first", "Newest first", "clickable", "m34"},
		{"35 Add a comment", "Add a comment...", "input", "m35"},
		// test 36 skipped — verify (Comment is disabled)
		{"37 Cancel", "Cancel", "clickable", "m37"},
		{"38 Like comment", "Like comment", "clickable", "m38"},
		{"39 Reply", "Reply", "clickable", "m39"},
		{"40 View 5 replies", "View 5 replies", "clickable", "m40"},

		// Autoplay & Recommendations
		{"41 Autoplay", "Autoplay", "clickable", "m41"},
		{"42 Funny Dogs Compilation", "Funny Dogs Compilation", "clickable", "m42"},
		{"43 Add to queue", "Add to queue", "clickable", "m43"},
		{"44 Nature Documentary", "Nature Documentary", "clickable", "m44"},
		// test 45 skipped — second "Add to queue" → same as 43
		{"45 Show more", "Show more", "clickable", "m46"},
		// test 46 skipped — verify
		{"47 Shuffle playlist", "Shuffle playlist", "clickable", "m48"},
		{"48 Save playlist", "Save playlist", "clickable", "m49"},
		{"49 Hide playlist", "Hide playlist", "clickable", "m50"},
		// test 50 absent from Python tests

		// Netflix-style Hero & Carousel
		// test 51 skipped — verify
		{"52 Play hero", "Play", "clickable", "m75"},
		{"53 More Info", "More Info", "clickable", "m53"},
		{"54 Mute trailer", "Mute trailer", "clickable", "m54"},
		// test 55 skipped — verify
		{"56 Scroll Left", "Scroll Left", "clickable", "m56"},
		{"57 Movie 1", "Movie 1", "clickable", "m57"},
		{"58 Scroll Right", "Scroll Right", "clickable", "m59"},
		{"59 Add to My List", "Add to My List", "clickable", "m60"},
		// test 60 absent from Python tests

		// Episodes
		// test 61 → select test below
		// test 62 skipped — verify
		{"63 Play Episode 1", "Play Episode 1", "clickable", "m63"},
		{"64 Download Episode 1", "Download Episode 1", "clickable", "m64"},
		// test 65 skipped — verify
		{"66 Play Episode 2", "Play Episode 2", "clickable", "m66"},
		{"67 Resume from 15:00", "Resume from 15:00", "clickable", "m67"},
		{"68 Play from beginning", "Play from beginning", "clickable", "m68"},
		{"69 Rate this title", "Rate this title", "clickable", "m69"},
		{"70 Remove from My List", "Remove from My List", "clickable", "m70"},

		// Spotify-style Music Player
		{"71 Save to Your Library", "Save to Your Library", "clickable", "m72"},
		{"72 Enable shuffle", "Enable shuffle", "clickable", "m73"},
		{"73 Previous track", "Previous track", "clickable", "m74"},
		{"74 Pause/Play", "Pause/Play", "clickable", "m75"},
		{"75 Next track", "Next track", "clickable", "m76"},
		{"76 Enable repeat", "Enable repeat", "clickable", "m77"},
		{"77 Lyrics", "Lyrics", "clickable", "m78"},
		{"78 Queue", "Queue", "clickable", "m79"},
		{"79 Connect to a device", "Connect to a device", "clickable", "m80"},
		// test 80 absent from Python tests

		// Search & Filters
		{"81 Search movies", "Movies, shows, and more", "input", "m81"},
		{"82 Clear Search", "Clear Search", "clickable", "m82"},
		{"83 Filter by Genre", "Filter by Genre", "clickable", "m83"},
		{"84 Action checkbox", "Action", "clickable", "m84"},
		{"85 Comedy checkbox", "Comedy", "clickable", "m85"},
		{"86 Sci-Fi checkbox", "Sci-Fi", "clickable", "m86"},
		{"87 Apply Filters", "Apply Filters", "clickable", "m87"},
		{"88 Podcasts", "Podcasts", "clickable", "m88"},
		{"89 Audiobooks", "Audiobooks", "clickable", "m89"},
		{"90 Live Radio", "Live Radio", "clickable", "m90"},

		// Ads & Prompts
		// test 91 skipped — verify
		{"92 Skip Ad", "Skip Ad", "clickable", "m92"},
		{"93 Learn More", "Learn More", "clickable", "m93"},
		// test 94 skipped — verify
		{"95 Continue Watching", "Continue Watching", "clickable", "m95"},
		{"96 Go to Homepage", "Go to Homepage", "clickable", "m96"},
		// test 97 skipped — verify
		{"98 I understand", "I understand and wish to proceed", "clickable", "m98"},
		{"99 Go back", "Go back", "clickable", "m99"},
		// test 100 skipped — optional/hidden
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rankFirstID(t, tt.query, "", tt.mode, elements)
			if got != tt.wantID {
				t.Errorf("expected %s, got %s", tt.wantID, got)
			}
		})
	}
}

func TestMedia_Select(t *testing.T) {
	elements := mediaDOM()

	tests := []struct {
		name   string
		query  string
		wantID string
	}{
		{"15 Playback Quality 4K", "Playback Quality", "m15"},
		{"16 Playback Speed 1.5x", "Playback Speed", "m16"},
		{"61 Season Selector", "Season Selector", "m61"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rankFirstID(t, tt.query, "", "select", elements)
			if got != tt.wantID {
				t.Errorf("expected %s, got %s", tt.wantID, got)
			}
		})
	}
}

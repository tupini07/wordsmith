package emoji

// Entry represents a single emoji with its shortcode name.
type Entry struct {
	Name  string
	Emoji string
}

// Search returns entries whose name contains the query (case-insensitive).
// If query is empty, returns all entries (capped at limit).
func Search(query string, limit int) []Entry {
	if limit <= 0 {
		limit = 20
	}
	var results []Entry
	q := toLower(query)
	for _, e := range catalog {
		if q == "" || contains(toLower(e.Name), q) {
			results = append(results, e)
			if len(results) >= limit {
				break
			}
		}
	}
	return results
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// catalog is a curated list of commonly used emojis, ordered by general usefulness.
var catalog = []Entry{
	// Smileys & Emotion
	{"smile", "😊"},
	{"grinning", "😀"},
	{"laugh", "😂"},
	{"joy", "🤣"},
	{"wink", "😉"},
	{"sweat_smile", "😅"},
	{"blush", "😊"},
	{"hugging", "🤗"},
	{"heart_eyes", "😍"},
	{"star_struck", "🤩"},
	{"kissing_heart", "😘"},
	{"thinking", "🤔"},
	{"shushing", "🤫"},
	{"zipper_mouth", "🤐"},
	{"raised_eyebrow", "🤨"},
	{"neutral", "😐"},
	{"expressionless", "😑"},
	{"unamused", "😒"},
	{"rolling_eyes", "🙄"},
	{"grimacing", "😬"},
	{"relieved", "😌"},
	{"pensive", "😔"},
	{"sleepy", "😪"},
	{"sleeping", "😴"},
	{"drooling", "🤤"},
	{"sick", "🤢"},
	{"vomiting", "🤮"},
	{"sneeze", "🤧"},
	{"hot", "🥵"},
	{"cold", "🥶"},
	{"dizzy", "😵"},
	{"exploding_head", "🤯"},
	{"cowboy", "🤠"},
	{"party", "🥳"},
	{"sunglasses", "😎"},
	{"nerd", "🤓"},
	{"monocle", "🧐"},
	{"confused", "😕"},
	{"worried", "😟"},
	{"frown", "☹️"},
	{"angry", "😠"},
	{"rage", "🤬"},
	{"cry", "😢"},
	{"sob", "😭"},
	{"scream", "😱"},
	{"fearful", "😨"},
	{"sweat", "😰"},
	{"hugging", "🤗"},
	{"clown", "🤡"},
	{"ghost", "👻"},
	{"skull", "💀"},
	{"alien", "👽"},
	{"robot", "🤖"},
	{"poop", "💩"},
	{"devil", "😈"},

	// Gestures & People
	{"wave", "👋"},
	{"ok_hand", "👌"},
	{"pinched", "🤌"},
	{"peace", "✌️"},
	{"crossed_fingers", "🤞"},
	{"love_you", "🤟"},
	{"metal", "🤘"},
	{"call_me", "🤙"},
	{"point_left", "👈"},
	{"point_right", "👉"},
	{"point_up", "👆"},
	{"point_down", "👇"},
	{"thumbsup", "👍"},
	{"thumbsdown", "👎"},
	{"fist", "✊"},
	{"punch", "👊"},
	{"clap", "👏"},
	{"raised_hands", "🙌"},
	{"open_hands", "👐"},
	{"palms_up", "🤲"},
	{"handshake", "🤝"},
	{"pray", "🙏"},
	{"writing_hand", "✍️"},
	{"muscle", "💪"},
	{"brain", "🧠"},
	{"eyes", "👀"},
	{"eye", "👁️"},
	{"tongue", "👅"},
	{"ear", "👂"},
	{"nose", "👃"},
	{"baby", "👶"},
	{"person", "🧑"},
	{"shrug", "🤷"},
	{"facepalm", "🤦"},

	// Hearts & Symbols
	{"heart", "❤️"},
	{"orange_heart", "🧡"},
	{"yellow_heart", "💛"},
	{"green_heart", "💚"},
	{"blue_heart", "💙"},
	{"purple_heart", "💜"},
	{"black_heart", "🖤"},
	{"white_heart", "🤍"},
	{"broken_heart", "💔"},
	{"sparkling_heart", "💖"},
	{"fire", "🔥"},
	{"sparkles", "✨"},
	{"star", "⭐"},
	{"glowing_star", "🌟"},
	{"lightning", "⚡"},
	{"boom", "💥"},
	{"collision", "💥"},
	{"100", "💯"},
	{"check", "✅"},
	{"cross_mark", "❌"},
	{"warning", "⚠️"},
	{"question", "❓"},
	{"exclamation", "❗"},
	{"plus", "➕"},
	{"minus", "➖"},

	// Nature & Animals
	{"sun", "☀️"},
	{"moon", "🌙"},
	{"cloud", "☁️"},
	{"rain", "🌧️"},
	{"snow", "❄️"},
	{"rainbow", "🌈"},
	{"umbrella", "☂️"},
	{"wave_water", "🌊"},
	{"flower", "🌸"},
	{"rose", "🌹"},
	{"sunflower", "🌻"},
	{"tree", "🌳"},
	{"herb", "🌿"},
	{"potted_plant", "🪴"},
	{"seedling", "🌱"},
	{"leaf", "🍃"},
	{"fallen_leaf", "🍂"},
	{"cactus", "🌵"},
	{"mushroom", "🍄"},
	{"dog", "🐶"},
	{"cat", "🐱"},
	{"mouse", "🐭"},
	{"rabbit", "🐰"},
	{"fox", "🦊"},
	{"bear", "🐻"},
	{"panda", "🐼"},
	{"penguin", "🐧"},
	{"bird", "🐦"},
	{"butterfly", "🦋"},
	{"bee", "🐝"},
	{"bug", "🐛"},
	{"snake", "🐍"},
	{"turtle", "🐢"},
	{"fish", "🐟"},
	{"octopus", "🐙"},
	{"whale", "🐳"},
	{"unicorn", "🦄"},
	{"dragon", "🐉"},

	// Food & Drink
	{"apple", "🍎"},
	{"banana", "🍌"},
	{"grapes", "🍇"},
	{"watermelon", "🍉"},
	{"strawberry", "🍓"},
	{"peach", "🍑"},
	{"avocado", "🥑"},
	{"pizza", "🍕"},
	{"hamburger", "🍔"},
	{"taco", "🌮"},
	{"burrito", "🌯"},
	{"sushi", "🍣"},
	{"egg", "🥚"},
	{"coffee", "☕"},
	{"tea", "🍵"},
	{"beer", "🍺"},
	{"wine", "🍷"},
	{"cocktail", "🍸"},
	{"cake", "🎂"},
	{"cookie", "🍪"},
	{"chocolate", "🍫"},
	{"ice_cream", "🍦"},
	{"donut", "🍩"},
	{"popcorn", "🍿"},

	// Activities & Objects
	{"soccer", "⚽"},
	{"basketball", "🏀"},
	{"football", "🏈"},
	{"baseball", "⚾"},
	{"tennis", "🎾"},
	{"guitar", "🎸"},
	{"microphone", "🎤"},
	{"headphones", "🎧"},
	{"art", "🎨"},
	{"movie", "🎬"},
	{"camera", "📷"},
	{"book", "📖"},
	{"notebook", "📓"},
	{"pencil", "✏️"},
	{"pen", "🖊️"},
	{"memo", "📝"},
	{"folder", "📁"},
	{"calendar", "📅"},
	{"clock", "🕐"},
	{"hourglass", "⏳"},
	{"phone", "📱"},
	{"laptop", "💻"},
	{"keyboard", "⌨️"},
	{"desktop", "🖥️"},
	{"printer", "🖨️"},
	{"gear", "⚙️"},
	{"wrench", "🔧"},
	{"hammer", "🔨"},
	{"link", "🔗"},
	{"lock", "🔒"},
	{"unlock", "🔓"},
	{"key", "🔑"},
	{"magnifying_glass", "🔍"},
	{"bulb", "💡"},
	{"battery", "🔋"},
	{"package", "📦"},
	{"gift", "🎁"},
	{"balloon", "🎈"},
	{"trophy", "🏆"},
	{"medal", "🏅"},
	{"crown", "👑"},
	{"gem", "💎"},
	{"money", "💰"},
	{"dollar", "💵"},
	{"credit_card", "💳"},
	{"bell", "🔔"},
	{"megaphone", "📢"},
	{"mail", "📧"},
	{"inbox", "📥"},

	// Transport & Places
	{"car", "🚗"},
	{"bus", "🚌"},
	{"train", "🚆"},
	{"airplane", "✈️"},
	{"rocket", "🚀"},
	{"ship", "🚢"},
	{"bike", "🚲"},
	{"house", "🏠"},
	{"office", "🏢"},
	{"hospital", "🏥"},
	{"school", "🏫"},
	{"tent", "⛺"},
	{"mountain", "⛰️"},
	{"beach", "🏖️"},
	{"globe", "🌍"},
	{"world", "🌎"},

	// Flags & Misc
	{"flag_white", "🏳️"},
	{"flag_black", "🏴"},
	{"checkered_flag", "🏁"},
	{"triangular_flag", "🚩"},
	{"red_flag", "🚩"},
	{"tada", "🎉"},
	{"confetti", "🎊"},
	{"ribbon", "🎀"},
	{"medal_sports", "🏅"},
	{"dart", "🎯"},
	{"dice", "🎲"},
	{"puzzle", "🧩"},
	{"recycle", "♻️"},
	{"peace_symbol", "☮️"},
	{"infinity", "♾️"},

	// Arrows & UI
	{"arrow_up", "⬆️"},
	{"arrow_down", "⬇️"},
	{"arrow_left", "⬅️"},
	{"arrow_right", "➡️"},
	{"back", "🔙"},
	{"soon", "🔜"},
	{"top", "🔝"},
	{"new", "🆕"},
	{"free", "🆓"},
	{"cool", "🆒"},
	{"ok", "🆗"},
	{"sos", "🆘"},
	{"no_entry", "⛔"},
	{"prohibited", "🚫"},
	{"stop_sign", "🛑"},

	// Developer/Tech
	{"bug_report", "🪲"},
	{"test_tube", "🧪"},
	{"dna", "🧬"},
	{"satellite", "🛰️"},
	{"telescope", "🔭"},
	{"microscope", "🔬"},
	{"pill", "💊"},
	{"syringe", "💉"},
	{"stethoscope", "🩺"},
	{"shield", "🛡️"},
	{"scroll", "📜"},
	{"map", "🗺️"},
	{"compass", "🧭"},
	{"label", "🏷️"},
	{"bookmark", "🔖"},
	{"paperclip", "📎"},
	{"pushpin", "📌"},
	{"scissors", "✂️"},
	{"wastebasket", "🗑️"},
}

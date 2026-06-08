package router

import "strings"

type Intent string

const (
	IntentMemoryAdd   Intent = "memory_add"
	IntentMemoryQuery Intent = "memory_query"
	IntentChat        Intent = "chat"
	IntentAction      Intent = "action"
	IntentUnknown     Intent = "unknown"
)

func Classify(message string) Intent {
	text := strings.ToLower(strings.TrimSpace(message))
	switch {
	case text == "":
		return IntentUnknown
	case containsAny(text, "nhớ là", "từ nay", "lưu lại", "remember that", "save this", "note that"):
		return IntentMemoryAdd
	case containsAny(text, "mày nhớ gì", "nhớ gì", "server chính", "đã lưu", "what do you remember", "recall", "saved memory"):
		return IntentMemoryQuery
	case strings.HasPrefix(text, "chạy ") || containsAny(text, "đọc file", "ghi file", "run ", "read file", "write file"):
		return IntentAction
	default:
		return IntentChat
	}
}

func ParseIntent(value string) Intent {
	switch Intent(strings.ToLower(strings.TrimSpace(value))) {
	case IntentMemoryAdd:
		return IntentMemoryAdd
	case IntentMemoryQuery:
		return IntentMemoryQuery
	case IntentAction:
		return IntentAction
	case IntentChat:
		return IntentChat
	default:
		return IntentUnknown
	}
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

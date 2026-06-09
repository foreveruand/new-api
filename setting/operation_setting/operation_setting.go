package operation_setting

import (
	"strings"

	"github.com/QuantumNous/new-api/types"
)

var DemoSiteEnabled = false
var SelfUseModeEnabled = false

var AutomaticDisableKeywords = []string{
	"Your credit balance is too low",
	"This organization has been disabled.",
	"You exceeded your current quota",
	"Permission denied",
	"The security token included in the request is invalid",
	"Operation not allowed",
	"Your account is not authorized",
}

func AutomaticDisableKeywordsToString() string {
	return strings.Join(AutomaticDisableKeywords, "\n")
}

func AutomaticDisableKeywordsFromString(s string) {
	AutomaticDisableKeywords = []string{}
	ak := strings.Split(s, "\n")
	for _, k := range ak {
		k = strings.TrimSpace(k)
		k = strings.ToLower(k)
		if k != "" {
			AutomaticDisableKeywords = append(AutomaticDisableKeywords, k)
		}
	}
}

func MatchAutomaticDisableKeyword(message string) bool {
	message = strings.ToLower(strings.TrimSpace(message))
	if message == "" {
		return false
	}
	for _, keyword := range AutomaticDisableKeywords {
		keyword = strings.ToLower(strings.TrimSpace(keyword))
		if keyword != "" && strings.Contains(message, keyword) {
			return true
		}
	}
	return false
}

func ApplyDisableKeywordErrorCode(err *types.NewAPIError) bool {
	if err == nil || !MatchAutomaticDisableKeyword(err.Error()) {
		return false
	}
	err.SetErrorCode(types.ErrorCodeChannelFailureKeyword)
	return true
}

package operation_setting

import (
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
)

var AutomaticRetryKeywords = []string{}

// AutomaticErrorCodeMapping is kept for compatibility with old code paths. Its
// values are ignored; only keys are used as retry keywords.
var AutomaticErrorCodeMapping = map[string]string{}

func AutomaticRetryKeywordsToString() string {
	return strings.Join(AutomaticRetryKeywords, "\n")
}

func AutomaticRetryKeywordsFromString(s string) error {
	keywords, mapping := parseRetryKeywords(s)
	AutomaticRetryKeywords = keywords
	AutomaticErrorCodeMapping = mapping
	return nil
}

func ParseRetryKeywords(s string) ([]string, error) {
	keywords, _ := parseRetryKeywords(s)
	return keywords, nil
}

func AutomaticErrorCodeMappingToString() string {
	return AutomaticRetryKeywordsToString()
}

func AutomaticErrorCodeMappingFromString(s string) error {
	return AutomaticRetryKeywordsFromString(s)
}

func ParseErrorCodeMapping(s string) (map[string]string, error) {
	keywords, _ := parseRetryKeywords(s)
	mapping := make(map[string]string, len(keywords))
	for _, keyword := range keywords {
		mapping[keyword] = string(types.ErrorCodeChannelRetryKeyword)
	}
	return mapping, nil
}

func ApplyRetryKeywordErrorCode(err *types.NewAPIError) bool {
	if err == nil || !MatchAutomaticRetryKeyword(err.Error()) {
		return false
	}
	err.SetErrorCode(types.ErrorCodeChannelRetryKeyword)
	return true
}

func ApplyErrorCodeMapping(err *types.NewAPIError) bool {
	return ApplyRetryKeywordErrorCode(err)
}

func MatchAutomaticRetryKeyword(message string) bool {
	message = strings.ToLower(strings.TrimSpace(message))
	if message == "" {
		return false
	}
	for _, keyword := range automaticRetryKeywordCandidates() {
		keyword = strings.ToLower(strings.TrimSpace(keyword))
		if keyword != "" && strings.Contains(message, keyword) {
			return true
		}
	}
	return false
}

func automaticRetryKeywordCandidates() []string {
	if len(AutomaticRetryKeywords) > 0 {
		return AutomaticRetryKeywords
	}
	if len(AutomaticErrorCodeMapping) == 0 {
		return nil
	}
	keys := make([]string, 0, len(AutomaticErrorCodeMapping))
	for pattern := range AutomaticErrorCodeMapping {
		keys = append(keys, pattern)
	}
	sort.Strings(keys)
	return keys
}

func parseRetryKeywords(s string) ([]string, map[string]string) {
	s = strings.TrimSpace(s)
	if s == "" || s == "{}" {
		return []string{}, map[string]string{}
	}
	if strings.HasPrefix(s, "{") {
		var raw map[string]string
		if err := common.Unmarshal([]byte(s), &raw); err == nil {
			keywords := make([]string, 0, len(raw))
			for pattern := range raw {
				pattern = strings.TrimSpace(pattern)
				if pattern != "" {
					keywords = append(keywords, pattern)
				}
			}
			sort.Strings(keywords)
			return keywords, raw
		}
	}
	keywords := make([]string, 0)
	mapping := map[string]string{}
	for _, keyword := range strings.Split(s, "\n") {
		keyword = strings.TrimSpace(keyword)
		if keyword != "" {
			keywords = append(keywords, keyword)
			mapping[keyword] = string(types.ErrorCodeChannelRetryKeyword)
		}
	}
	return keywords, mapping
}

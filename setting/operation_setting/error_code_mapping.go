package operation_setting

import (
	"fmt"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
)

var AutomaticErrorCodeMapping = map[string]string{}

func AutomaticErrorCodeMappingToString() string {
	if len(AutomaticErrorCodeMapping) == 0 {
		return "{}"
	}
	data, err := common.Marshal(AutomaticErrorCodeMapping)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func AutomaticErrorCodeMappingFromString(s string) error {
	mapping, err := parseErrorCodeMapping(s)
	if err != nil {
		return err
	}
	AutomaticErrorCodeMapping = mapping
	return nil
}

func ParseErrorCodeMapping(s string) (map[string]string, error) {
	return parseErrorCodeMapping(s)
}

func ApplyErrorCodeMapping(err *types.NewAPIError) bool {
	if err == nil || len(AutomaticErrorCodeMapping) == 0 {
		return false
	}
	message := strings.ToLower(err.Error())
	if message == "" {
		return false
	}

	keys := make([]string, 0, len(AutomaticErrorCodeMapping))
	for pattern := range AutomaticErrorCodeMapping {
		keys = append(keys, pattern)
	}
	sort.Slice(keys, func(i, j int) bool {
		return len(keys[i]) > len(keys[j])
	})

	for _, pattern := range keys {
		if strings.Contains(message, strings.ToLower(pattern)) {
			err.SetErrorCode(types.ErrorCode(AutomaticErrorCodeMapping[pattern]))
			return true
		}
	}
	return false
}

func parseErrorCodeMapping(s string) (map[string]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return map[string]string{}, nil
	}
	var raw map[string]string
	if err := common.Unmarshal([]byte(s), &raw); err != nil {
		return nil, err
	}
	mapping := make(map[string]string, len(raw))
	for pattern, code := range raw {
		pattern = strings.TrimSpace(pattern)
		code = strings.TrimSpace(code)
		if pattern == "" || code == "" {
			return nil, fmt.Errorf("error code mapping requires non-empty message patterns and error codes")
		}
		mapping[pattern] = code
	}
	return mapping, nil
}

package relay

import (
	"fmt"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

func validateNonEmptyUsage(c *gin.Context, info *relaycommon.RelayInfo, usage *dto.Usage) *types.NewAPIError {
	if usage != nil {
		if usage.PromptTokens+usage.CompletionTokens > 0 {
			return nil
		}
		if usage.TotalTokens > 0 {
			usage.PromptTokens = usage.TotalTokens
			return nil
		}
	}
	if c != nil && c.Writer != nil && c.Writer.Written() {
		return nil
	}
	model := ""
	channelID := 0
	if info != nil {
		model = info.OriginModelName
		if info.ChannelMeta != nil {
			channelID = info.ChannelId
		}
	}
	return service.EmptyUpstreamResponseError(fmt.Sprintf("empty upstream response, channelId %d, model %s", channelID, model))
}

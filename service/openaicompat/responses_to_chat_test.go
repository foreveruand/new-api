package openaicompat

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestResponsesResponseToChatCompletionsResponseTreatsEmptyOutputAsFailureWhenEnabled(t *testing.T) {
	old := common.RetryEmptyResponseEnabled
	common.RetryEmptyResponseEnabled = true
	t.Cleanup(func() {
		common.RetryEmptyResponseEnabled = old
	})

	resp := &dto.OpenAIResponsesResponse{
		ID:    "resp-test",
		Model: "gpt-test",
		Output: []dto.ResponsesOutput{
			{
				Type: "message",
				Role: "assistant",
				Content: []dto.ResponsesOutputContent{
					{Type: "output_text", Text: ""},
				},
			},
		},
		Usage: &dto.Usage{
			InputTokens:  10,
			OutputTokens: 0,
			TotalTokens:  10,
		},
	}

	usage, chatResp, err := ResponsesResponseToChatCompletionsResponse(resp, "chat-test")

	require.Error(t, err)
	require.Nil(t, usage)
	require.Nil(t, chatResp)
}

func TestResponsesResponseToChatCompletionsResponseAllowsEmptyOutputWhenRetryDisabled(t *testing.T) {
	old := common.RetryEmptyResponseEnabled
	common.RetryEmptyResponseEnabled = false
	t.Cleanup(func() {
		common.RetryEmptyResponseEnabled = old
	})

	resp := &dto.OpenAIResponsesResponse{
		ID:    "resp-test",
		Model: "gpt-test",
		Output: []dto.ResponsesOutput{
			{
				Type: "message",
				Role: "assistant",
				Content: []dto.ResponsesOutputContent{
					{Type: "output_text", Text: ""},
				},
			},
		},
		Usage: &dto.Usage{
			InputTokens:  10,
			OutputTokens: 0,
			TotalTokens:  10,
		},
	}

	usage, chatResp, err := ResponsesResponseToChatCompletionsResponse(resp, "chat-test")

	require.NoError(t, err)
	require.NotNil(t, usage)
	require.NotNil(t, chatResp)
	require.Equal(t, "", chatResp.Choices[0].Message.StringContent())
}


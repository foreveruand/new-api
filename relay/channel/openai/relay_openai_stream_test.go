package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupOpenAIStreamTest(t *testing.T, body string) (*gin.Context, *httptest.ResponseRecorder, *relaycommon.RelayInfo, *http.Response) {
	t.Helper()

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	info := &relaycommon.RelayInfo{
		IsStream:          true,
		RelayFormat:       types.RelayFormatOpenAI,
		RelayMode:         relayconstant.RelayModeChatCompletions,
		UpstreamModelName: "gpt-test",
	}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	return c, recorder, info, resp
}

func withRetryEmptyResponseEnabled(t *testing.T, enabled bool) {
	t.Helper()

	old := common.RetryEmptyResponseEnabled
	common.RetryEmptyResponseEnabled = enabled
	t.Cleanup(func() {
		common.RetryEmptyResponseEnabled = old
	})
}

func TestOaiStreamHandlerTreatsUsageOnlyStreamAsEmptyWhenEnabled(t *testing.T) {
	withRetryEmptyResponseEnabled(t, true)

	body := "data: {\"id\":\"\",\"object\":\"chat.completion.chunk\",\"created\":0,\"model\":\"gpt-test\",\"choices\":[],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":0,\"total_tokens\":10}}\n\n" +
		"data: [DONE]\n"
	c, recorder, info, resp := setupOpenAIStreamTest(t, body)

	usage, newAPIError := OaiStreamHandler(c, info, resp)

	require.Nil(t, usage)
	require.NotNil(t, newAPIError)
	require.Equal(t, types.ErrorCodeEmptyResponse, newAPIError.GetErrorCode())
	require.Equal(t, http.StatusBadGateway, newAPIError.StatusCode)
	require.Empty(t, recorder.Body.String())
}

func TestOaiStreamHandlerDoesNotFlushInitialEmptyChunkBeforeEmptyFailure(t *testing.T) {
	withRetryEmptyResponseEnabled(t, true)

	body := "data: {\"id\":\"chatcmpl-test\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"gpt-test\",\"choices\":[{\"delta\":{\"role\":\"assistant\",\"content\":\"\"},\"finish_reason\":null,\"index\":0}],\"usage\":null}\n\n" +
		"data: {\"id\":\"chatcmpl-test\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"gpt-test\",\"choices\":[],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":0,\"total_tokens\":10}}\n\n" +
		"data: [DONE]\n"
	c, recorder, info, resp := setupOpenAIStreamTest(t, body)

	usage, newAPIError := OaiStreamHandler(c, info, resp)

	require.Nil(t, usage)
	require.NotNil(t, newAPIError)
	require.Equal(t, types.ErrorCodeEmptyResponse, newAPIError.GetErrorCode())
	require.Empty(t, recorder.Body.String())
}

func TestOaiStreamHandlerAllowsUsageOnlyStreamWhenEmptyRetryDisabled(t *testing.T) {
	withRetryEmptyResponseEnabled(t, false)

	body := "data: {\"id\":\"\",\"object\":\"chat.completion.chunk\",\"created\":0,\"model\":\"gpt-test\",\"choices\":[],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":0,\"total_tokens\":10}}\n\n" +
		"data: [DONE]\n"
	c, recorder, info, resp := setupOpenAIStreamTest(t, body)

	usage, newAPIError := OaiStreamHandler(c, info, resp)

	require.NotNil(t, usage)
	require.Nil(t, newAPIError)
	require.Contains(t, recorder.Body.String(), `"choices":[]`)
	require.Contains(t, recorder.Body.String(), "data: [DONE]")
}

func TestOaiStreamHandlerFlushesBufferedEmptyChunkAfterContent(t *testing.T) {
	withRetryEmptyResponseEnabled(t, true)

	body := "data: {\"id\":\"chatcmpl-test\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"gpt-test\",\"choices\":[{\"delta\":{\"role\":\"assistant\",\"content\":\"\"},\"finish_reason\":null,\"index\":0}],\"usage\":null}\n\n" +
		"data: {\"id\":\"chatcmpl-test\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"gpt-test\",\"choices\":[{\"delta\":{\"content\":\"hello\"},\"finish_reason\":null,\"index\":0}],\"usage\":null}\n\n" +
		"data: {\"id\":\"chatcmpl-test\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"gpt-test\",\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\",\"index\":0}],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":1,\"total_tokens\":11}}\n\n" +
		"data: [DONE]\n"
	c, recorder, info, resp := setupOpenAIStreamTest(t, body)

	usage, newAPIError := OaiStreamHandler(c, info, resp)

	require.NotNil(t, usage)
	require.Nil(t, newAPIError)
	require.Contains(t, recorder.Body.String(), `"content":""`)
	require.Contains(t, recorder.Body.String(), `"content":"hello"`)
	require.Contains(t, recorder.Body.String(), "data: [DONE]")
}

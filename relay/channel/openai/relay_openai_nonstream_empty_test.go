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

func setupOpenAINonStreamTest(t *testing.T, body string) (*gin.Context, *httptest.ResponseRecorder, *relaycommon.RelayInfo, *http.Response) {
	t.Helper()

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	info := &relaycommon.RelayInfo{
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

func TestOpenaiHandlerTreatsEmptyChoicesAsFailureWhenEnabled(t *testing.T) {
	old := common.RetryEmptyResponseEnabled
	common.RetryEmptyResponseEnabled = true
	t.Cleanup(func() {
		common.RetryEmptyResponseEnabled = old
	})

	body := `{"id":"chatcmpl-test","object":"chat.completion","created":1,"model":"gpt-test","choices":[],"usage":{"prompt_tokens":10,"completion_tokens":0,"total_tokens":10}}`
	c, recorder, info, resp := setupOpenAINonStreamTest(t, body)

	usage, newAPIError := OpenaiHandler(c, info, resp)

	require.Nil(t, usage)
	require.NotNil(t, newAPIError)
	require.Equal(t, types.ErrorCodeEmptyResponse, newAPIError.GetErrorCode())
	require.Equal(t, http.StatusBadGateway, newAPIError.StatusCode)
	require.Empty(t, recorder.Body.String())
}


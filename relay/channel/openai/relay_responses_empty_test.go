package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupResponsesStreamTest(t *testing.T, body string) (*gin.Context, *httptest.ResponseRecorder, *relaycommon.RelayInfo, *http.Response) {
	t.Helper()

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	info := &relaycommon.RelayInfo{
		IsStream:          true,
		RelayFormat:       types.RelayFormatOpenAI,
		RelayMode:         relayconstant.RelayModeResponses,
		UpstreamModelName: "gpt-test",
	}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	return c, recorder, info, resp
}

func TestOaiResponsesStreamHandlerTreatsEmptyResponseAsFailureWhenEnabled(t *testing.T) {
	withRetryEmptyResponseEnabled(t, true)

	body := "data: {\"type\":\"response.created\",\"response\":{\"model\":\"gpt-test\",\"created_at\":1}}\n\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"model\":\"gpt-test\",\"created_at\":1,\"output\":[],\"usage\":{\"input_tokens\":10,\"output_tokens\":0,\"total_tokens\":10}}}\n\n" +
		"data: [DONE]\n"
	c, recorder, info, resp := setupResponsesStreamTest(t, body)

	usage, newAPIError := OaiResponsesStreamHandler(c, info, resp)

	require.Nil(t, usage)
	require.NotNil(t, newAPIError)
	require.Equal(t, types.ErrorCodeEmptyResponse, newAPIError.GetErrorCode())
	require.Equal(t, http.StatusBadGateway, newAPIError.StatusCode)
	require.Empty(t, recorder.Body.String())
}

func TestOaiResponsesToChatStreamHandlerTreatsEmptyResponseAsFailureWhenEnabled(t *testing.T) {
	withRetryEmptyResponseEnabled(t, true)

	body := "data: {\"type\":\"response.created\",\"response\":{\"model\":\"gpt-test\",\"created_at\":1}}\n\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"model\":\"gpt-test\",\"created_at\":1,\"output\":[],\"usage\":{\"input_tokens\":10,\"output_tokens\":0,\"total_tokens\":10}}}\n\n" +
		"data: [DONE]\n"
	c, recorder, info, resp := setupResponsesStreamTest(t, body)

	usage, newAPIError := OaiResponsesToChatStreamHandler(c, info, resp)

	require.Nil(t, usage)
	require.NotNil(t, newAPIError)
	require.Equal(t, types.ErrorCodeEmptyResponse, newAPIError.GetErrorCode())
	require.Equal(t, http.StatusBadGateway, newAPIError.StatusCode)
	require.Empty(t, recorder.Body.String())
}

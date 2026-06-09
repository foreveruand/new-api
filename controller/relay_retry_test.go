package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestShouldRetryEmptyResponseEvenWithSuccessfulStatus(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	newAPIError := types.NewOpenAIError(
		fmt.Errorf("empty response"),
		types.ErrorCodeEmptyResponse,
		http.StatusOK,
	)

	require.True(t, shouldRetry(c, newAPIError, 1))
}

func TestShouldRetryFailureKeywordChannelError(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	newAPIError := types.NewOpenAIError(
		fmt.Errorf("Your credit balance is too low"),
		types.ErrorCodeChannelFailureKeyword,
		http.StatusBadRequest,
	)

	require.True(t, shouldRetry(c, newAPIError, 1))
}

package controller

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fakeRelayBilling struct {
	preConsumed    int
	reserveTargets []int
	settleActuals  []int
	refunded       bool
}

func (b *fakeRelayBilling) Settle(actualQuota int) error {
	b.settleActuals = append(b.settleActuals, actualQuota)
	b.preConsumed = actualQuota
	return nil
}

func (b *fakeRelayBilling) Refund(_ *gin.Context) {
	b.refunded = true
}

func (b *fakeRelayBilling) NeedsRefund() bool {
	return b.preConsumed > 0 && !b.refunded
}

func (b *fakeRelayBilling) GetPreConsumedQuota() int {
	return b.preConsumed
}

func (b *fakeRelayBilling) Reserve(targetQuota int) error {
	b.reserveTargets = append(b.reserveTargets, targetQuota)
	if targetQuota > b.preConsumed {
		b.preConsumed = targetQuota
	}
	return nil
}

func configureRelayBillingTestSettings(t *testing.T, modelName, tieredExpr string) {
	t.Helper()

	savedConfig := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		savedConfig[key] = value
		return nil
	}))
	savedModelRatio := ratio_setting.ModelRatio2JSONString()
	savedModelPrice := ratio_setting.ModelPrice2JSONString()
	savedGroupRatio := ratio_setting.GroupRatio2JSONString()
	savedGroupGroupRatio := ratio_setting.GroupGroupRatio2JSONString()

	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(savedConfig))
		require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(savedModelRatio))
		require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(savedModelPrice))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(savedGroupRatio))
		require.NoError(t, ratio_setting.UpdateGroupGroupRatioByJSONString(savedGroupGroupRatio))
	})

	modelRatioJSON, err := common.Marshal(map[string]float64{modelName: 1})
	require.NoError(t, err)
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(string(modelRatioJSON)))
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"free":0,"paid":1}`))
	require.NoError(t, ratio_setting.UpdateGroupGroupRatioByJSONString(`{}`))

	billingModeJSON := `{}`
	billingExprJSON := `{}`
	if tieredExpr != "" {
		modeBytes, err := common.Marshal(map[string]string{modelName: billing_setting.BillingModeTieredExpr})
		require.NoError(t, err)
		exprBytes, err := common.Marshal(map[string]string{modelName: tieredExpr})
		require.NoError(t, err)
		billingModeJSON = string(modeBytes)
		billingExprJSON = string(exprBytes)
	}
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"quota_setting.enable_free_model_pre_consume": "false",
		"billing_setting.billing_mode":                billingModeJSON,
		"billing_setting.billing_expr":                billingExprJSON,
	}))
}

func newRelayBillingTestContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
	return ctx
}

func newRelayBillingTestInfo(modelName string) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		OriginModelName: modelName,
		UserGroup:       "default",
		UsingGroup:      "auto",
	}
}

func TestPrepareRelayBillingCreatesSessionWhenRetryLeavesFreeGroup(t *testing.T) {
	modelName := "relay-billing-ratio-cross-group"
	configureRelayBillingTestSettings(t, modelName, "")
	ctx := newRelayBillingTestContext()
	info := newRelayBillingTestInfo(modelName)

	oldPreConsume := preConsumeRelayBilling
	preConsumeCalls := 0
	preConsumeRelayBilling = func(_ *gin.Context, quota int, relayInfo *relaycommon.RelayInfo) *types.NewAPIError {
		preConsumeCalls++
		billing := &fakeRelayBilling{preConsumed: quota}
		relayInfo.Billing = billing
		relayInfo.FinalPreConsumedQuota = quota
		return nil
	}
	t.Cleanup(func() {
		preConsumeRelayBilling = oldPreConsume
	})

	ctx.Set("auto_group", "free")
	apiErr := prepareRelayBilling(ctx, info, 1000, &types.TokenCountMeta{})
	require.Nil(t, apiErr)
	require.True(t, info.PriceData.FreeModel)
	require.Equal(t, 0.0, info.PriceData.GroupRatioInfo.GroupRatio)
	require.Nil(t, info.Billing)
	require.Equal(t, 0, preConsumeCalls)

	ctx.Set("auto_group", "paid")
	apiErr = prepareRelayBilling(ctx, info, 1000, &types.TokenCountMeta{})
	require.Nil(t, apiErr)
	require.False(t, info.PriceData.FreeModel)
	require.Equal(t, "paid", info.UsingGroup)
	require.Equal(t, 1.0, info.PriceData.GroupRatioInfo.GroupRatio)
	require.Greater(t, info.PriceData.QuotaToPreConsume, 0)
	require.Equal(t, info.PriceData.QuotaToPreConsume, info.Billing.GetPreConsumedQuota())
	require.Equal(t, 1, preConsumeCalls)
}

func TestPrepareRelayBillingReservesHigherRetryQuota(t *testing.T) {
	modelName := "relay-billing-ratio-reserve"
	configureRelayBillingTestSettings(t, modelName, "")
	ctx := newRelayBillingTestContext()
	ctx.Set("auto_group", "paid")

	billing := &fakeRelayBilling{preConsumed: 100}
	info := newRelayBillingTestInfo(modelName)
	info.Billing = billing

	apiErr := prepareRelayBilling(ctx, info, 1000, &types.TokenCountMeta{})
	require.Nil(t, apiErr)
	require.Equal(t, []int{info.PriceData.QuotaToPreConsume}, billing.reserveTargets)
	require.Greater(t, billing.GetPreConsumedQuota(), 100)
}

func TestPrepareRelayBillingRefreshesTieredSnapshotForRetryGroup(t *testing.T) {
	modelName := "relay-billing-tiered-cross-group"
	configureRelayBillingTestSettings(t, modelName, `tier("base", p * 2)`)
	ctx := newRelayBillingTestContext()
	info := newRelayBillingTestInfo(modelName)

	oldPreConsume := preConsumeRelayBilling
	preConsumeRelayBilling = func(_ *gin.Context, quota int, relayInfo *relaycommon.RelayInfo) *types.NewAPIError {
		billing := &fakeRelayBilling{preConsumed: quota}
		relayInfo.Billing = billing
		relayInfo.FinalPreConsumedQuota = quota
		return nil
	}
	t.Cleanup(func() {
		preConsumeRelayBilling = oldPreConsume
	})

	ctx.Set("auto_group", "free")
	apiErr := prepareRelayBilling(ctx, info, 1000, &types.TokenCountMeta{})
	require.Nil(t, apiErr)
	require.True(t, info.PriceData.FreeModel)
	require.NotNil(t, info.TieredBillingSnapshot)
	require.Equal(t, 0.0, info.TieredBillingSnapshot.GroupRatio)
	require.Equal(t, 0, info.TieredBillingSnapshot.EstimatedQuotaAfterGroup)
	require.Nil(t, info.Billing)

	ctx.Set("auto_group", "paid")
	apiErr = prepareRelayBilling(ctx, info, 1000, &types.TokenCountMeta{})
	require.Nil(t, apiErr)
	require.False(t, info.PriceData.FreeModel)
	require.NotNil(t, info.TieredBillingSnapshot)
	require.NotNil(t, info.BillingRequestInput)
	require.Equal(t, 1.0, info.TieredBillingSnapshot.GroupRatio)
	require.Equal(t, 1.0, info.PriceData.GroupRatioInfo.GroupRatio)
	require.Greater(t, info.TieredBillingSnapshot.EstimatedQuotaAfterGroup, 0)
	require.Equal(t, info.TieredBillingSnapshot.EstimatedQuotaAfterGroup, info.Billing.GetPreConsumedQuota())
}

func TestPrepareRelayBillingKeepsPaidSessionWhenRetryGroupIsFree(t *testing.T) {
	modelName := "relay-billing-paid-to-free"
	configureRelayBillingTestSettings(t, modelName, "")
	ctx := newRelayBillingTestContext()
	ctx.Set("auto_group", "free")

	billing := &fakeRelayBilling{preConsumed: 1000}
	info := newRelayBillingTestInfo(modelName)
	info.Billing = billing

	apiErr := prepareRelayBilling(ctx, info, 1000, &types.TokenCountMeta{})
	require.Nil(t, apiErr)
	require.True(t, info.PriceData.FreeModel)
	require.Equal(t, 0.0, info.PriceData.GroupRatioInfo.GroupRatio)
	actualBilling, ok := info.Billing.(*fakeRelayBilling)
	require.True(t, ok)
	require.Same(t, billing, actualBilling)
	require.Empty(t, billing.reserveTargets)

	require.NoError(t, service.SettleBilling(ctx, info, 0))
	require.Equal(t, []int{0}, billing.settleActuals)
}

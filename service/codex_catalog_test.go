package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupCodexCatalogTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.RedisEnabled = false

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.CodexCatalogCandidate{}, &model.CodexCatalogSnapshot{}))

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func TestValidateCodexCandidateUsesListInput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Input  []map[string]any `json:"input"`
			Stream bool             `json:"stream"`
		}
		require.NoError(t, common.DecodeJson(r.Body, &payload))
		require.Len(t, payload.Input, 1)
		assert.True(t, payload.Stream)
		assert.Equal(t, "message", payload.Input[0]["type"])
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"O\"}\n\n" +
			"data: {\"type\":\"response.output_text.delta\",\"delta\":\"k\"}\n\n" +
			"data: [DONE]\n\n"))
	}))
	defer server.Close()

	channel := &model.Channel{
		Id:   5,
		Type: constant.ChannelTypeCodex,
		Key:  `{"access_token":"access-test","account_id":"acct-test"}`,
	}
	channel.BaseURL = common.GetPointer(server.URL)

	output, err := validateCodexCandidate(context.Background(), channel, "gpt-5.4")
	require.NoError(t, err)
	assert.Equal(t, "Ok", output)
}

func TestValidateCodexCandidateDetectsStreamWithoutContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("event: response.output_text.delta\n" +
			"data: {\"type\":\"response.output_text.delta\",\"delta\":\"Ok\"}\n\n" +
			"event: response.output_text.done\n" +
			"data: {\"type\":\"response.output_text.done\",\"text\":\"Ok\"}\n\n"))
	}))
	defer server.Close()

	channel := &model.Channel{
		Id:   5,
		Type: constant.ChannelTypeCodex,
		Key:  `{"access_token":"access-test","account_id":"acct-test"}`,
	}
	channel.BaseURL = common.GetPointer(server.URL)

	output, err := validateCodexCandidate(context.Background(), channel, "gpt-5.4")
	require.NoError(t, err)
	assert.Equal(t, "Ok", output)
}

func TestSyncCodexChannelModelsRejectsEmptyPromotion(t *testing.T) {
	channel := &model.Channel{Id: 5, Models: "gpt-5.4,gpt-5.4-mini"}

	err := syncCodexChannelModels(channel, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty promoted catalog")
	assert.Equal(t, "gpt-5.4,gpt-5.4-mini", channel.Models)
}

func TestDefaultCodexCatalogPolicyIncludesOfficialGPT56Overrides(t *testing.T) {
	policy := defaultCodexCatalogPolicy()

	expectedNames := map[string]string{
		"gpt-5.6-sol":   "OpenAI Codex GPT-5.6 Sol",
		"gpt-5.6-terra": "OpenAI Codex GPT-5.6 Terra",
		"gpt-5.6-luna":  "OpenAI Codex GPT-5.6 Luna",
	}
	expectedEfforts := map[string][]string{
		"gpt-5.6-sol":   {"low", "medium", "high", "xhigh", "max", "ultra"},
		"gpt-5.6-terra": {"low", "medium", "high", "xhigh", "max", "ultra"},
		"gpt-5.6-luna":  {"low", "medium", "high", "xhigh", "max"},
	}
	for modelName, displayName := range expectedNames {
		meta, ok := policy.Overrides[modelName]
		require.True(t, ok, modelName)
		assert.Equal(t, displayName, meta.DisplayName)
		assert.Equal(t, 1050000, meta.ContextWindowTokens)
		assert.Equal(t, 1050000, meta.MaxTokens)
		assert.Equal(t, 128000, meta.MaxCompletionTokens)
		assert.Equal(t, constant.EndpointTypeOpenAIResponse, meta.EndpointPreference)
		assert.Equal(t, []constant.EndpointType{constant.EndpointTypeOpenAIResponse, constant.EndpointTypeOpenAI}, meta.SupportedEndpoints)
		assert.Equal(t, expectedEfforts[modelName], meta.SupportedReasoningEfforts)
		assert.ElementsMatch(t, []string{
			"text_input",
			"image_input",
			"text_output",
			"streaming",
			"function_calling",
			"structured_outputs",
			"web_search",
			"file_search",
			"image_generation",
			"code_interpreter",
			"hosted_shell",
			"apply_patch",
			"skills",
			"computer_use",
			"mcp",
			"tool_search",
		}, meta.Capabilities)
	}
}

func TestFallbackCodexModelIDsIncludesOfficialGPT56Models(t *testing.T) {
	fallback := fallbackCodexModelIDs()

	assert.Contains(t, fallback, "gpt-5.6-sol")
	assert.Contains(t, fallback, "gpt-5.6-terra")
	assert.Contains(t, fallback, "gpt-5.6-luna")
}

func TestCodexCatalogCandidateModelIDsCombinesDiscoveryAndCuratedFallback(t *testing.T) {
	candidates := codexCatalogCandidateModelIDs([]string{"gpt-5.4", "codex-auto-review"})

	assert.Equal(t, 1, countModelName(candidates, "gpt-5.4"))
	assert.Contains(t, candidates, "codex-auto-review")
	assert.Contains(t, candidates, "gpt-5.5")
	assert.Contains(t, candidates, "gpt-5.6-sol")
	assert.Contains(t, candidates, "gpt-5.6-terra")
	assert.Contains(t, candidates, "gpt-5.6-luna")
}

func TestDefaultCodexCatalogPolicyDeniesInternalAutoReviewModel(t *testing.T) {
	policy := defaultCodexCatalogPolicy()

	assert.True(t, isDeniedCodexModel("codex-auto-review", policy))
}

func TestIsExpectedCodexValidationOutputAcceptsTerminalPeriod(t *testing.T) {
	assert.True(t, isExpectedCodexValidationOutput("Ok"))
	assert.True(t, isExpectedCodexValidationOutput(" Ok. "))
	assert.False(t, isExpectedCodexValidationOutput("Okay"))
}

func countModelName(models []string, target string) int {
	count := 0
	for _, modelName := range models {
		if modelName == target {
			count++
		}
	}
	return count
}

func TestCodexCatalogSignatureChangesWithPolicy(t *testing.T) {
	models := []string{"gpt-5.6-sol", "gpt-5.4"}
	basePolicy := defaultCodexCatalogPolicy()
	baseSignature, err := codexCatalogSignature(models, basePolicy)
	require.NoError(t, err)

	changedPolicy := defaultCodexCatalogPolicy()
	meta := changedPolicy.Overrides["gpt-5.6-sol"]
	meta.MaxCompletionTokens = 64000
	changedPolicy.Overrides["gpt-5.6-sol"] = meta
	changedSignature, err := codexCatalogSignature(models, changedPolicy)
	require.NoError(t, err)

	assert.NotEqual(t, baseSignature, changedSignature)
}

func TestPrioritizeCodexDefaultModel(t *testing.T) {
	reordered := prioritizeCodexDefaultModel([]string{"gpt-5.5", "gpt-5.4", "gpt-5.4-mini"}, "gpt-5.4")
	require.Equal(t, []string{"gpt-5.4", "gpt-5.5", "gpt-5.4-mini"}, reordered)
}

func TestNextCodexCatalogSyncDelay(t *testing.T) {
	location := time.FixedZone("BRT", -3*60*60)

	delay := nextCodexCatalogSyncDelay(time.Date(2026, 7, 7, 3, 30, 0, 0, location))
	assert.Equal(t, 30*time.Minute, delay)

	delay = nextCodexCatalogSyncDelay(time.Date(2026, 7, 7, 4, 30, 0, 0, location))
	assert.Equal(t, 23*time.Hour+30*time.Minute, delay)
}

func TestMergeCodexCatalogMetadataPrefersOverrideAndKeepsContextWindowGroup(t *testing.T) {
	meta := mergeCodexCatalogMetadata(
		"gpt-5.4",
		CodexCatalogMetadata{
			DisplayName: "Source Name",
			OwnedBy:     "source-owner",
		},
		CodexCatalogMetadata{
			DisplayName:               "Override Name",
			EndpointPreference:        constant.EndpointTypeOpenAIResponse,
			SupportedEndpoints:        []constant.EndpointType{constant.EndpointTypeOpenAIResponse, constant.EndpointTypeOpenAI},
			ContextWindowTokens:       1000000,
			MaxTokens:                 1000000,
			MaxCompletionTokens:       128000,
			SupportedReasoningEfforts: []string{"none", "high", "max"},
			Capabilities:              []string{"streaming", "function_calling"},
		},
	)

	assert.Equal(t, "Override Name", meta.DisplayName)
	assert.Equal(t, []constant.EndpointType{constant.EndpointTypeOpenAIResponse, constant.EndpointTypeOpenAI}, meta.SupportedEndpoints)
	assert.Equal(t, 1000000, meta.ContextWindowTokens)
	assert.Equal(t, 1000000, meta.MaxTokens)
	assert.Equal(t, 128000, meta.MaxCompletionTokens)
	assert.Equal(t, []string{"none", "high", "max"}, meta.SupportedReasoningEfforts)
	assert.Equal(t, []string{"streaming", "function_calling"}, meta.Capabilities)
}

func TestPromotedCodexMetadataByModelNameReappliesCurrentPolicyOverrides(t *testing.T) {
	db := setupCodexCatalogTestDB(t)
	endpoints, err := common.Marshal([]string{
		string(constant.EndpointTypeOpenAIResponse),
		string(constant.EndpointTypeOpenAI),
	})
	require.NoError(t, err)

	require.NoError(t, db.Create(&model.CodexCatalogCandidate{
		ChannelID:            9,
		ModelName:            "gpt-5.4",
		Promoted:             true,
		DisplayName:          "Stale GPT-5.4",
		Provider:             "OpenAI Codex",
		OwnedBy:              "codex",
		EndpointPreference:   string(constant.EndpointTypeOpenAIResponse),
		SupportedEndpoints:   string(endpoints),
		ContextWindowTokens:  272000,
		MaxTokens:            272000,
		MaxCompletionTokens:  128000,
		SourceMetadata:       `{}`,
		OverrideMetadata:     `{}`,
		DiscoveryMetadata:    `{}`,
		DiscoveryHash:        "hash",
		LastDiscoveredTime:   common.GetTimestamp(),
		LastSeenTime:         common.GetTimestamp(),
		LastValidatedTime:    common.GetTimestamp(),
		LastPromotedTime:     common.GetTimestamp(),
		Status:               model.CodexCatalogStatusPromoted,
		ValidationState:      "ok",
	}).Error)

	metadataByModel, err := promotedCodexMetadataByModelName([]string{"gpt-5.4"})
	require.NoError(t, err)
	meta, ok := metadataByModel["gpt-5.4"]
	require.True(t, ok)
	assert.Equal(t, "OpenAI Codex GPT-5.4", meta.DisplayName)
	assert.Equal(t, 1000000, meta.ContextWindowTokens)
	assert.Equal(t, 1000000, meta.MaxTokens)
	assert.Equal(t, 128000, meta.MaxCompletionTokens)
}

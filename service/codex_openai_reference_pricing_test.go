package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type codexOpenAIReferencePricingDoer func(*http.Request) (*http.Response, error)

func (doer codexOpenAIReferencePricingDoer) Do(req *http.Request) (*http.Response, error) {
	return doer(req)
}

func codexOpenAIReferencePricingFixture() string {
	return `# Pricing
<div data-content-switcher-pane data-value="standard">
  <small>Prices per 1M tokens.</small>
  rows={[
    ["gpt-5.6-sol", 5, 0.5, 6.25, 30],
    ["gpt-5.6-terra", 2.5, 0.25, 3.125, 15],
    ["gpt-5.6-luna", 1, 0.1, 1.25, 6],
    ["gpt-5.5 (<272K context length)", 5, 0.5, "-", 30],
    ["gpt-future", 7, null, 42],
  ]}
</div>
<div data-content-switcher-pane data-value="batch" hidden>
  rows={[
    ["gpt-5.6-sol", 2.5, 0.25, 3.125, 15],
    ["gpt-5.6-terra", 1.25, 0.125, 1.5625, 7.5],
    ["gpt-5.6-luna", 0.5, 0.05, 0.625, 3],
    ["gpt-5.5 (<272K context length)", 2.5, 0.25, "-", 15],
  ]}
</div>`
}

func TestParseCodexOpenAIStandardPricingUsesOfficialStandardTier(t *testing.T) {
	fetchedAt := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	snapshot, err := parseCodexOpenAIStandardPricing(
		[]byte(codexOpenAIReferencePricingFixture()),
		fetchedAt,
		`"pricing-etag"`,
		"Sat, 18 Jul 2026 21:40:48 GMT",
	)

	require.NoError(t, err)
	assert.Equal(t, CodexOpenAIReferencePricingSourceURL, snapshot.SourceURL)
	assert.Equal(t, fetchedAt, snapshot.FetchedAt)
	assert.Equal(t, `"pricing-etag"`, snapshot.ETag)
	require.Len(t, snapshot.Prices, 5)

	sol := snapshot.Prices["gpt-5.6-sol"]
	assert.Equal(t, 5.0, sol.InputPerMillion)
	assert.Equal(t, 0.5, sol.CachedInputPerMillion)
	require.NotNil(t, sol.CacheWritePerMillion)
	assert.Equal(t, 6.25, *sol.CacheWritePerMillion)
	assert.Equal(t, 30.0, sol.OutputPerMillion)
	assert.Zero(t, sol.MaxCompletionTokens)

	gpt55 := snapshot.Prices["gpt-5.5"]
	assert.Equal(t, 5.0, gpt55.InputPerMillion)
	assert.Nil(t, gpt55.CacheWritePerMillion)
	assert.Equal(t, 30.0, gpt55.OutputPerMillion)
	assert.Equal(t, 7.0, snapshot.Prices["gpt-future"].InputPerMillion)
	assert.Zero(t, snapshot.Prices["gpt-future"].CachedInputPerMillion)
	assert.Equal(t, 42.0, snapshot.Prices["gpt-future"].OutputPerMillion)
}

func TestParseCodexOpenAIReferenceMaxCompletionTokens(t *testing.T) {
	maxCompletionTokens, err := parseCodexOpenAIReferenceMaxCompletionTokens(
		[]byte(`<div>128,000<!-- --> max output tokens</div>`),
	)

	require.NoError(t, err)
	assert.Equal(t, 128000, maxCompletionTokens)
}

func TestParseCodexOpenAIStandardPricingRejectsPartialUpdates(t *testing.T) {
	partial := strings.Replace(
		codexOpenAIReferencePricingFixture(),
		`    ["gpt-5.6-luna", 1, 0.1, 1.25, 6],`+"\n",
		"",
		1,
	)

	_, err := parseCodexOpenAIStandardPricing([]byte(partial), time.Now(), "", "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required model gpt-5.6-luna")
}

func TestFetchCodexOpenAIReferencePricingUsesMarkdownAndETag(t *testing.T) {
	client := codexOpenAIReferencePricingDoer(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, CodexOpenAIReferencePricingSourceURL, req.URL.String())
		assert.Equal(t, "text/markdown", req.Header.Get("Accept"))
		assert.Equal(t, `"old-etag"`, req.Header.Get("If-None-Match"))
		assert.Equal(t, "Fri, 17 Jul 2026 20:00:00 GMT", req.Header.Get("If-Modified-Since"))
		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Etag":          []string{`"new-etag"`},
				"Last-Modified": []string{"Sat, 18 Jul 2026 21:40:48 GMT"},
			},
			Body: io.NopCloser(strings.NewReader(codexOpenAIReferencePricingFixture())),
		}, nil
	})

	snapshot, notModified, err := fetchCodexOpenAIReferencePricing(context.Background(), client, `"old-etag"`, "Fri, 17 Jul 2026 20:00:00 GMT")

	require.NoError(t, err)
	assert.False(t, notModified)
	assert.Equal(t, `"new-etag"`, snapshot.ETag)
	assert.Equal(t, 2.5, snapshot.Prices["gpt-5.6-terra"].InputPerMillion)
}

func TestFetchCodexOpenAIReferencePricingKeepsSnapshotOnNotModified(t *testing.T) {
	client := codexOpenAIReferencePricingDoer(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotModified,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	})

	_, notModified, err := fetchCodexOpenAIReferencePricing(context.Background(), client, `"same"`, "")

	require.NoError(t, err)
	assert.True(t, notModified)
}

func TestFetchCodexOpenAIReferenceMaxCompletionTokensUsesOfficialModelPage(t *testing.T) {
	client := codexOpenAIReferencePricingDoer(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, codexOpenAIReferenceModelURL("gpt-5.6-sol"), req.URL.String())
		assert.Equal(t, "text/html", req.Header.Get("Accept"))
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`<div>128,000<!-- --> max output tokens</div>`)),
		}, nil
	})

	maxCompletionTokens, err := fetchCodexOpenAIReferenceMaxCompletionTokens(
		context.Background(),
		client,
		"gpt-5.6-sol",
	)

	require.NoError(t, err)
	assert.Equal(t, 128000, maxCompletionTokens)
}

func TestDefaultCodexOpenAIReferencePricingSnapshotMatchesOfficialTable(t *testing.T) {
	snapshot := defaultCodexOpenAIReferencePricingSnapshot()
	require.NoError(t, validateCodexOpenAIReferencePricingSnapshot(snapshot))

	assert.Equal(t, 5.0, snapshot.Prices["gpt-5.6-sol"].InputPerMillion)
	assert.Equal(t, 30.0, snapshot.Prices["gpt-5.6-sol"].OutputPerMillion)
	assert.Equal(t, 128000, snapshot.Prices["gpt-5.6-sol"].MaxCompletionTokens)
	assert.Equal(t, 2.5, snapshot.Prices["gpt-5.6-terra"].InputPerMillion)
	assert.Equal(t, 15.0, snapshot.Prices["gpt-5.6-terra"].OutputPerMillion)
	assert.Equal(t, 1.0, snapshot.Prices["gpt-5.6-luna"].InputPerMillion)
	assert.Equal(t, 6.0, snapshot.Prices["gpt-5.6-luna"].OutputPerMillion)
	assert.Equal(t, 5.0, snapshot.Prices["gpt-5.5"].InputPerMillion)
	assert.Equal(t, 30.0, snapshot.Prices["gpt-5.5"].OutputPerMillion)
	_, sparkHasInventedPrice := snapshot.Prices["gpt-5.3-codex-spark"]
	assert.False(t, sparkHasInventedPrice)
}

func TestCodexOpenAIReferencePricingPatchesOnlyPromotedModels(t *testing.T) {
	originalDB := model.DB
	t.Cleanup(func() { model.DB = originalDB })

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	require.NoError(t, db.AutoMigrate(&model.CodexCatalogCandidate{}, &model.CodexCatalogSnapshot{}))
	require.NoError(t, db.Create(&[]model.CodexCatalogCandidate{
		{ChannelID: codexCatalogDefaultChannelID, ModelName: "gpt-5.6-sol", Promoted: true},
		{ChannelID: codexCatalogDefaultChannelID, ModelName: "gpt-5.6-luna", Promoted: true},
		{ChannelID: codexCatalogDefaultChannelID, ModelName: "gpt-5.5", Promoted: true},
		{ChannelID: codexCatalogDefaultChannelID, ModelName: "gpt-5.6-terra", Promoted: false},
	}).Error)

	snapshot, err := parseCodexOpenAIStandardPricing(
		[]byte(codexOpenAIReferencePricingFixture()),
		time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC),
		`"etag"`,
		"Sat, 18 Jul 2026 21:40:48 GMT",
	)
	require.NoError(t, err)

	patches := codexOpenAIReferencePricingPatches(snapshot)
	require.Equal(t, 5.0, patches["gpt-5.6-sol"].Input)
	require.Equal(t, 30.0, patches["gpt-5.6-sol"].Output)
	require.True(t, patches["gpt-5.6-sol"].SyncCacheRead)
	require.True(t, patches["gpt-5.6-sol"].SyncCacheWrite)
	require.NotNil(t, patches["gpt-5.6-sol"].CacheReadRatio)
	require.InDelta(t, 0.1, *patches["gpt-5.6-sol"].CacheReadRatio, 1e-12)
	require.NotNil(t, patches["gpt-5.6-sol"].CacheWriteRatio)
	require.InDelta(t, 1.25, *patches["gpt-5.6-sol"].CacheWriteRatio, 1e-12)
	require.Equal(t, 1.0, patches["gpt-5.6-luna"].Input)
	require.Equal(t, 6.0, patches["gpt-5.6-luna"].Output)
	require.True(t, patches["gpt-5.5"].SyncCacheWrite)
	require.Nil(t, patches["gpt-5.5"].CacheWriteRatio)
	require.NotContains(t, patches, "gpt-5.6-terra")
}

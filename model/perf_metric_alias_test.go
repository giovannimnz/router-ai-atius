package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestPerfMetricsCanonicalizeLegacyRerankerAlias(t *testing.T) {
	previousDB := DB
	previousType := common.MainDatabaseType()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	initCol()
	t.Cleanup(func() {
		DB = previousDB
		common.SetMainDatabaseType(previousType)
		initCol()
	})
	require.NoError(t, db.AutoMigrate(&PerfMetric{}))

	require.NoError(t, db.Create(&PerfMetric{ModelName: constant.LegacyAtiusLocalRerankerModel, Group: "default", BucketTs: 3_600, RequestCount: 2, SuccessCount: 1, TotalLatencyMs: 20}).Error)
	require.NoError(t, db.Create(&PerfMetric{ModelName: constant.AtiusLocalRerankerModel, Group: "default", BucketTs: 3_600, RequestCount: 3, SuccessCount: 3, TotalLatencyMs: 30}).Error)
	require.NoError(t, db.Create(&PerfMetric{ModelName: constant.LegacyAtiusLocalRerankerModel, Group: "default", BucketTs: 7_200, RequestCount: 4, SuccessCount: 4, TotalLatencyMs: 40}).Error)

	summary, err := GetPerfMetricsSummaryAll(0, 10_000, nil)
	require.NoError(t, err)
	require.Len(t, summary, 1)
	assert.Equal(t, constant.AtiusLocalRerankerModel, summary[0].ModelName)
	assert.Equal(t, int64(9), summary[0].RequestCount)
	assert.Equal(t, int64(8), summary[0].SuccessCount)
	assert.Equal(t, int64(90), summary[0].TotalLatencyMs)

	buckets, err := GetPerfMetricsSummaryBucketsAll(0, 10_000, nil)
	require.NoError(t, err)
	require.Len(t, buckets, 2)
	assert.Equal(t, constant.AtiusLocalRerankerModel, buckets[0].ModelName)
	assert.Equal(t, int64(5), buckets[0].RequestCount)

	rows, err := GetPerfMetrics(constant.AtiusLocalRerankerModel, "default", 0, 10_000)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, int64(5), rows[0].RequestCount)
	assert.Equal(t, int64(4), rows[0].SuccessCount)
	for _, row := range rows {
		assert.Equal(t, constant.AtiusLocalRerankerModel, row.ModelName)
	}
}

package perfmetrics

import (
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlushAllNowPersistsCurrentBucket(t *testing.T) {
	previousUpsert := upsertPerfMetric
	t.Cleanup(func() {
		upsertPerfMetric = previousUpsert
		hotBuckets = sync.Map{}
	})

	hotBuckets = sync.Map{}

	key := bucketKey{
		model:    "embedding-gte-v1",
		group:    "default",
		bucketTs: bucketStart(time.Now().Unix()),
	}
	bucket := &atomicBucket{}
	bucket.add(Sample{
		Model:     key.model,
		Group:     key.group,
		Success:   true,
		LatencyMs: 25,
	})
	hotBuckets.Store(key, bucket)

	var persisted []*model.PerfMetric
	upsertPerfMetric = func(metric *model.PerfMetric) error {
		copied := *metric
		persisted = append(persisted, &copied)
		return nil
	}

	FlushAllNow()

	require.Len(t, persisted, 1)
	assert.Equal(t, key.model, persisted[0].ModelName)
	assert.Equal(t, key.group, persisted[0].Group)
	assert.Equal(t, key.bucketTs, persisted[0].BucketTs)
	assert.EqualValues(t, 1, persisted[0].RequestCount)
	assert.EqualValues(t, 1, persisted[0].SuccessCount)

	_, exists := hotBuckets.Load(key)
	assert.False(t, exists)
}

func TestFlushAllNowRetainsCountersWhenPersistenceFails(t *testing.T) {
	previousUpsert := upsertPerfMetric
	t.Cleanup(func() {
		upsertPerfMetric = previousUpsert
		hotBuckets = sync.Map{}
	})

	hotBuckets = sync.Map{}
	key := bucketKey{
		model:    "embedding-gte-v1",
		group:    "default",
		bucketTs: bucketStart(time.Now().Unix()),
	}
	bucket := &atomicBucket{}
	bucket.add(Sample{Model: key.model, Group: key.group, Success: true})
	hotBuckets.Store(key, bucket)
	upsertPerfMetric = func(*model.PerfMetric) error {
		return assert.AnError
	}

	FlushAllNow()

	stored, exists := hotBuckets.Load(key)
	require.True(t, exists)
	assert.EqualValues(t, 1, stored.(*atomicBucket).snapshot().requestCount)
	assert.EqualValues(t, 1, stored.(*atomicBucket).snapshot().successCount)
}

package cache_test

import (
	"sync"
	"testing"
	"time"

	"wbtest/pkg/cache"
	mock_logger "wbtest/pkg/logger/mock"
	mock_metric "wbtest/pkg/metric/mock"

	"github.com/golang/mock/gomock"
)

type cacheOperation struct {
	op    string
	key   int
	value string
	ttl   time.Duration
}

type getTestInput struct {
	capacity int
	ops      []cacheOperation
}

type getTestExpected struct {
	results map[int]struct {
		value string
		ok    bool
	}
	finalGet struct {
		key   int
		value string
		ok    bool
	}
	len int
}

func TestLRUCache_GetPut(t *testing.T) {
	key1, key2, key3 := 1, 2, 3
	value1, value2, value3 := "one", "two", "three"
	noValue := struct {
		value string
		ok    bool
	}{"", false}

	testCases := []struct {
		desc     string
		input    getTestInput
		expected getTestExpected
	}{
		{
			desc: "BasicGetPut",
			input: getTestInput{
				capacity: 2,
				ops: []cacheOperation{
					{"put", key1, value1, 0},
					{"put", key2, value2, 0},
					{"get", key1, "", 0},
				},
			},
			expected: getTestExpected{
				results: map[int]struct {
					value string
					ok    bool
				}{
					key1: {value1, true},
					key2: {value2, true},
				},
				finalGet: struct {
					key   int
					value string
					ok    bool
				}{key1, value1, true},
				len: 2,
			},
		},
		{
			desc: "LRUEviction",
			input: getTestInput{
				capacity: 2,
				ops: []cacheOperation{
					{"put", key1, value1, 0},
					{"put", key2, value2, 0},
					{"get", key1, "", 0},
					{"put", key3, value3, 0},
					{"get", key3, "", 0},
				},
			},
			expected: getTestExpected{
				results: map[int]struct {
					value string
					ok    bool
				}{
					key1: {value1, true},
					key2: noValue,
					key3: {value3, true},
				},
				finalGet: struct {
					key   int
					value string
					ok    bool
				}{key3, value3, true},
				len: 2,
			},
		},
		{
			desc: "UpdateExistingKey",
			input: getTestInput{
				capacity: 2,
				ops: []cacheOperation{
					{"put", key1, value1, 0},
					{"put", key2, value2, 0},
					{"put", key1, value3, 0},
					{"get", key1, "", 0},
				},
			},
			expected: getTestExpected{
				results: map[int]struct {
					value string
					ok    bool
				}{
					key1: {value3, true},
				},
				finalGet: struct {
					key   int
					value string
					ok    bool
				}{key1, value3, true},
				len: 2,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockLogger := mock_logger.NewMockLogger(ctrl)
			mockMetrics := mock_metric.NewMockCache(ctrl)

			mockMetrics.EXPECT().Hit(gomock.Any()).AnyTimes()
			mockMetrics.EXPECT().Miss(gomock.Any()).AnyTimes()
			mockMetrics.EXPECT().Eviction(gomock.Any(), gomock.Any()).AnyTimes()

			c, _ := cache.NewLRUCache[int, string](tc.input.capacity, mockLogger, mockMetrics)
			var lastGet struct {
				key   int
				value string
				ok    bool
			}

			for _, op := range tc.input.ops {
				switch op.op {
				case "put":
					c.Put(op.key, op.value, op.ttl)
				case "get":
					lastGet.key = op.key
					lastGet.value, lastGet.ok = c.Get(op.key)
				}
			}

			for key, want := range tc.expected.results {
				got, ok := c.Get(key)
				if got != want.value || ok != want.ok {
					t.Errorf("Get(%d) = %q, %v; want %q, %v",
						key, got, ok, want.value, want.ok)
				}
			}

			if lastGet.value != tc.expected.finalGet.value ||
				lastGet.ok != tc.expected.finalGet.ok {
				t.Errorf("Final Get(%d) = %q, %v; want %q, %v",
					lastGet.key, lastGet.value, lastGet.ok,
					tc.expected.finalGet.value, tc.expected.finalGet.ok)
			}

			if c.Len() != tc.expected.len {
				t.Errorf("Len() = %d; want %d", c.Len(), tc.expected.len)
			}
		})
	}
}

type ttlTestInput struct {
	capacity int
	key      int
	value    string
	ttl      time.Duration
	sleep    time.Duration
}

type ttlTestExpected struct {
	value string
	ok    bool
}

func TestLRUCache_TTL(t *testing.T) {
	key := 1
	value := "one"

	testCases := []struct {
		desc     string
		input    ttlTestInput
		expected ttlTestExpected
	}{
		{
			desc: "TTLNotExpired",
			input: ttlTestInput{
				capacity: 1,
				key:      key,
				value:    value,
				ttl:      200 * time.Millisecond,
				sleep:    100 * time.Millisecond,
			},
			expected: ttlTestExpected{
				value: value,
				ok:    true,
			},
		},
		{
			desc: "TTLExpired",
			input: ttlTestInput{
				capacity: 1,
				key:      key,
				value:    value,
				ttl:      100 * time.Millisecond,
				sleep:    200 * time.Millisecond,
			},
			expected: ttlTestExpected{
				value: "",
				ok:    false,
			},
		},
		{
			desc: "NoTTL",
			input: ttlTestInput{
				capacity: 1,
				key:      key,
				value:    value,
				ttl:      0,
				sleep:    300 * time.Millisecond,
			},
			expected: ttlTestExpected{
				value: value,
				ok:    true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockLogger := mock_logger.NewMockLogger(ctrl)
			mockMetrics := mock_metric.NewMockCache(ctrl)

			mockMetrics.EXPECT().Hit(gomock.Any()).AnyTimes()
			mockMetrics.EXPECT().Miss(gomock.Any()).AnyTimes()
			mockMetrics.EXPECT().Eviction(gomock.Any(), gomock.Any()).AnyTimes()

			c, _ := cache.NewLRUCache[int, string](tc.input.capacity, mockLogger, mockMetrics)
			c.Put(tc.input.key, tc.input.value, tc.input.ttl)
			time.Sleep(tc.input.sleep)

			got, ok := c.Get(tc.input.key)
			if got != tc.expected.value || ok != tc.expected.ok {
				t.Errorf("Get() = %q, %v; want %q, %v",
					got, ok, tc.expected.value, tc.expected.ok)
			}
		})
	}
}

type hasTestInput struct {
	capacity int
	key      int
	value    string
	ttl      time.Duration
	sleep    time.Duration
}

func TestLRUCache_Has(t *testing.T) {
	key := 1
	value := "one"

	testCases := []struct {
		desc     string
		input    hasTestInput
		expected bool
	}{
		{
			desc: "ValidKey",
			input: hasTestInput{
				capacity: 1,
				key:      key,
				value:    value,
				ttl:      0,
				sleep:    0,
			},
			expected: true,
		},
		{
			desc: "ExpiredKey",
			input: hasTestInput{
				capacity: 1,
				key:      key,
				value:    value,
				ttl:      100 * time.Millisecond,
				sleep:    200 * time.Millisecond,
			},
			expected: false,
		},
		{
			desc: "NonExistentKey",
			input: hasTestInput{
				capacity: 1,
				key:      99,
				value:    "",
				ttl:      0,
				sleep:    0,
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockLogger := mock_logger.NewMockLogger(ctrl)
			mockMetrics := mock_metric.NewMockCache(ctrl)

			mockMetrics.EXPECT().Hit(gomock.Any()).AnyTimes()
			mockMetrics.EXPECT().Miss(gomock.Any()).AnyTimes()
			mockMetrics.EXPECT().Eviction(gomock.Any(), gomock.Any()).AnyTimes()

			c, _ := cache.NewLRUCache[int, string](tc.input.capacity, mockLogger, mockMetrics)
			if tc.input.value != "" {
				c.Put(tc.input.key, tc.input.value, tc.input.ttl)
			}

			time.Sleep(tc.input.sleep)
			if got := c.Has(tc.input.key); got != tc.expected {
				t.Errorf("Has() = %v; want %v", got, tc.expected)
			}
		})
	}
}

type onEvictedTestInput struct {
	capacity  int
	ops       []int
	wantPurge bool
}

type onEvictedTestExpected struct {
	evictedKeys []int
	finalLen    int
}

func TestLRUCache_OnEvicted(t *testing.T) {
	testCases := []struct {
		desc     string
		input    onEvictedTestInput
		expected onEvictedTestExpected
	}{
		{
			desc: "SingleEviction",
			input: onEvictedTestInput{
				capacity: 2,
				ops:      []int{1, 2, 3},
			},
			expected: onEvictedTestExpected{
				evictedKeys: []int{1},
				finalLen:    2,
			},
		},
		{
			desc: "MultipleEvictions",
			input: onEvictedTestInput{
				capacity: 1,
				ops:      []int{1, 2, 3},
			},
			expected: onEvictedTestExpected{
				evictedKeys: []int{1, 2},
				finalLen:    1,
			},
		},
		{
			desc: "PurgeEvictions",
			input: onEvictedTestInput{
				capacity:  2,
				ops:       []int{1, 2},
				wantPurge: true,
			},
			expected: onEvictedTestExpected{
				evictedKeys: []int{1, 2},
				finalLen:    0,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockLogger := mock_logger.NewMockLogger(ctrl)
			mockMetrics := mock_metric.NewMockCache(ctrl)

			mockMetrics.EXPECT().Hit(gomock.Any()).AnyTimes()
			mockMetrics.EXPECT().Miss(gomock.Any()).AnyTimes()
			mockMetrics.EXPECT().Eviction(gomock.Any(), gomock.Any()).AnyTimes()

			var (
				mu          sync.Mutex
				evictedKeys []int
			)

			c, _ := cache.NewLRUCache[int, string](tc.input.capacity, mockLogger, mockMetrics)
			c.SetOnEvicted(func(key int, _ string) {
				mu.Lock()
				defer mu.Unlock()
				evictedKeys = append(evictedKeys, key)
			})

			for _, key := range tc.input.ops {
				c.Put(key, "value", 0)
			}

			if tc.input.wantPurge {
				c.Purge()
			}

			mu.Lock()
			defer mu.Unlock()

			if len(evictedKeys) != len(tc.expected.evictedKeys) {
				t.Fatalf("Evicted count = %d; want %d",
					len(evictedKeys), len(tc.expected.evictedKeys))
			}

			for i, key := range evictedKeys {
				if key != tc.expected.evictedKeys[i] {
					t.Errorf("evictedKeys[%d] = %d; want %d",
						i, key, tc.expected.evictedKeys[i])
				}
			}

			if c.Len() != tc.expected.finalLen {
				t.Errorf("Final Len() = %d; want %d", c.Len(), tc.expected.finalLen)
			}
		})
	}
}

func TestLRUCache_Capacity(t *testing.T) {
	testCases := []struct {
		desc     string
		capacity int
	}{
		{"Capacity1", 1},
		{"Capacity10", 10},
		{"Capacity100", 100},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockLogger := mock_logger.NewMockLogger(ctrl)
			mockMetrics := mock_metric.NewMockCache(ctrl)

			mockMetrics.EXPECT().Hit(gomock.Any()).AnyTimes()
			mockMetrics.EXPECT().Miss(gomock.Any()).AnyTimes()
			mockMetrics.EXPECT().Eviction(gomock.Any(), gomock.Any()).AnyTimes()

			c, _ := cache.NewLRUCache[int, string](tc.capacity, mockLogger, mockMetrics)
			if got := c.Capacity(); got != tc.capacity {
				t.Errorf("Capacity() = %d; want %d", got, tc.capacity)
			}
		})
	}
}

func TestLRUCache_NewLRUCache(t *testing.T) {
	testCases := []struct {
		desc      string
		capacity  int
		wantError bool
	}{
		{"NegativeCapacity", -1, true},
		{"ZeroCapacity", 0, true},
		{"PositiveCapacity", 10, false},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockLogger := mock_logger.NewMockLogger(ctrl)
			mockMetrics := mock_metric.NewMockCache(ctrl)

			_, err := cache.NewLRUCache[int, string](tc.capacity, mockLogger, mockMetrics)
			if (err != nil) != tc.wantError {
				t.Errorf("NewLRUCache[int, string]() error = %v, wantError %v", err, tc.wantError)
			}
		})
	}
}

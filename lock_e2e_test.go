package goRedisLock

import (
	"context"
	"github.com/go-redis/redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestClient_TryLock_e2e(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})
	c := NewClient(rdb)
	c.Wait()
	testCases := []struct {
		name       string
		key        string
		expiration time.Duration
		wantErr    error
		wantLock   *Lock

		// 准备数据
		before func()
		// 校验redis数据且清理数据
		after func()
	}{
		{
			name:       "locked",
			key:        "locked-key",
			expiration: time.Minute,
			before: func() {

			},
			after: func() {
				res, err := rdb.Del(context.Background(), "locked-key").Result()
				require.NoError(t, err)
				require.Equal(t, int64(1), res)
			},
			wantLock: &Lock{
				key: "locked-key",
			},
		},
		{
			name:       "failed to lock",
			key:        "failed-key",
			expiration: time.Minute,
			before: func() {
				res, err := rdb.Set(context.Background(), "failed-key", "1234", time.Minute).Result()
				require.NoError(t, err)
				require.Equal(t, "OK", res)
			},
			after: func() {
				res, err := rdb.Get(context.Background(), "failed-key").Result()
				require.NoError(t, err)
				require.Equal(t, "1234", res)
			},
			wantErr: ErrFailedToPreemptLock,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before()
			l, err := c.TryLock(context.Background(), tc.key, tc.expiration)
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			tc.after()
			assert.NotNil(t, l.client)
			assert.Equal(t, tc.wantLock.key, l.key)
			assert.NotEmpty(t, l.value)
		})
	}
}

func TestLock_Unlock_e2e(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})
	c := NewClient(rdb)
	c.Wait()

	testCases := []struct {
		name    string
		lock    *Lock
		key     string
		wantErr error
		before  func()
		after   func()
	}{
		{
			name: "unlocked",
			key:  "unlocked-key",
			lock: func() *Lock {
				l, err := c.TryLock(context.Background(), "unlocked-key", time.Minute)
				require.NoError(t, err)
				return l
			}(),
			before: func() {

			},
			after: func() {
				res, err := rdb.Exists(context.Background(), "unlocked-key").Result()
				require.NoError(t, err)
				require.Equal(t, 1, res)
			},
		},
		{
			name:    "unlocked no exist",
			key:     "not-hold-key",
			lock:    newLock(c.client, "not-hold-key", "123"),
			wantErr: ErrLockNotHold,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.lock.Unlock(context.Background(), tc.key)
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
		})
	}
}

func (c *Client) Wait() {
	for c.client.Ping(context.Background()).Err() != nil {
	}
}

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

func (c *Client) Wait() {
	for c.client.Ping(context.Background()).Err() != nil {
	}
}

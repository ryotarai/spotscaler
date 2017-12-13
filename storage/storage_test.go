package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type dummyRedis struct {
	reply       interface{}
	lastCommand string
	lastArgs    []interface{}
}

func (r *dummyRedis) Close() error {
	return nil
}
func (r *dummyRedis) Err() error {
	return nil
}
func (r *dummyRedis) Do(commandName string, args ...interface{}) (reply interface{}, err error) {
	r.lastCommand = commandName
	r.lastArgs = args
	return r.reply, nil
}
func (r *dummyRedis) Send(commandName string, args ...interface{}) error {
	return nil
}
func (r *dummyRedis) Flush() error {
	return nil
}
func (r *dummyRedis) Receive() (reply interface{}, err error) {
	return nil, nil
}

func TestAddSchedule(t *testing.T) {
	redis := &dummyRedis{}
	s := &RedisStorage{
		redis:  redis,
		Prefix: "test/",
	}

	s.AddSchedule(&Schedule{
		ID:       "a",
		StartAt:  time.Unix(0, 0),
		EndAt:    time.Unix(600, 0),
		Capacity: 10.0,
	})

	assert.Equal(t, redis.lastCommand, "HSet")
	assert.Equal(t, redis.lastArgs, []interface{}{"test/schedules", "a", "{\"ID\":\"a\",\"StartAt\":\"1970-01-01T09:00:00+09:00\",\"EndAt\":\"1970-01-01T09:10:00+09:00\",\"Capacity\":10}"})
}

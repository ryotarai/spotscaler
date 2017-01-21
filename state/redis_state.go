package state

type RedisState struct {
	redisHost string
}

func NewRedisState(redisHost string) *RedisState {
	return &RedisState{
		redisHost: redisHost,
	}
}

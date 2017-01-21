package state

type RedisState struct {
	redisHost string
	status    *Status
}

func NewRedisState(redisHost string) *RedisState {
	return &RedisState{
		redisHost: redisHost,
	}
}

func (s *RedisState) UpdateStatus(status *Status) error {
	// TODO thread safe
	s.status = status
	return nil
}

package state

type State struct {
	redisHost string
}

func NewState(redisHost string) *State {
	return &State{
		redisHost: redisHost,
	}
}

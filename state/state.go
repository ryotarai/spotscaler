package state

type Status struct {
	CurrentOndemandCapacity int
	CurrentSpotCapacity     int
}

type State interface {
	UpdateStatus(*Status) error
}

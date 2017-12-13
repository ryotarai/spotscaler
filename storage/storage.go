package storage

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/google/uuid"
)

type Storage interface {
	AddSchedule(sch *Schedule) error
	ListSchedules() ([]*Schedule, error)
	RemoveSchedule(id string) error
	ActiveSchedule() (*Schedule, error)
}

type RedisStorage struct {
	redis  redis.Conn
	Prefix string
}

func NewRedisStorage(url, prefix string) (*RedisStorage, error) {
	r, err := redis.DialURL(url)
	if err != nil {
		return nil, err
	}

	return &RedisStorage{
		redis:  r,
		Prefix: prefix,
	}, nil
}

type Schedule struct {
	ID       string    `json:"ID"`
	StartAt  time.Time `json:"StartAt"`
	EndAt    time.Time `json:"EndAt"`
	Capacity float64   `json:"Capacity"`
}

func NewScheduleID() string {
	return uuid.New().String()
}

func (s *RedisStorage) AddSchedule(sch *Schedule) error {
	j, err := json.Marshal(sch)
	if err != nil {
		return err
	}

	_, err = s.redis.Do("HSet", s.key("schedules"), sch.ID, string(j))
	return err
}

func (s *RedisStorage) RemoveSchedule(id string) error {
	_, err := s.redis.Do("HDel", s.key("schedules"), id)
	return err
}

func (s *RedisStorage) ListSchedules() ([]*Schedule, error) {
	reply, err := redis.StringMap(s.redis.Do("HGetAll", s.key("schedules")))
	if err != nil {
		return nil, err
	}

	schedules := []*Schedule{}
	for _, j := range reply {
		sch := &Schedule{}
		json.Unmarshal([]byte(j), sch)
		schedules = append(schedules, sch)
	}

	return schedules, nil
}

func (s *RedisStorage) ActiveSchedule() (*Schedule, error) {
	schs, err := s.ListSchedules()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var active *Schedule
	for _, sch := range schs {
		if now.After(sch.StartAt) && now.Before(sch.EndAt) && (active == nil || sch.StartAt.After(active.StartAt)) {
			active = sch
		}
	}

	return active, nil
}

func (s *RedisStorage) key(k string) string {
	return fmt.Sprintf("%s%s", s.Prefix, k)
}

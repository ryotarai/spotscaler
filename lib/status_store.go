package autoscaler

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"gopkg.in/redis.v4"
)

type StatusStoreIface interface {
	StoreCooldownEndsAt(t time.Time) error
	FetchCooldownEndsAt() (time.Time, error)
	ListSchedules() ([]*Schedule, error)
	AddSchedules(sch *Schedule) error
	RemoveSchedule(key string) error
	UpdateTimer(key string, t time.Time) error
	DeleteTimer(key string) error
	GetExpiredTimers() ([]string, error)
	StoreMetric(values map[string]float64) error
	GetMetric() (map[string]float64, error)
}

// StatusStore stores status data in Redis
type StatusStore struct {
	redisClient *redis.Client
	KeyPrefix   string
}

func NewStatusStore(redisHost string, autoscalerID string) *StatusStore {
	client := redis.NewClient(&redis.Options{
		Addr: redisHost,
		DB:   0,
	})

	return &StatusStore{
		redisClient: client,
		KeyPrefix:   fmt.Sprintf("%s/", autoscalerID),
	}
}

func (s *StatusStore) key(k string) string {
	return s.KeyPrefix + k
}

func (s *StatusStore) storeTime(k string, t time.Time) error {
	_, err := s.redisClient.Set(s.key(k), fmt.Sprint(t.Unix()), 0).Result()
	return err
}

func (s *StatusStore) fetchTime(k string) (time.Time, error) {
	str, err := s.redisClient.Get(s.key(k)).Result()
	if err != nil {
		return time.Time{}, err
	}
	i, err := strconv.Atoi(str)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(int64(i), 0), err
}

func (s *StatusStore) StoreCooldownEndsAt(t time.Time) error {
	return s.storeTime("cooldownEndsAt", t)
}

func (s *StatusStore) FetchCooldownEndsAt() (time.Time, error) {
	t, err := s.fetchTime("cooldownEndsAt")
	if err == redis.Nil {
		// not found
		return time.Time{}, nil
	}
	return t, err
}

func (s *StatusStore) ListSchedules() ([]*Schedule, error) {
	result, err := s.redisClient.HGetAll(s.key("schedules")).Result()
	if err != nil {
		return nil, err
	}

	schedules := []*Schedule{}
	for _, j := range result {
		var s Schedule
		json.Unmarshal([]byte(j), &s)
		schedules = append(schedules, &s)
	}

	return schedules, nil
}

func (s *StatusStore) AddSchedules(sch *Schedule) error {
	j, err := json.Marshal(sch)
	if err != nil {
		return err
	}

	_, err = s.redisClient.HSet(s.key("schedules"), sch.Key, string(j)).Result()
	if err != nil {
		return err
	}

	return nil
}

func (s *StatusStore) RemoveSchedule(key string) error {
	_, err := s.redisClient.HDel(s.key("schedules"), key).Result()
	if err != nil {
		return err
	}

	return nil
}

func (s *StatusStore) UpdateTimer(key string, t time.Time) error {
	_, err := s.redisClient.HSet(s.key("timers"), key, fmt.Sprintf("%d", t.Unix())).Result()
	if err != nil {
		return err
	}
	return nil
}

func (s *StatusStore) DeleteTimer(key string) error {
	_, err := s.redisClient.HDel(s.key("timers"), key).Result()
	if err != nil {
		return err
	}
	return nil
}

func (s *StatusStore) GetExpiredTimers() ([]string, error) {
	m, err := s.redisClient.HGetAll(s.key("timers")).Result()
	if err != nil {
		return nil, err
	}
	keys := []string{}
	for k, v := range m {
		i, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}
		if time.Unix(int64(i), 0).Before(time.Now()) {
			keys = append(keys, k)
		}
	}

	return keys, nil
}

func (s *StatusStore) StoreMetric(values map[string]float64) error {
	svalues := map[string]string{}
	for k, v := range values {
		svalues[k] = fmt.Sprintf("%f", v)
	}

	_, err := s.redisClient.HMSet(s.key("metric"), svalues).Result()
	if err != nil {
		return err
	}
	return nil
}

func (s *StatusStore) GetMetric() (map[string]float64, error) {
	strM, err := s.redisClient.HGetAll(s.key("metric")).Result()
	if err != nil {
		return nil, err
	}

	floatM := map[string]float64{}
	for k, v := range strM {
		floatM[k], err = strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, err
		}
	}

	return floatM, nil
}

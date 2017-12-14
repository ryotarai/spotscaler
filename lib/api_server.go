package autoscaler

import (
	"fmt"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
)

type APIServer struct {
	status  StatusStoreIface
	metrics map[string]float64
}

func NewAPIServer(status StatusStoreIface) *APIServer {
	return &APIServer{
		status:  status,
		metrics: map[string]float64{},
	}
}

func (s *APIServer) UpdateMetrics(metrics map[string]float64) {
	s.metrics = metrics
}

func (s *APIServer) Run(addr string) {
	r := gin.Default()
	r.GET("/metrics", s.getMetricsHandler)
	r.GET("/schedules", s.getSchedulesHandler)
	r.POST("/schedules", s.postSchedulesHandler)
	r.DELETE("/schedules", s.deleteSchedulesHandler)
	go func() {
		r.Run(addr)
	}()
}

func (s *APIServer) getMetricsHandler(c *gin.Context) {
	lines := []string{}
	for k, v := range s.metrics {
		lines = append(lines, fmt.Sprintf("spotscaler_%s{} %f", k, v))
	}
	body := fmt.Sprintf("%s\n", strings.Join(lines, "\n"))
	c.String(200, body)
}

func (s *APIServer) getSchedulesHandler(c *gin.Context) {
	schedules, err := s.status.ListSchedules()
	if err != nil {
		log.Printf("[ERROR] %v", err)
	}

	c.JSON(200, schedules)
}

func (s *APIServer) postSchedulesHandler(c *gin.Context) {
	sch := NewSchedule()
	if err := c.BindJSON(sch); err == nil {
		s.status.AddSchedules(sch)
		c.JSON(201, sch)
	} else {
		c.String(400, "%s", err)
	}
}

func (s *APIServer) deleteSchedulesHandler(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		c.String(400, "key is not specified")
		return
	}

	if err := s.status.RemoveSchedule(key); err != nil {
		c.String(400, "%s", err)
		return
	}

	c.JSON(200, gin.H{
		"key":     key,
		"deleted": true,
	})
}

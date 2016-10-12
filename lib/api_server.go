package autoscaler

import (
	"github.com/gin-gonic/gin"
)

type APIServer struct {
	status *StatusStore
}

func NewAPIServer(status *StatusStore) *APIServer {
	return &APIServer{
		status: status,
	}
}

func (s *APIServer) Run(addr string) {
	r := gin.Default()
	r.GET("/schedules", s.getSchedulesHandler)
	r.POST("/schedules", s.postSchedulesHandler)
	r.DELETE("/schedules", s.deleteSchedulesHandler)
	r.Run(addr)
}

func (s *APIServer) getSchedulesHandler(c *gin.Context) {
	schedules, _ := s.status.ListSchedules()
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

	if err := s.status.RemoveSchedules(key); err != nil {
		c.String(400, "%s", err)
		return
	}

	c.JSON(200, gin.H{
		"key":     key,
		"deleted": true,
	})
}

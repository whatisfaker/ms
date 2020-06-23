package ms

import (
	"encoding/json"
	"net/http"
)

type StatusContainer struct {
	s *Server
}

func NewStatusContainer(s *Server) *StatusContainer {
	return &StatusContainer{
		s: s,
	}
}

func (c *StatusContainer) StatusHandler(w http.ResponseWriter, r *http.Request) {
	status := c.s.Status()
	b, err := json.Marshal(status)
	if err != nil {
		_, _ = w.Write([]byte("Error marshal"))
		return
	}
	_, _ = w.Write(b)
}

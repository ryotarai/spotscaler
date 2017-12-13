package httpapi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ryotarai/spotscaler/storage"
)

type Handler struct {
	metrics map[string]float64
	Storage storage.Storage
}

func NewHandler(storage storage.Storage) *Handler {
	return &Handler{
		metrics: map[string]float64{},
		Storage: storage,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/metrics" {
		h.handleMetrics(w, r)
	} else if r.URL.Path == "/schedules" && r.Method == "GET" {
		h.handleSchedulesGet(w, r)
	} else if r.URL.Path == "/schedules" && r.Method == "POST" {
		h.handleSchedulesPost(w, r)
	} else if r.URL.Path == "/schedules" && r.Method == "DELETE" {
		h.handleSchedulesDelete(w, r)
	} else {
		w.WriteHeader(404)
		fmt.Fprintln(w, "404 not found")
	}
}

func (h *Handler) UpdateMetric(k string, v float64) {
	h.metrics[k] = v
}

func (h *Handler) handleMetrics(w http.ResponseWriter, r *http.Request) {
	for k, v := range h.metrics {
		fmt.Fprintf(w, "spotscaler_%s{} %f\n", k, v)
	}
}

func (h *Handler) handleSchedulesPost(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		h.error(w, err)
		return
	}
	r.Body.Close()

	sch := &storage.Schedule{}
	err = json.Unmarshal(b, sch)
	if err != nil {
		h.error(w, err)
		return
	}
	sch.ID = storage.NewScheduleID()

	err = h.Storage.AddSchedule(sch)
	if err != nil {
		h.error(w, err)
		return
	}

	b, err = json.Marshal(sch)
	if err != nil {
		h.error(w, err)
		return
	}

	w.WriteHeader(201)
	fmt.Fprintf(w, "%s\n", string(b))
}

func (h *Handler) handleSchedulesGet(w http.ResponseWriter, r *http.Request) {
	schs, err := h.Storage.ListSchedules()
	if err != nil {
		h.error(w, err)
		return
	}

	b, err := json.Marshal(schs)
	if err != nil {
		h.error(w, err)
		return
	}

	w.WriteHeader(200)
	fmt.Fprintf(w, "%s\n", string(b))
}

func (h *Handler) handleSchedulesDelete(w http.ResponseWriter, r *http.Request) {
	k := r.URL.Query().Get("key")
	if k == "" {
		w.WriteHeader(400)
		fmt.Fprintln(w, "'key' query param is missing")
	}

	err := h.Storage.RemoveSchedule(k)
	if err != nil {
		h.error(w, err)
		return
	}

	w.WriteHeader(200)
	fmt.Fprintln(w, "Deleted")
}

func (h *Handler) error(w http.ResponseWriter, err error) {
	w.WriteHeader(500)
	fmt.Fprintln(w, err)
}

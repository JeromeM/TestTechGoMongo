package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/JeromeM/TestTechGoMongo/client"
	"github.com/JeromeM/TestTechGoMongo/schemas"
	"github.com/go-chi/chi/v5"
	"github.com/kataras/golog"
)

type Server struct {
	tasks client.MongoClient
}

func NewServer(tasks client.MongoClient) *Server {
	return &Server{tasks: tasks}
}

func (s *Server) Serve(port string) {

	// Add a signal channel to properly exit
	// Not very useful right now, there's nothing to "kill" :)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	kill := func() {
		fmt.Print("\n")
		golog.Info("Gracefully stopping server")
		os.Exit(0)
	}

	go func() {
		<-sigChan
		kill()
	}()

	r := chi.NewRouter()

	r.Get("/tasks", s.handleGetTasks)
	r.Patch("/tasks/{taskId}", s.handlePatchTask)

	server := http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%s", port),
		Handler: r,
	}

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			golog.Errorf("Error while starting server: %v\n", err)
			kill()
		}
	}()

	<-sigChan

	if err := server.Shutdown(context.TODO()); err != nil {
		fmt.Printf("Erreur lors de l'arrêt du serveur: %v\n", err)
	}

	kill()

}

// Get all tasks
func (s *Server) handleGetTasks(w http.ResponseWriter, r *http.Request) {
	params := getParams(r.URL.Query())
	tasks, err := s.tasks.GetTasks(params)
	if err != nil {
		internalServerError(w, err)
		return
	}

	pagination := client.GetPagination(params)

	res := schemas.Tasks{
		Tasks:      tasks,
		Pagination: pagination,
	}
	s.sendJSON(w, res)
}

// Patch task
func (s *Server) handlePatchTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskId")

	var req schemas.TaskUpdate

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.missingDataError(w, fmt.Errorf("empty payload"))
		return
	}

	// Verify input data
	if err := assertTaskUpdatePayload(&req); err != nil {
		s.missingDataError(w, err)
		return
	}

	if err := s.tasks.UpdateOne(taskID, req); err != nil {
		s.missingDataError(w, err)
		return
	}

	s.sendJSON(w, req)
}

////////////////////////////

func getParams(values url.Values) *schemas.TasksSearchParams {
	const (
		STATUS = "status"
		LIMIT  = "limit"
		PAGE   = "page"
	)
	var (
		limit uint64
		page  uint64
	)
	limit, err := strconv.ParseUint(values.Get(LIMIT), 10, 8)
	if err != nil {
		golog.Warn("can't parse limit parameter")
	}
	page, err = strconv.ParseUint(values.Get(PAGE), 10, 8)
	if err != nil {
		golog.Warn("can't parse page parameter")
	}
	if page == 0 {
		page = 1
	}
	return &schemas.TasksSearchParams{
		Status: strings.ToLower(values.Get(STATUS)),
		Limit:  uint16(limit),
		Page:   uint16(page),
	}
}

func assertTaskUpdatePayload(source *schemas.TaskUpdate) error {
	if len(source.AssigneeId) == 0 {
		return fmt.Errorf("empty assignee ID")
	}
	return nil
}

func (s *Server) missingDataError(w http.ResponseWriter, err error) {
	golog.Errorf("error %v: %s", http.StatusUnprocessableEntity, err)

	w.WriteHeader(http.StatusUnprocessableEntity)
	msg := schemas.BodyError{
		Error: fmt.Sprintf("%s", err),
	}
	s.sendJSON(w, msg)
}

func internalServerError(w http.ResponseWriter, err error) {
	golog.Errorf("error %v: %s", http.StatusInternalServerError, err)

	w.WriteHeader(http.StatusInternalServerError)
	msg := http.StatusText(http.StatusInternalServerError)
	_, _ = fmt.Fprint(w, msg)
}

func (s *Server) sendJSON(w http.ResponseWriter, body interface{}) {
	w.Header().Add("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(body)
	if err != nil {
		internalServerError(w, err)
	}
}

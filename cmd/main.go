package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Todo struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
}

var (
	todos  = make(map[int]Todo)
	mu     sync.RWMutex
	nextID = 1
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/todos", todosHandler)
	mux.HandleFunc("/todos/", todoByIDHandler)

	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func todosHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case http.MethodGet:
		mu.RLock()
		list := make([]Todo, 0, len(todos))
		for _, t := range todos {
			list = append(list, t)
		}
		mu.RUnlock()

		writeJSON(w, http.StatusOK, list)

	case http.MethodPost:
		var input struct {
			Title string `json:"title"`
		}

		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		if strings.TrimSpace(input.Title) == "" {
			http.Error(w, "title required", http.StatusBadRequest)
			return
		}

		mu.Lock()
		todo := Todo{
			ID:        nextID,
			Title:     input.Title,
			Completed: false,
			CreatedAt: time.Now(),
		}
		todos[nextID] = todo
		nextID++
		mu.Unlock()

		writeJSON(w, http.StatusCreated, todo)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func todoByIDHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/todos/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	todo, exists := todos[id]
	if !exists {
		http.NotFound(w, r)
		return
	}

	switch r.Method {

	case http.MethodGet:
		writeJSON(w, http.StatusOK, todo)

	case http.MethodPut:
		var input struct {
			Title     *string `json:"title"`
			Completed *bool   `json:"completed"`
		}

		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		if input.Title != nil {
			todo.Title = *input.Title
		}
		if input.Completed != nil {
			todo.Completed = *input.Completed
		}

		todos[id] = todo
		writeJSON(w, http.StatusOK, todo)

	case http.MethodDelete:
		delete(todos, id)
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

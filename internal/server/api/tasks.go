package api

import "net/http"

func (s *Server) handleTaskByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id, _ := splitSubpath(r.URL.Path, "/api/tasks/")
	if id == "" {
		writeBadRequest(w, "missing task id")
		return
	}

	task, err := s.store.GetTask(id)
	if err != nil {
		writeNotFound(w, "task not found")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

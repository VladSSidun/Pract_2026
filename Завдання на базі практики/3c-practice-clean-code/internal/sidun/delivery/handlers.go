package delivery

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"very-bad-project/internal/sidun/usecases"
)

// Handler — HTTP-обробники (Фаза 2: форматування відповіді)
type Handler struct {
	authUC     *usecases.AuthUseCase
	scheduleUC *usecases.ScheduleUseCase
	loggedIn   int // поточний авторизований користувач (спрощена сесія)
}

func NewHandler(authUC *usecases.AuthUseCase, scheduleUC *usecases.ScheduleUseCase) *Handler {
	return &Handler{authUC: authUC, scheduleUC: scheduleUC}
}

// RegisterRoutes реєструє маршрути
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/login", h.handleLogin).Methods("POST")
	r.HandleFunc("/schedule", h.handleListSchedule).Methods("GET")
	r.HandleFunc("/schedule/add", h.handleAddSchedule).Methods("POST")
	r.HandleFunc("/schedule/{id}", h.handleGetSchedule).Methods("GET")
	r.HandleFunc("/schedule/{id}", h.handleUpdateSchedule).Methods("PUT")
}

// --- Фаза 2: HTTP-обробка та форматування ---

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Невалідний JSON")
		return
	}

	result, err := h.authUC.Login(req.Username, req.Password)
	if err != nil {
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	h.loggedIn = result.UserID

	respondJSON(w, http.StatusOK, LoginResponse{
		Message:  "Login successful",
		UserID:   result.UserID,
		UserName: result.UserName,
		Token:    result.Token,
	})
}

func (h *Handler) handleListSchedule(w http.ResponseWriter, r *http.Request) {
	results, err := h.scheduleUC.ListSchedule()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := make([]ScheduleResponse, 0, len(results))
	for _, res := range results {
		response = append(response, toScheduleResponse(res))
	}

	respondJSON(w, http.StatusOK, response)
}

func (h *Handler) handleGetSchedule(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid schedule ID")
		return
	}

	result, err := h.scheduleUC.GetScheduleByID(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Schedule not found")
		return
	}

	respondJSON(w, http.StatusOK, toScheduleResponse(result))
}

func (h *Handler) handleAddSchedule(w http.ResponseWriter, r *http.Request) {
	if h.loggedIn == 0 {
		respondError(w, http.StatusUnauthorized, "Not logged in")
		return
	}

	var dto AddScheduleDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		respondError(w, http.StatusBadRequest, "Невалідний JSON")
		return
	}

	if dto.Subject == "" || dto.Teacher == "" || dto.Day == "" || dto.TimeSlot == "" || dto.Room == "" {
		respondError(w, http.StatusBadRequest, "All fields required")
		return
	}

	if dto.MaxStudents == 0 {
		dto.MaxStudents = 30
	}

	newID, err := h.scheduleUC.AddSchedule(usecases.AddScheduleRequest{
		Subject:     dto.Subject,
		Teacher:     dto.Teacher,
		GroupName:   dto.Group,
		Day:         dto.Day,
		TimeSlot:    dto.TimeSlot,
		Room:        dto.Room,
		MaxStudents: dto.MaxStudents,
	})
	if err != nil {
		if err.Error() == "виявлено конфлікт у розкладі" {
			respondError(w, http.StatusConflict, "Schedule conflict detected")
			return
		}
		respondError(w, http.StatusBadRequest, "Invalid schedule data")
		return
	}

	respondJSON(w, http.StatusCreated, SuccessResponse{
		Message: "Class added",
		ID:      newID,
	})
}

func (h *Handler) handleUpdateSchedule(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid schedule ID")
		return
	}

	var dto UpdateScheduleDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		respondError(w, http.StatusBadRequest, "Невалідний JSON")
		return
	}

	err = h.scheduleUC.UpdateSchedule(usecases.UpdateScheduleRequest{
		ID:      id,
		Room:    dto.Room,
		Teacher: dto.Teacher,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, SuccessResponse{
		Message: "Schedule updated",
		ID:      id,
	})
}

// --- Допоміжні функції форматування ---

func toScheduleResponse(res usecases.ScheduleResult) ScheduleResponse {
	return ScheduleResponse{
		ID:       res.ID,
		Subject:  res.Subject,
		Teacher:  res.Teacher,
		Group:    res.Group,
		Day:      res.Day,
		Time:     res.TimeSlot,
		Room:     res.Room,
		Max:      res.MaxStudents,
		Enrolled: res.Enrolled,
	}
}

func parseIDFromPath(r *http.Request) (int, error) {
	vars := mux.Vars(r)
	return strconv.Atoi(vars["id"])
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

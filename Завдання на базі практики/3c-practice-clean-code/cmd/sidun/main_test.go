package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"very-bad-project/internal/sidun/delivery"
	"very-bad-project/internal/sidun/entities"
	"very-bad-project/internal/sidun/events"
	"very-bad-project/internal/sidun/infrastructure"
	"very-bad-project/internal/sidun/usecases"
)

// --- Тести доменних сутностей (Value Objects, Aggregate Root) ---

func TestNewDay_ValidDays(t *testing.T) {
	validDays := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"}
	for _, d := range validDays {
		day, err := entities.NewDay(d)
		if err != nil {
			t.Errorf("NewDay(%q) повернув помилку: %v", d, err)
		}
		if day.String() != d {
			t.Errorf("Очікувалось %q, отримано %q", d, day.String())
		}
	}
}

func TestNewDay_InvalidDay(t *testing.T) {
	_, err := entities.NewDay("Saturday")
	if err == nil {
		t.Error("NewDay('Saturday') мав повернути помилку")
	}
}

func TestNewCapacity_Valid(t *testing.T) {
	cap, err := entities.NewCapacity(30)
	if err != nil {
		t.Errorf("NewCapacity(30) повернув помилку: %v", err)
	}
	if cap.Value() != 30 {
		t.Errorf("Очікувалось 30, отримано %d", cap.Value())
	}
}

func TestNewCapacity_Invalid(t *testing.T) {
	_, err := entities.NewCapacity(0)
	if err == nil {
		t.Error("NewCapacity(0) мав повернути помилку")
	}

	_, err = entities.NewCapacity(201)
	if err == nil {
		t.Error("NewCapacity(201) мав повернути помилку")
	}
}

func TestCapacity_IsAvailable(t *testing.T) {
	cap, _ := entities.NewCapacity(30)
	if !cap.IsAvailable(29) {
		t.Error("Очікувалось true для 29 з 30")
	}
	if cap.IsAvailable(30) {
		t.Error("Очікувалось false для 30 з 30")
	}
}

func TestNewSchedule_Valid(t *testing.T) {
	s, err := entities.NewSchedule(entities.ScheduleParams{
		Subject:     "Math",
		Teacher:     "Dr. Smith",
		GroupName:   "CS-101",
		Day:         "Monday",
		TimeSlot:    "9:00",
		Room:        "A101",
		MaxStudents: 30,
	})
	if err != nil {
		t.Fatalf("NewSchedule повернув помилку: %v", err)
	}
	if s.Subject() != "Math" {
		t.Errorf("Очікувалось 'Math', отримано %q", s.Subject())
	}
	if !s.IsAvailable() {
		t.Error("Нове заняття має бути доступним")
	}
}

func TestNewSchedule_InvalidDay(t *testing.T) {
	_, err := entities.NewSchedule(entities.ScheduleParams{
		Subject: "Math", Teacher: "Dr. Smith", Day: "Sunday",
		TimeSlot: "9:00", Room: "A101", MaxStudents: 30,
	})
	if err == nil {
		t.Error("Мав повернути помилку для недопустимого дня")
	}
}

func TestNewSchedule_EmptySubject(t *testing.T) {
	_, err := entities.NewSchedule(entities.ScheduleParams{
		Subject: "", Teacher: "Dr. Smith", Day: "Monday",
		TimeSlot: "9:00", Room: "A101", MaxStudents: 30,
	})
	if err == nil {
		t.Error("Мав повернути помилку для порожнього предмету")
	}
}

func TestSchedule_OverrideRoom(t *testing.T) {
	s, _ := entities.NewSchedule(entities.ScheduleParams{
		Subject: "Math", Teacher: "Dr. Smith", GroupName: "CS-101",
		Day: "Monday", TimeSlot: "9:00", Room: "A101", MaxStudents: 30,
	})
	if s.EffectiveRoom().String() != "A101" {
		t.Errorf("Без перевизначення аудиторія має бути A101")
	}

	s.OverrideRoom("B202")
	if s.EffectiveRoom().String() != "B202" {
		t.Errorf("Після перевизначення аудиторія має бути B202")
	}
}

func TestSchedule_HasConflictWith(t *testing.T) {
	s1 := entities.RestoreSchedule(entities.ScheduleParams{
		ID: 1, Subject: "Math", Teacher: "Dr. Smith",
		Day: "Monday", TimeSlot: "9:00", Room: "A101", MaxStudents: 30,
	})
	s2 := entities.RestoreSchedule(entities.ScheduleParams{
		ID: 2, Subject: "Physics", Teacher: "Dr. Jones",
		Day: "Monday", TimeSlot: "9:00", Room: "A101", MaxStudents: 30,
	})
	if !s1.HasConflictWith(s2) {
		t.Error("Має бути конфлікт — одна аудиторія, один час")
	}

	s3 := entities.RestoreSchedule(entities.ScheduleParams{
		ID: 3, Subject: "Physics", Teacher: "Dr. Smith",
		Day: "Monday", TimeSlot: "9:00", Room: "B202", MaxStudents: 30,
	})
	if !s1.HasConflictWith(s3) {
		t.Error("Має бути конфлікт — один викладач, один час")
	}

	s4 := entities.RestoreSchedule(entities.ScheduleParams{
		ID: 4, Subject: "Physics", Teacher: "Dr. Jones",
		Day: "Tuesday", TimeSlot: "9:00", Room: "A101", MaxStudents: 30,
	})
	if s1.HasConflictWith(s4) {
		t.Error("Не має бути конфлікту — різні дні")
	}
}

func TestWorkload(t *testing.T) {
	w := entities.CalculateWorkload(entities.WorkloadParams{
		Lectures: 10, Labs: 5, Practices: 3, IsHead: true, YearsActive: 12,
	})
	if w.Hours() != 40 {
		t.Errorf("Очікувалось 40 (обмеження), отримано %d", w.Hours())
	}
	if !w.IsOverloaded() {
		t.Error("Очікувалось IsOverloaded == true")
	}

	w2 := entities.CalculateWorkload(entities.WorkloadParams{
		Lectures: 5, Labs: 2, Practices: 1, IsHead: false, YearsActive: 3,
	})
	expected := 5*2 + 2 + 1
	if w2.Hours() != expected {
		t.Errorf("Очікувалось %d, отримано %d", expected, w2.Hours())
	}
}

// --- Тести Use Cases з InMemory репозиторієм ---

func TestAuthUseCase_Login(t *testing.T) {
	userRepo := infrastructure.NewInMemoryUserRepository()
	userRepo.AddUser("admin", "123", "admin")
	authUC := usecases.NewAuthUseCase(userRepo)

	result, err := authUC.Login("admin", "123")
	if err != nil {
		t.Fatalf("Login повернув помилку: %v", err)
	}
	if result.UserName != "admin" {
		t.Errorf("Очікувалось 'admin', отримано %q", result.UserName)
	}
	if result.Token == "" {
		t.Error("Token не має бути порожнім")
	}
}

func TestAuthUseCase_LoginInvalid(t *testing.T) {
	userRepo := infrastructure.NewInMemoryUserRepository()
	authUC := usecases.NewAuthUseCase(userRepo)

	_, err := authUC.Login("wrong", "wrong")
	if err == nil {
		t.Error("Login з невірними даними мав повернути помилку")
	}
}

func TestScheduleUseCase_AddAndList(t *testing.T) {
	repo := infrastructure.NewInMemoryScheduleRepository()
	bus := events.NewEventBus()
	uc := usecases.NewScheduleUseCase(repo, bus)

	newID, err := uc.AddSchedule(usecases.AddScheduleRequest{
		Subject: "Math", Teacher: "Dr. Smith", GroupName: "CS-101",
		Day: "Monday", TimeSlot: "9:00", Room: "A101", MaxStudents: 30,
	})
	if err != nil {
		t.Fatalf("AddSchedule повернув помилку: %v", err)
	}
	if newID <= 0 {
		t.Error("Новий ID має бути більше 0")
	}

	results, err := uc.ListSchedule()
	if err != nil {
		t.Fatalf("ListSchedule повернув помилку: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Очікувалось 1 запис, отримано %d", len(results))
	}
}

func TestScheduleUseCase_ConflictDetection(t *testing.T) {
	repo := infrastructure.NewInMemoryScheduleRepository()
	bus := events.NewEventBus()
	uc := usecases.NewScheduleUseCase(repo, bus)

	uc.AddSchedule(usecases.AddScheduleRequest{
		Subject: "Math", Teacher: "Dr. Smith", GroupName: "CS-101",
		Day: "Monday", TimeSlot: "9:00", Room: "A101", MaxStudents: 30,
	})

	_, err := uc.AddSchedule(usecases.AddScheduleRequest{
		Subject: "Physics", Teacher: "Dr. Jones", GroupName: "CS-102",
		Day: "Monday", TimeSlot: "9:00", Room: "A101", MaxStudents: 25,
	})
	if err == nil {
		t.Error("Має бути конфлікт аудиторії")
	}
}

// --- Інтеграційні тести з HTTP ---

func setupIntegrationTest(t *testing.T) (*mux.Router, func()) {
	os.Remove("./test_schedule.db")
	db, err := sqlx.Connect("sqlite3", "./test_schedule.db")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)

	db.Exec(`CREATE TABLE IF NOT EXISTS users(id INTEGER PRIMARY KEY, name TEXT, password TEXT, role TEXT)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS groups(id INTEGER PRIMARY KEY, name TEXT)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS schedule(id INTEGER PRIMARY KEY, subject TEXT, teacher TEXT, group_name TEXT, day TEXT, time_slot TEXT, room TEXT, max_students INTEGER, enrolled INTEGER DEFAULT 0)`)
	db.Exec("INSERT INTO users(name, password, role) VALUES('admin', '123', 'admin')")
	db.Exec("INSERT INTO schedule(subject, teacher, group_name, day, time_slot, room, max_students, enrolled) VALUES('Math', 'Dr. Smith', 'CS-101', 'Monday', '9:00', 'A101', 30, 0)")

	scheduleRepo := infrastructure.NewSQLiteScheduleRepository(db)
	userRepo := infrastructure.NewSQLiteUserRepository(db)
	eventBus := events.NewEventBus()
	authUC := usecases.NewAuthUseCase(userRepo)
	scheduleUC := usecases.NewScheduleUseCase(scheduleRepo, eventBus)

	handler := delivery.NewHandler(authUC, scheduleUC)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	cleanup := func() {
		db.Close()
		os.Remove("./test_schedule.db")
	}
	return router, cleanup
}

func TestIntegration_LoginSuccess(t *testing.T) {
	router, cleanup := setupIntegrationTest(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "123"})
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Errorf("Очікувалось 200, отримано %d, body: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "Login successful") {
		t.Errorf("Відповідь має містити 'Login successful'")
	}
	if !strings.Contains(rr.Body.String(), "token") {
		t.Errorf("Відповідь має містити 'token'")
	}
}

func TestIntegration_LoginInvalid(t *testing.T) {
	router, cleanup := setupIntegrationTest(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]string{"username": "wrong", "password": "wrong"})
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != 401 {
		t.Errorf("Очікувалось 401, отримано %d", rr.Code)
	}
}

func TestIntegration_LoginMissingCredentials(t *testing.T) {
	router, cleanup := setupIntegrationTest(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]string{})
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != 401 {
		t.Errorf("Очікувалось 401, отримано %d", rr.Code)
	}
}

func TestIntegration_ListSchedule(t *testing.T) {
	router, cleanup := setupIntegrationTest(t)
	defer cleanup()

	req, _ := http.NewRequest("GET", "/schedule", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Errorf("Очікувалось 200, отримано %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Math") {
		t.Errorf("Відповідь має містити 'Math'")
	}
}

func TestIntegration_GetScheduleByID(t *testing.T) {
	router, cleanup := setupIntegrationTest(t)
	defer cleanup()

	req, _ := http.NewRequest("GET", "/schedule/1", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Errorf("Очікувалось 200, отримано %d, body: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "Dr. Smith") {
		t.Errorf("Відповідь має містити 'Dr. Smith'")
	}
}

func TestIntegration_GetScheduleNotFound(t *testing.T) {
	router, cleanup := setupIntegrationTest(t)
	defer cleanup()

	req, _ := http.NewRequest("GET", "/schedule/999", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != 404 {
		t.Errorf("Очікувалось 404, отримано %d", rr.Code)
	}
}

func TestIntegration_AddSchedule(t *testing.T) {
	router, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Спочатку логін
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "123"})
	loginReq, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(loginBody))
	loginRR := httptest.NewRecorder()
	router.ServeHTTP(loginRR, loginReq)

	body, _ := json.Marshal(map[string]interface{}{
		"subject": "Physics", "teacher": "Dr. Jones", "group": "CS-102",
		"day": "Tuesday", "time_slot": "11:00", "room": "B202", "max_students": 25,
	})
	req, _ := http.NewRequest("POST", "/schedule/add", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != 201 {
		t.Errorf("Очікувалось 201, отримано %d, body: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "Class added") {
		t.Errorf("Відповідь має містити 'Class added'")
	}
}

func TestIntegration_AddScheduleConflict(t *testing.T) {
	router, cleanup := setupIntegrationTest(t)
	defer cleanup()

	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "123"})
	loginReq, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(loginBody))
	loginRR := httptest.NewRecorder()
	router.ServeHTTP(loginRR, loginReq)

	body, _ := json.Marshal(map[string]interface{}{
		"subject": "Physics", "teacher": "Dr. Jones", "group": "CS-102",
		"day": "Monday", "time_slot": "9:00", "room": "A101", "max_students": 25,
	})
	req, _ := http.NewRequest("POST", "/schedule/add", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != 409 {
		t.Errorf("Очікувалось 409 (конфлікт), отримано %d, body: %s", rr.Code, rr.Body.String())
	}
}

func TestIntegration_AddScheduleUnauthorized(t *testing.T) {
	router, cleanup := setupIntegrationTest(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{
		"subject": "Physics", "teacher": "Dr. Jones",
		"day": "Wednesday", "time_slot": "14:00", "room": "C303", "max_students": 20,
	})
	req, _ := http.NewRequest("POST", "/schedule/add", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != 401 {
		t.Errorf("Очікувалось 401, отримано %d", rr.Code)
	}
}

func TestIntegration_UpdateSchedule(t *testing.T) {
	router, cleanup := setupIntegrationTest(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{
		"teacher": "Dr. Brown",
	})
	req, _ := http.NewRequest("PUT", "/schedule/1", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Errorf("Очікувалось 200, отримано %d, body: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "Schedule updated") {
		t.Errorf("Відповідь має містити 'Schedule updated'")
	}
}

func TestIntegration_InvalidScheduleID(t *testing.T) {
	router, cleanup := setupIntegrationTest(t)
	defer cleanup()

	req, _ := http.NewRequest("GET", "/schedule/abc", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != 400 {
		t.Errorf("Очікувалось 400, отримано %d", rr.Code)
	}
}

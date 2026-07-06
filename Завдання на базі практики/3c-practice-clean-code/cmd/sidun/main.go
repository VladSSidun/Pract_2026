package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"

	"very-bad-project/internal/sidun/delivery"
	"very-bad-project/internal/sidun/events"
	"very-bad-project/internal/sidun/infrastructure"
	"very-bad-project/internal/sidun/usecases"
)

// main — Composition Root: збирає всі залежності та запускає сервер.
// Це єдине місце, де відбувається Dependency Injection.
func main() {
	godotenv.Load()

	dbPath := os.Getenv("DB_PATH")
	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "8080"
	}

	// 1. Ініціалізація інфраструктури (зовнішній шар)
	db, err := sqlx.Connect("sqlite3", dbPath)
	if err != nil {
		log.Fatal("DB connection failed:", err)
	}
	db.SetMaxOpenConns(1)
	initDB(db)
	seedDatabase(db)

	// 2. Створення репозиторіїв (адаптери)
	scheduleRepo := infrastructure.NewSQLiteScheduleRepository(db)
	userRepo := infrastructure.NewSQLiteUserRepository(db)

	// 3. Event Bus для зв'язку між Bounded Contexts
	eventBus := events.NewEventBus()
	reportingHandler := events.NewReportingEventHandler()
	eventBus.Subscribe(reportingHandler)

	// 4. Use Cases (бізнес-логіка)
	authUC := usecases.NewAuthUseCase(userRepo)
	scheduleUC := usecases.NewScheduleUseCase(scheduleRepo, eventBus)

	// 5. Delivery (HTTP-шар)
	handler := delivery.NewHandler(authUC, scheduleUC)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	// 6. Запуск сервера
	fmt.Println("Schedule server running on :" + serverPort)
	log.Fatal(http.ListenAndServe(":"+serverPort, router))
}

// seedDatabase заповнює базу початковими даними, якщо вона порожня
func seedDatabase(db *sqlx.DB) {
	var cnt int
	db.Get(&cnt, "SELECT COUNT(*) FROM users")
	if cnt == 0 {
		db.Exec("INSERT INTO users(name, password, role) VALUES('admin', '123', 'admin')")
		db.Exec("INSERT INTO users(name, password, role) VALUES('teacher1', 'pass', 'teacher')")
		db.Exec("INSERT INTO groups(name) VALUES('CS-101')")
		db.Exec("INSERT INTO groups(name) VALUES('CS-102')")
		db.Exec("INSERT INTO schedule(subject, teacher, group_name, day, time_slot, room, max_students, enrolled) VALUES('Math', 'Dr. Smith', 'CS-101', 'Monday', '9:00', 'A101', 30, 0)")
		db.Exec("INSERT INTO schedule(subject, teacher, group_name, day, time_slot, room, max_students, enrolled) VALUES('Physics', 'Dr. Jones', 'CS-102', 'Tuesday', '11:00', 'B202', 25, 0)")
	}
}



// initDB створює таблиці, якщо їх ще немає
func initDB(db *sqlx.DB) {
	db.MustExec(`CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		password TEXT NOT NULL,
		role TEXT DEFAULT 'student'
	)`)
	db.MustExec(`CREATE TABLE IF NOT EXISTS groups (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL
	)`)
	db.MustExec(`CREATE TABLE IF NOT EXISTS schedule (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		subject TEXT NOT NULL,
		teacher TEXT NOT NULL,
		group_name TEXT NOT NULL,
		day TEXT NOT NULL,
		time_slot TEXT NOT NULL,
		room TEXT NOT NULL,
		max_students INTEGER DEFAULT 30,
		enrolled INTEGER DEFAULT 0
	)`)
}

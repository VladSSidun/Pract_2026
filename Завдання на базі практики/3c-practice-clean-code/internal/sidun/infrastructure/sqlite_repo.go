package infrastructure

import (
	"errors"

	"github.com/jmoiron/sqlx"
	"very-bad-project/internal/sidun/entities"
)

// SQLiteScheduleRepository — адаптер для SQLite
type SQLiteScheduleRepository struct {
	db *sqlx.DB
}

func NewSQLiteScheduleRepository(db *sqlx.DB) *SQLiteScheduleRepository {
	return &SQLiteScheduleRepository{db: db}
}

func (r *SQLiteScheduleRepository) GetAll() ([]*entities.Schedule, error) {
	rows, err := r.db.Queryx("SELECT id, subject, teacher, group_name, day, time_slot, room, max_students, enrolled FROM schedule ORDER BY day, time_slot")
	if err != nil {
		return nil, errors.New("помилка читання розкладу з бази даних")
	}
	defer rows.Close()

	var schedules []*entities.Schedule
	for rows.Next() {
		s, err := r.scanSchedule(rows)
		if err != nil {
			continue
		}
		schedules = append(schedules, s)
	}
	return schedules, nil
}

func (r *SQLiteScheduleRepository) GetByID(id int) (*entities.Schedule, error) {
	rows, err := r.db.Queryx(
		"SELECT id, subject, teacher, group_name, day, time_slot, room, max_students, enrolled FROM schedule WHERE id = ?", id,
	)
	if err != nil {
		return nil, errors.New("запис розкладу не знайдено")
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, errors.New("запис розкладу не знайдено")
	}
	return r.scanSchedule(rows)
}

func (r *SQLiteScheduleRepository) Save(schedule *entities.Schedule) (int, error) {
	res, err := r.db.Exec(
		"INSERT INTO schedule(subject, teacher, group_name, day, time_slot, room, max_students, enrolled) VALUES(?, ?, ?, ?, ?, ?, ?, 0)",
		schedule.Subject(),
		schedule.Teacher(),
		schedule.GroupName(),
		schedule.Day().String(),
		schedule.TimeSlot().String(),
		schedule.Room().String(),
		schedule.MaxStudents().Value(),
	)
	if err != nil {
		return 0, errors.New("помилка збереження запису в базу даних")
	}
	newID, _ := res.LastInsertId()
	return int(newID), nil
}

func (r *SQLiteScheduleRepository) Update(schedule *entities.Schedule) error {
	_, err := r.db.Exec(
		"UPDATE schedule SET teacher = ? WHERE id = ?",
		schedule.Teacher(),
		schedule.ID(),
	)
	if err != nil {
		return errors.New("помилка оновлення запису в базі даних")
	}
	return nil
}

func (r *SQLiteScheduleRepository) FindConflicts(day, timeSlot, room, teacher string, excludeID int) (int, error) {
	var roomConflicts int
	err := r.db.Get(&roomConflicts,
		"SELECT COUNT(*) FROM schedule WHERE day = ? AND time_slot = ? AND room = ? AND id != ?",
		day, timeSlot, room, excludeID,
	)
	if err != nil {
		return 0, errors.New("помилка перевірки конфліктів")
	}
	if roomConflicts > 0 {
		return roomConflicts, nil
	}

	var teacherConflicts int
	err = r.db.Get(&teacherConflicts,
		"SELECT COUNT(*) FROM schedule WHERE day = ? AND time_slot = ? AND teacher = ? AND id != ?",
		day, timeSlot, teacher, excludeID,
	)
	if err != nil {
		return 0, errors.New("помилка перевірки конфліктів")
	}
	return teacherConflicts, nil
}

// scanSchedule зчитує один рядок з результату запиту
func (r *SQLiteScheduleRepository) scanSchedule(rows *sqlx.Rows) (*entities.Schedule, error) {
	m := make(map[string]interface{})
	if err := rows.MapScan(m); err != nil {
		return nil, err
	}

	return entities.RestoreSchedule(entities.ScheduleParams{
		ID:          int(m["id"].(int64)),
		Subject:     m["subject"].(string),
		Teacher:     m["teacher"].(string),
		GroupName:   m["group_name"].(string),
		Day:         m["day"].(string),
		TimeSlot:    m["time_slot"].(string),
		Room:        m["room"].(string),
		MaxStudents: int(m["max_students"].(int64)),
		Enrolled:    int(m["enrolled"].(int64)),
	}), nil
}

// SQLiteUserRepository — адаптер для роботи з користувачами
type SQLiteUserRepository struct {
	db *sqlx.DB
}

func NewSQLiteUserRepository(db *sqlx.DB) *SQLiteUserRepository {
	return &SQLiteUserRepository{db: db}
}

func (r *SQLiteUserRepository) FindByCredentials(username, password string) (*entities.User, error) {
	rows, err := r.db.Queryx(
		"SELECT id, name, role FROM users WHERE name = ? AND password = ?",
		username, password,
	)
	if err != nil {
		return nil, errors.New("невірні облікові дані")
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, errors.New("невірні облікові дані")
	}

	m := make(map[string]interface{})
	if err := rows.MapScan(m); err != nil {
		return nil, errors.New("помилка зчитування даних користувача")
	}

	role := ""
	if r, ok := m["role"].(string); ok {
		role = r
	}

	return entities.NewUser(
		int(m["id"].(int64)),
		m["name"].(string),
		password,
		role,
	)
}

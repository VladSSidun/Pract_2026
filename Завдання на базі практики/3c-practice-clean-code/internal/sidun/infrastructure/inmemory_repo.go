package infrastructure

import (
	"errors"
	"very-bad-project/internal/sidun/entities"
)

// InMemoryScheduleRepository — репозиторій у пам'яті для тестування
type InMemoryScheduleRepository struct {
	schedules map[int]*entities.Schedule
	nextID    int
}

func NewInMemoryScheduleRepository() *InMemoryScheduleRepository {
	return &InMemoryScheduleRepository{
		schedules: make(map[int]*entities.Schedule),
		nextID:    1,
	}
}

func (r *InMemoryScheduleRepository) GetAll() ([]*entities.Schedule, error) {
	result := make([]*entities.Schedule, 0, len(r.schedules))
	for _, s := range r.schedules {
		result = append(result, s)
	}
	return result, nil
}

func (r *InMemoryScheduleRepository) GetByID(id int) (*entities.Schedule, error) {
	s, ok := r.schedules[id]
	if !ok {
		return nil, errors.New("запис не знайдено")
	}
	return s, nil
}

func (r *InMemoryScheduleRepository) Save(schedule *entities.Schedule) (int, error) {
	id := r.nextID
	r.nextID++
	restored := entities.RestoreSchedule(entities.ScheduleParams{
		ID:          id,
		Subject:     schedule.Subject(),
		Teacher:     schedule.Teacher(),
		GroupName:   schedule.GroupName(),
		Day:         schedule.Day().String(),
		TimeSlot:    schedule.TimeSlot().String(),
		Room:        schedule.Room().String(),
		MaxStudents: schedule.MaxStudents().Value(),
		Enrolled:    schedule.Enrolled(),
	})
	r.schedules[id] = restored
	return id, nil
}

func (r *InMemoryScheduleRepository) Update(schedule *entities.Schedule) error {
	if _, ok := r.schedules[schedule.ID()]; !ok {
		return errors.New("запис не знайдено")
	}
	r.schedules[schedule.ID()] = schedule
	return nil
}

func (r *InMemoryScheduleRepository) FindConflicts(day, timeSlot, room, teacher string, excludeID int) (int, error) {
	count := 0
	for _, s := range r.schedules {
		if s.ID() == excludeID {
			continue
		}
		if s.Day().String() == day && s.TimeSlot().String() == timeSlot {
			if s.Room().String() == room || s.Teacher() == teacher {
				count++
			}
		}
	}
	return count, nil
}

// InMemoryUserRepository — репозиторій користувачів у пам'яті
type InMemoryUserRepository struct {
	users []*entities.User
}

func NewInMemoryUserRepository() *InMemoryUserRepository {
	return &InMemoryUserRepository{}
}

func (r *InMemoryUserRepository) AddUser(name, password, role string) {
	id := len(r.users) + 1
	u, _ := entities.NewUser(id, name, password, role)
	r.users = append(r.users, u)
}

func (r *InMemoryUserRepository) FindByCredentials(username, password string) (*entities.User, error) {
	for _, u := range r.users {
		if u.Name() == username && u.VerifyPassword(password) {
			return u, nil
		}
	}
	return nil, errors.New("невірні облікові дані")
}

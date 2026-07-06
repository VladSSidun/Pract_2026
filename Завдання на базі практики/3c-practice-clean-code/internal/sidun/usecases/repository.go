package usecases

import "very-bad-project/internal/sidun/entities"

// ScheduleRepository — контракт для збереження розкладу.
// Інтерфейс визначений у шарі use cases (Ownership Inversion).
type ScheduleRepository interface {
	GetAll() ([]*entities.Schedule, error)
	GetByID(id int) (*entities.Schedule, error)
	Save(schedule *entities.Schedule) (int, error)
	Update(schedule *entities.Schedule) error
	FindConflicts(day, timeSlot, room, teacher string, excludeID int) (int, error)
}

// UserRepository — контракт для роботи з користувачами
type UserRepository interface {
	FindByCredentials(username, password string) (*entities.User, error)
}

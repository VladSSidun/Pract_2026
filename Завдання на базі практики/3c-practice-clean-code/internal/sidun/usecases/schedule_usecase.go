package usecases

import (
	"errors"
	"very-bad-project/internal/sidun/entities"
)

// ScheduleUseCase — прикладна логіка роботи з розкладом
type ScheduleUseCase struct {
	scheduleRepo ScheduleRepository
	eventBus     EventPublisher
}

// EventPublisher — інтерфейс для публікації подій
type EventPublisher interface {
	Publish(event interface{})
}

func NewScheduleUseCase(repo ScheduleRepository, bus EventPublisher) *ScheduleUseCase {
	return &ScheduleUseCase{scheduleRepo: repo, eventBus: bus}
}

// --- Проміжні структури (Split Phase) ---

// ScheduleResult — результат обчислення для одного запису розкладу
type ScheduleResult struct {
	ID          int
	Subject     string
	Teacher     string
	Group       string
	Day         string
	TimeSlot    string
	Room        string
	MaxStudents int
	Enrolled    int
	Available   bool
}

// AddScheduleRequest — запит на додавання заняття
type AddScheduleRequest struct {
	Subject     string
	Teacher     string
	GroupName   string
	Day         string
	TimeSlot    string
	Room        string
	MaxStudents int
}

// UpdateScheduleRequest — запит на оновлення заняття
type UpdateScheduleRequest struct {
	ID      int
	Room    string
	Teacher string
}

// --- Фаза 1: бізнес-логіка (без HTTP/JSON) ---

// ListSchedule повертає весь розклад
func (uc *ScheduleUseCase) ListSchedule() ([]ScheduleResult, error) {
	schedules, err := uc.scheduleRepo.GetAll()
	if err != nil {
		return nil, errors.New("помилка отримання розкладу")
	}

	results := make([]ScheduleResult, 0, len(schedules))
	for _, s := range schedules {
		results = append(results, uc.toResult(s))
	}
	return results, nil
}

// GetScheduleByID повертає один запис розкладу
func (uc *ScheduleUseCase) GetScheduleByID(id int) (ScheduleResult, error) {
	schedule, err := uc.scheduleRepo.GetByID(id)
	if err != nil {
		return ScheduleResult{}, errors.New("запис розкладу не знайдено")
	}
	return uc.toResult(schedule), nil
}

// AddSchedule додає новий запис до розкладу
func (uc *ScheduleUseCase) AddSchedule(req AddScheduleRequest) (int, error) {
	schedule, err := entities.NewSchedule(entities.ScheduleParams{
		Subject:     req.Subject,
		Teacher:     req.Teacher,
		GroupName:   req.GroupName,
		Day:         req.Day,
		TimeSlot:    req.TimeSlot,
		Room:        req.Room,
		MaxStudents: req.MaxStudents,
	})
	if err != nil {
		return 0, err
	}

	conflicts, err := uc.scheduleRepo.FindConflicts(
		req.Day, req.TimeSlot, req.Room, req.Teacher, 0,
	)
	if err != nil {
		return 0, errors.New("помилка перевірки конфліктів")
	}
	if conflicts > 0 {
		return 0, errors.New("виявлено конфлікт у розкладі")
	}

	newID, err := uc.scheduleRepo.Save(schedule)
	if err != nil {
		return 0, errors.New("помилка збереження запису")
	}

	if uc.eventBus != nil {
		uc.eventBus.Publish(ScheduleCreatedEvent{
			ScheduleID: newID,
			Subject:    req.Subject,
			Teacher:    req.Teacher,
			Day:        req.Day,
			TimeSlot:   req.TimeSlot,
		})
	}

	return newID, nil
}

// UpdateSchedule оновлює існуючий запис розкладу
func (uc *ScheduleUseCase) UpdateSchedule(req UpdateScheduleRequest) error {
	schedule, err := uc.scheduleRepo.GetByID(req.ID)
	if err != nil {
		return errors.New("запис розкладу не знайдено")
	}

	if req.Room != "" {
		if err := schedule.OverrideRoom(req.Room); err != nil {
			return err
		}
	}

	if req.Teacher != "" {
		if err := schedule.ChangeTeacher(req.Teacher); err != nil {
			return err
		}
		if err := uc.scheduleRepo.Update(schedule); err != nil {
			return errors.New("помилка оновлення запису")
		}
	}

	return nil
}

// toResult конвертує доменну сутність у проміжну структуру
func (uc *ScheduleUseCase) toResult(s *entities.Schedule) ScheduleResult {
	return ScheduleResult{
		ID:          s.ID(),
		Subject:     s.Subject(),
		Teacher:     s.Teacher(),
		Group:       s.GroupName(),
		Day:         s.Day().String(),
		TimeSlot:    s.TimeSlot().String(),
		Room:        s.EffectiveRoom().String(),
		MaxStudents: s.MaxStudents().Value(),
		Enrolled:    s.Enrolled(),
		Available:   s.IsAvailable(),
	}
}

// --- Domain Events ---

// ScheduleCreatedEvent — подія створення нового запису в розкладі
type ScheduleCreatedEvent struct {
	ScheduleID int
	Subject    string
	Teacher    string
	Day        string
	TimeSlot   string
}

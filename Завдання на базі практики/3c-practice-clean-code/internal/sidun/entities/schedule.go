package entities

import (
	"errors"
	"fmt"
)

// --- Value Objects (незмінні, самовалідні) ---

// Day — дозволений день тижня (Value Object)
type Day string

const (
	Monday    Day = "Monday"
	Tuesday   Day = "Tuesday"
	Wednesday Day = "Wednesday"
	Thursday  Day = "Thursday"
	Friday    Day = "Friday"
)

func NewDay(raw string) (Day, error) {
	switch Day(raw) {
	case Monday, Tuesday, Wednesday, Thursday, Friday:
		return Day(raw), nil
	default:
		return "", fmt.Errorf("недопустимий день тижня: %s", raw)
	}
}

func (d Day) String() string { return string(d) }

// TimeSlot — часовий слот заняття (Value Object)
type TimeSlot struct {
	value string
}

func NewTimeSlot(raw string) (TimeSlot, error) {
	if raw == "" {
		return TimeSlot{}, errors.New("часовий слот не може бути порожнім")
	}
	return TimeSlot{value: raw}, nil
}

func (ts TimeSlot) String() string { return ts.value }

// Room — аудиторія (Value Object)
type Room struct {
	value string
}

func NewRoom(raw string) (Room, error) {
	if raw == "" {
		return Room{}, errors.New("аудиторія не може бути порожньою")
	}
	return Room{value: raw}, nil
}

func (r Room) String() string { return r.value }

// Capacity — максимальна кількість студентів (Value Object)
type Capacity struct {
	value int
}

func NewCapacity(max int) (Capacity, error) {
	if max <= 0 || max > 200 {
		return Capacity{}, fmt.Errorf("кількість студентів має бути від 1 до 200, отримано: %d", max)
	}
	return Capacity{value: max}, nil
}

func (c Capacity) Value() int  { return c.value }
func (c Capacity) IsAvailable(enrolled int) bool {
	return enrolled < c.value
}

// ScheduleStatus — статус запису розкладу (Value Object)
type ScheduleStatus string

const (
	StatusActive   ScheduleStatus = "active"
	StatusOverride ScheduleStatus = "override"
)

// --- Aggregate Root: Schedule ---

// Schedule — корінь агрегату, що контролює заняття в розкладі
type Schedule struct {
	id          int
	subject     string
	teacher     string
	groupName   string
	day         Day
	timeSlot    TimeSlot
	room        Room
	maxStudents Capacity
	enrolled    int
	overrides   map[string]string
}

// ScheduleParams об'єднує параметри для створення запису розкладу
type ScheduleParams struct {
	ID          int
	Subject     string
	Teacher     string
	GroupName   string
	Day         string
	TimeSlot    string
	Room        string
	MaxStudents int
	Enrolled    int
}

// NewSchedule створює новий запис розкладу з валідацією
func NewSchedule(params ScheduleParams) (*Schedule, error) {
	if params.Subject == "" {
		return nil, errors.New("предмет не може бути порожнім")
	}
	if params.Teacher == "" {
		return nil, errors.New("викладач не може бути порожнім")
	}

	day, err := NewDay(params.Day)
	if err != nil {
		return nil, err
	}

	ts, err := NewTimeSlot(params.TimeSlot)
	if err != nil {
		return nil, err
	}

	room, err := NewRoom(params.Room)
	if err != nil {
		return nil, err
	}

	cap, err := NewCapacity(params.MaxStudents)
	if err != nil {
		return nil, err
	}

	return &Schedule{
		id:          params.ID,
		subject:     params.Subject,
		teacher:     params.Teacher,
		groupName:   params.GroupName,
		day:         day,
		timeSlot:    ts,
		room:        room,
		maxStudents: cap,
		enrolled:    params.Enrolled,
		overrides:   make(map[string]string),
	}, nil
}

// RestoreSchedule відновлює запис із бази (без повторної валідації capacity)
func RestoreSchedule(params ScheduleParams) *Schedule {
	return &Schedule{
		id:          params.ID,
		subject:     params.Subject,
		teacher:     params.Teacher,
		groupName:   params.GroupName,
		day:         Day(params.Day),
		timeSlot:    TimeSlot{value: params.TimeSlot},
		room:        Room{value: params.Room},
		maxStudents: Capacity{value: params.MaxStudents},
		enrolled:    params.Enrolled,
		overrides:   make(map[string]string),
	}
}

// --- Методи Aggregate Root ---

func (s *Schedule) ID() int               { return s.id }
func (s *Schedule) Subject() string        { return s.subject }
func (s *Schedule) Teacher() string        { return s.teacher }
func (s *Schedule) GroupName() string      { return s.groupName }
func (s *Schedule) Day() Day              { return s.day }
func (s *Schedule) TimeSlot() TimeSlot    { return s.timeSlot }
func (s *Schedule) MaxStudents() Capacity { return s.maxStudents }
func (s *Schedule) Enrolled() int         { return s.enrolled }

// EffectiveRoom повертає аудиторію з урахуванням перевизначень
func (s *Schedule) EffectiveRoom() Room {
	if override, ok := s.overrides["room"]; ok {
		return Room{value: override}
	}
	return s.room
}

func (s *Schedule) Room() Room { return s.room }

// IsAvailable перевіряє, чи є вільні місця
func (s *Schedule) IsAvailable() bool {
	return s.maxStudents.IsAvailable(s.enrolled)
}

// OverrideRoom встановлює тимчасове перевизначення аудиторії
func (s *Schedule) OverrideRoom(newRoom string) error {
	if newRoom == "" {
		return errors.New("нова аудиторія не може бути порожньою")
	}
	s.overrides["room"] = newRoom
	return nil
}

// ChangeTeacher змінює викладача
func (s *Schedule) ChangeTeacher(newTeacher string) error {
	if newTeacher == "" {
		return errors.New("ім'я викладача не може бути порожнім")
	}
	s.teacher = newTeacher
	return nil
}

// HasConflictWith перевіряє конфлікт з іншим записом
func (s *Schedule) HasConflictWith(other *Schedule) bool {
	if s.day != other.day || s.timeSlot.value != other.timeSlot.value {
		return false
	}
	if s.id == other.id && s.id != 0 {
		return false
	}
	// Конфлікт аудиторії або викладача
	return s.room.value == other.room.value || s.teacher == other.teacher
}

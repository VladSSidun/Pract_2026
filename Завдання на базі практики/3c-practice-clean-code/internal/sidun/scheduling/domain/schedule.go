// Package domain визначає доменну модель контексту Scheduling.
// У контексті Scheduling "заняття" — це запис розкладу з предметом,
// викладачем, аудиторією та групою. Цей контекст відповідає за
// валідацію, перевірку конфліктів та управління записами.
package domain

// ClassEntry — модель заняття в контексті Scheduling
type ClassEntry struct {
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

// ScheduleCreatedEvent — подія, що заняття створено.
// Публікується контекстом Scheduling, обробляється контекстом Reporting.
type ScheduleCreatedEvent struct {
	ScheduleID int
	Subject    string
	Teacher    string
	Day        string
	TimeSlot   string
}

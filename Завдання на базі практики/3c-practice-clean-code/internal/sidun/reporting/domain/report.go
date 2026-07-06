// Package domain визначає доменну модель контексту Reporting.
// У контексті Reporting "заняття" — це елемент звітності з
// навантаженням, відвідуваністю та статистикою. Цей контекст
// не знає деталей розкладу і отримує дані через доменні події.
package domain

// TeacherReport — звіт по навантаженню викладача (контекст Reporting)
type TeacherReport struct {
	TeacherName  string
	TotalClasses int
	TotalHours   int
}

// AttendanceRecord — запис відвідуваності (контекст Reporting)
type AttendanceRecord struct {
	ScheduleRef string // посилання на розклад, не залежність
	StudentID   int
	Present     bool
}

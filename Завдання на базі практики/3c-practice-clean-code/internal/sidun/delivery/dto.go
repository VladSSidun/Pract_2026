package delivery

// --- Request DTOs (парсинг HTTP/JSON) ---

// LoginRequest — DTO для входу
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AddScheduleDTO — DTO для додавання заняття
type AddScheduleDTO struct {
	Subject     string `json:"subject"`
	Teacher     string `json:"teacher"`
	Group       string `json:"group"`
	Day         string `json:"day"`
	TimeSlot    string `json:"time_slot"`
	Room        string `json:"room"`
	MaxStudents int    `json:"max_students"`
}

// UpdateScheduleDTO — DTO для оновлення заняття
type UpdateScheduleDTO struct {
	Room    string `json:"room"`
	Teacher string `json:"teacher"`
}

// --- Response DTOs (форматування JSON-відповіді) ---

// LoginResponse — відповідь після входу
type LoginResponse struct {
	Message  string `json:"message"`
	UserID   int    `json:"user_id"`
	UserName string `json:"user_name"`
	Token    string `json:"token"`
}

// ScheduleResponse — один запис розкладу
type ScheduleResponse struct {
	ID       int    `json:"id"`
	Subject  string `json:"subject"`
	Teacher  string `json:"teacher"`
	Group    string `json:"group"`
	Day      string `json:"day"`
	Time     string `json:"time"`
	Room     string `json:"room"`
	Max      int    `json:"max"`
	Enrolled int    `json:"enrolled"`
}

// ErrorResponse — помилка
type ErrorResponse struct {
	Error string `json:"error"`
}

// SuccessResponse — загальна успішна відповідь
type SuccessResponse struct {
	Message string `json:"message"`
	ID      int    `json:"id,omitempty"`
}

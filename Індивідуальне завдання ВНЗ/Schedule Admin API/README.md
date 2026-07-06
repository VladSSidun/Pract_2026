# Schedule Admin API

REST API для розкладу занять ЗВО з JWT-автентифікацією та розмежуванням прав
доступу `admin` / `student`. Адміністратор керує групами, предметами,
викладачами та розкладом; студент має доступ лише на читання.

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Python 3.11+, **FastAPI** |
| ORM / DB | **SQLAlchemy 2.0 (async)**, **SQLite** (`aiosqlite`) |
| Validation | **Pydantic v2**, `pydantic-settings` |
| Authentication | JWT (`python-jose`), password hashing (`passlib[bcrypt]`) |
| Tests | **pytest**, **pytest-asyncio**, **httpx.AsyncClient** |

## Project Structure

```
app/
├── main.py                  # точка входу, FastAPI app, lifespan, CORS
├── core/
│   ├── config.py            # Settings (.env): database_url, secret_key, algorithm, expire
│   ├── database.py           # async engine/session, Base, get_db, create_all
│   └── security.py          # хешування паролів, створення/декодування JWT
├── models/                  # SQLAlchemy-моделі
│   ├── user.py
│   ├── group.py
│   ├── subject.py
│   ├── teacher.py
│   └── schedule.py
├── schemas/                 # Pydantic-схеми (Create/Update/Response)
│   ├── user.py, group.py, subject.py, teacher.py, schedule.py, token.py
├── repositories/            # SQL-запити через AsyncSession
│   ├── user_repository.py
│   ├── group_repository.py
│   ├── subject_repository.py
│   ├── teacher_repository.py
│   └── schedule_repository.py
├── services/                 # бізнес-логіка, перевірка конфліктів, seed-дані
│   ├── auth_service.py
│   ├── group_service.py
│   ├── subject_service.py
│   ├── teacher_service.py
│   └── schedule_service.py
├── routes/                   # FastAPI-роутери
│   ├── auth.py
│   ├── groups.py
│   ├── subjects.py
│   ├── teachers.py
│   └── schedule.py
└── dependencies/
    └── auth.py               # get_current_user, require_admin

tests/
├── conftest.py               # async_client (in-memory SQLite), admin_token, student_token
├── test_auth.py
├── test_groups.py
├── test_subjects.py
├── test_teachers.py
└── test_schedule.py
```

Шарова архітектура: `routes` → `services` → `repositories` → `models`.
Кожен шар звертається лише до сусіднього нижчого рівня.

## Getting Started

### 1. Install dependencies

```bash
python -m venv venv
venv\Scripts\activate        # Windows
# source venv/bin/activate   # macOS / Linux
pip install -r requirements.txt
```

### 2. Configure environment

Скопіюйте `.env.example` у `.env`:

```env
DATABASE_URL=sqlite+aiosqlite:///./schedule.db
SECRET_KEY=your-secret-key-change-in-production
ALGORITHM=HS256
ACCESS_TOKEN_EXPIRE_MINUTES=60
```

### 3. Run the server

```bash
uvicorn app.main:app --reload
```

Застосунок: `http://127.0.0.1:8000`
Swagger-документація: `http://127.0.0.1:8000/docs`

## Tests

```bash
pytest tests/ -v
```

## API Reference

### Auth (`/api/v1/auth`)

| Method | Endpoint | Access | Description |
|--------|----------|--------|-------------|
| `POST` | `/register` | public | Реєстрація (username, email, password, role) |
| `POST` | `/login` | public | Вхід, отримання JWT |
| `GET` | `/me` | auth | Дані поточного користувача |

### Groups / Subjects / Teachers (`/api/v1/groups`, `/subjects`, `/teachers`)

| Method | Endpoint | Access | Description |
|--------|----------|--------|-------------|
| `GET` | `/` | public | Список |
| `GET` | `/{id}` | public | Отримати за ID (404, якщо немає) |
| `POST` | `/` | admin | Створити |
| `PUT` | `/{id}` | admin | Оновити |
| `DELETE` | `/{id}` | admin | Видалити |

### Schedule (`/api/v1/schedule`)

| Method | Endpoint | Access | Description |
|--------|----------|--------|-------------|
| `GET` | `/?group_id=&teacher_id=&day_of_week=` | public | Список із фільтрами, вкладені subject/teacher/group |
| `GET` | `/{id}` | public | Отримати запис розкладу за ID |
| `GET` | `/group/{group_id}` | public | Розклад групи |
| `GET` | `/teacher/{teacher_id}` | public | Розклад викладача |
| `POST` | `/` | admin | Створити запис (перевірка конфлікту 409) |
| `PUT` | `/{id}` | admin | Оновити запис (перевірка конфлікту 409, окрім себе) |
| `DELETE` | `/{id}` | admin | Видалити запис |
| `POST` | `/seed` | admin | Згенерувати тестові дані (групи/предмети/викладачі/розклад) |

Конфлікт розкладу (`409 Schedule conflict detected`) виникає, коли на той
самий день тижня й слот часу вже зайнята або аудиторія, або викладач.

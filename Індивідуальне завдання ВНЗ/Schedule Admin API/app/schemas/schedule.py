from datetime import datetime

from pydantic import BaseModel, ConfigDict

from app.schemas.group import GroupResponse
from app.schemas.subject import SubjectResponse
from app.schemas.teacher import TeacherResponse


class ScheduleCreate(BaseModel):
    subject_id: int
    teacher_id: int
    group_id: int
    day_of_week: int
    time_slot: str
    room: str
    max_students: int = 30
    notes: str | None = None


class ScheduleUpdate(BaseModel):
    subject_id: int | None = None
    teacher_id: int | None = None
    group_id: int | None = None
    day_of_week: int | None = None
    time_slot: str | None = None
    room: str | None = None
    max_students: int | None = None
    notes: str | None = None


class ScheduleResponse(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: int
    day_of_week: int
    time_slot: str
    room: str
    max_students: int
    notes: str | None = None
    created_at: datetime
    updated_at: datetime
    subject: SubjectResponse
    teacher: TeacherResponse
    group: GroupResponse

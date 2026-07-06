from datetime import datetime

from pydantic import BaseModel, ConfigDict, EmailStr


class TeacherCreate(BaseModel):
    first_name: str
    last_name: str
    email: EmailStr | None = None
    department: str | None = None


class TeacherUpdate(BaseModel):
    first_name: str | None = None
    last_name: str | None = None
    email: EmailStr | None = None
    department: str | None = None


class TeacherResponse(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: int
    first_name: str
    last_name: str
    email: EmailStr | None = None
    department: str | None = None
    created_at: datetime

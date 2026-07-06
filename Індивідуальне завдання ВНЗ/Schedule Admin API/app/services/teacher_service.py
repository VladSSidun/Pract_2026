from fastapi import HTTPException, status
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.teacher import Teacher
from app.repositories import teacher_repository
from app.schemas.teacher import TeacherCreate, TeacherUpdate


async def list_teachers(db: AsyncSession) -> list[Teacher]:
    return await teacher_repository.get_all(db)


async def get_teacher(db: AsyncSession, teacher_id: int) -> Teacher:
    teacher = await teacher_repository.get_by_id(db, teacher_id)
    if not teacher:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="Teacher not found")
    return teacher


async def create_teacher(db: AsyncSession, data: TeacherCreate) -> Teacher:
    if data.email and await teacher_repository.get_by_email(db, data.email):
        raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail="Teacher email already exists")
    return await teacher_repository.create(
        db,
        first_name=data.first_name,
        last_name=data.last_name,
        email=data.email,
        department=data.department,
    )


async def update_teacher(db: AsyncSession, teacher_id: int, data: TeacherUpdate) -> Teacher:
    teacher = await get_teacher(db, teacher_id)
    fields = data.model_dump(exclude_unset=True)
    if fields.get("email") and fields["email"] != teacher.email:
        if await teacher_repository.get_by_email(db, fields["email"]):
            raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail="Teacher email already exists")
    return await teacher_repository.update(db, teacher, **fields)


async def delete_teacher(db: AsyncSession, teacher_id: int) -> None:
    teacher = await get_teacher(db, teacher_id)
    await teacher_repository.delete(db, teacher)

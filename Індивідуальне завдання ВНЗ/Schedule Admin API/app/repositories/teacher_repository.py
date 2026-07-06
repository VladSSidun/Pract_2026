from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.teacher import Teacher


async def get_all(db: AsyncSession) -> list[Teacher]:
    result = await db.execute(select(Teacher).order_by(Teacher.id))
    return list(result.scalars().all())


async def get_by_id(db: AsyncSession, teacher_id: int) -> Teacher | None:
    result = await db.execute(select(Teacher).where(Teacher.id == teacher_id))
    return result.scalar_one_or_none()


async def get_by_email(db: AsyncSession, email: str) -> Teacher | None:
    result = await db.execute(select(Teacher).where(Teacher.email == email))
    return result.scalar_one_or_none()


async def create(db: AsyncSession, first_name: str, last_name: str, email: str | None, department: str | None) -> Teacher:
    teacher = Teacher(first_name=first_name, last_name=last_name, email=email, department=department)
    db.add(teacher)
    await db.commit()
    await db.refresh(teacher)
    return teacher


async def update(db: AsyncSession, teacher: Teacher, **fields) -> Teacher:
    for key, value in fields.items():
        if value is not None:
            setattr(teacher, key, value)
    await db.commit()
    await db.refresh(teacher)
    return teacher


async def delete(db: AsyncSession, teacher: Teacher) -> None:
    await db.delete(teacher)
    await db.commit()

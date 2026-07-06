from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.subject import Subject


async def get_all(db: AsyncSession) -> list[Subject]:
    result = await db.execute(select(Subject).order_by(Subject.id))
    return list(result.scalars().all())


async def get_by_id(db: AsyncSession, subject_id: int) -> Subject | None:
    result = await db.execute(select(Subject).where(Subject.id == subject_id))
    return result.scalar_one_or_none()


async def get_by_name(db: AsyncSession, name: str) -> Subject | None:
    result = await db.execute(select(Subject).where(Subject.name == name))
    return result.scalar_one_or_none()


async def create(db: AsyncSession, name: str, description: str | None) -> Subject:
    subject = Subject(name=name, description=description)
    db.add(subject)
    await db.commit()
    await db.refresh(subject)
    return subject


async def update(db: AsyncSession, subject: Subject, **fields) -> Subject:
    for key, value in fields.items():
        if value is not None:
            setattr(subject, key, value)
    await db.commit()
    await db.refresh(subject)
    return subject


async def delete(db: AsyncSession, subject: Subject) -> None:
    await db.delete(subject)
    await db.commit()

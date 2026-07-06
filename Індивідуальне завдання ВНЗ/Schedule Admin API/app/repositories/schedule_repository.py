from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from app.models.schedule import Schedule

_EAGER = (
    selectinload(Schedule.subject),
    selectinload(Schedule.teacher),
    selectinload(Schedule.group),
)


async def get_all(
    db: AsyncSession,
    group_id: int | None = None,
    teacher_id: int | None = None,
    day_of_week: int | None = None,
) -> list[Schedule]:
    query = select(Schedule).options(*_EAGER)
    if group_id is not None:
        query = query.where(Schedule.group_id == group_id)
    if teacher_id is not None:
        query = query.where(Schedule.teacher_id == teacher_id)
    if day_of_week is not None:
        query = query.where(Schedule.day_of_week == day_of_week)
    query = query.order_by(Schedule.day_of_week, Schedule.time_slot)
    result = await db.execute(query)
    return list(result.scalars().all())


async def get_by_id(db: AsyncSession, schedule_id: int) -> Schedule | None:
    query = select(Schedule).options(*_EAGER).where(Schedule.id == schedule_id)
    result = await db.execute(query)
    return result.scalar_one_or_none()


async def get_by_group(db: AsyncSession, group_id: int) -> list[Schedule]:
    return await get_all(db, group_id=group_id)


async def get_by_teacher(db: AsyncSession, teacher_id: int) -> list[Schedule]:
    return await get_all(db, teacher_id=teacher_id)


async def find_conflict(
    db: AsyncSession,
    day_of_week: int,
    time_slot: str,
    room: str,
    teacher_id: int,
    exclude_id: int | None = None,
) -> Schedule | None:
    query = select(Schedule).where(
        Schedule.day_of_week == day_of_week,
        Schedule.time_slot == time_slot,
        (Schedule.room == room) | (Schedule.teacher_id == teacher_id),
    )
    if exclude_id is not None:
        query = query.where(Schedule.id != exclude_id)
    result = await db.execute(query)
    return result.scalars().first()


async def count_all(db: AsyncSession) -> int:
    result = await db.execute(select(Schedule))
    return len(result.scalars().all())


async def create(db: AsyncSession, **fields) -> Schedule:
    schedule = Schedule(**fields)
    db.add(schedule)
    await db.commit()
    await db.refresh(schedule)
    return await get_by_id(db, schedule.id)


async def update(db: AsyncSession, schedule: Schedule, **fields) -> Schedule:
    for key, value in fields.items():
        if value is not None:
            setattr(schedule, key, value)
    await db.commit()
    await db.refresh(schedule)
    return await get_by_id(db, schedule.id)


async def delete(db: AsyncSession, schedule: Schedule) -> None:
    await db.delete(schedule)
    await db.commit()

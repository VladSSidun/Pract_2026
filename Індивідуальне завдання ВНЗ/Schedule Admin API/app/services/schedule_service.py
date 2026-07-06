from fastapi import HTTPException, status
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.schedule import Schedule
from app.repositories import group_repository, schedule_repository, subject_repository, teacher_repository
from app.schemas.schedule import ScheduleCreate, ScheduleUpdate


async def list_schedules(
    db: AsyncSession,
    group_id: int | None = None,
    teacher_id: int | None = None,
    day_of_week: int | None = None,
) -> list[Schedule]:
    return await schedule_repository.get_all(db, group_id=group_id, teacher_id=teacher_id, day_of_week=day_of_week)


async def get_schedule(db: AsyncSession, schedule_id: int) -> Schedule:
    schedule = await schedule_repository.get_by_id(db, schedule_id)
    if not schedule:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="Schedule entry not found")
    return schedule


async def get_by_group(db: AsyncSession, group_id: int) -> list[Schedule]:
    return await schedule_repository.get_by_group(db, group_id)


async def get_by_teacher(db: AsyncSession, teacher_id: int) -> list[Schedule]:
    return await schedule_repository.get_by_teacher(db, teacher_id)


async def _check_conflict(
    db: AsyncSession,
    day_of_week: int,
    time_slot: str,
    room: str,
    teacher_id: int,
    exclude_id: int | None = None,
) -> None:
    conflict = await schedule_repository.find_conflict(db, day_of_week, time_slot, room, teacher_id, exclude_id)
    if conflict:
        raise HTTPException(status_code=status.HTTP_409_CONFLICT, detail="Schedule conflict detected")


async def create_schedule(db: AsyncSession, data: ScheduleCreate) -> Schedule:
    await _check_conflict(db, data.day_of_week, data.time_slot, data.room, data.teacher_id)
    return await schedule_repository.create(db, **data.model_dump())


async def update_schedule(db: AsyncSession, schedule_id: int, data: ScheduleUpdate) -> Schedule:
    schedule = await get_schedule(db, schedule_id)
    fields = data.model_dump(exclude_unset=True)

    day_of_week = fields.get("day_of_week", schedule.day_of_week)
    time_slot = fields.get("time_slot", schedule.time_slot)
    room = fields.get("room", schedule.room)
    teacher_id = fields.get("teacher_id", schedule.teacher_id)

    await _check_conflict(db, day_of_week, time_slot, room, teacher_id, exclude_id=schedule_id)
    return await schedule_repository.update(db, schedule, **fields)


async def delete_schedule(db: AsyncSession, schedule_id: int) -> None:
    schedule = await get_schedule(db, schedule_id)
    await schedule_repository.delete(db, schedule)


async def generate_seed_data(db: AsyncSession) -> dict:
    existing = await schedule_repository.count_all(db)
    if existing > 0:
        return {"message": "Already seeded"}

    group_names = ["ІПЗ-31", "ІПЗ-32", "ІПЗ-41"]
    groups = []
    for name in group_names:
        group = await group_repository.get_by_name(db, name)
        if not group:
            group = await group_repository.create(db, name)
        groups.append(group)

    subject_defs = [
        ("Бази даних", "Проєктування та використання СУБД"),
        ("Веб-програмування", "Розробка серверних та клієнтських застосунків"),
        ("Алгоритми та структури даних", "Основи алгоритмізації"),
        ("Операційні системи", "Принципи побудови ОС"),
        ("Математичний аналіз", "Диференціальне та інтегральне числення"),
    ]
    subjects = []
    for name, description in subject_defs:
        subject = await subject_repository.get_by_name(db, name)
        if not subject:
            subject = await subject_repository.create(db, name, description)
        subjects.append(subject)

    teacher_defs = [
        ("Олена", "Коваленко", "kovalenko@example.edu", "Кафедра програмної інженерії"),
        ("Ігор", "Шевченко", "shevchenko@example.edu", "Кафедра програмної інженерії"),
        ("Марія", "Бондаренко", "bondarenko@example.edu", "Кафедра математики"),
        ("Андрій", "Мельник", "melnyk@example.edu", "Кафедра комп'ютерних наук"),
    ]
    teachers = []
    for first_name, last_name, email, department in teacher_defs:
        teacher = await teacher_repository.create(db, first_name, last_name, email, department)
        teachers.append(teacher)

    entries = [
        (groups[0], subjects[0], teachers[0], 1, "1", "101"),
        (groups[0], subjects[1], teachers[1], 1, "2", "102"),
        (groups[0], subjects[2], teachers[3], 2, "1", "101"),
        (groups[1], subjects[3], teachers[2], 1, "1", "103"),
        (groups[1], subjects[4], teachers[2], 2, "2", "104"),
        (groups[1], subjects[0], teachers[0], 3, "1", "101"),
        (groups[2], subjects[1], teachers[1], 2, "1", "105"),
        (groups[2], subjects[2], teachers[3], 4, "1", "101"),
    ]

    for group, subject, teacher, day_of_week, time_slot, room in entries:
        await schedule_repository.create(
            db,
            subject_id=subject.id,
            teacher_id=teacher.id,
            group_id=group.id,
            day_of_week=day_of_week,
            time_slot=time_slot,
            room=room,
        )

    return {"message": "Seed data generated"}

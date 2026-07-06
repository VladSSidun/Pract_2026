from fastapi import APIRouter, Depends, status
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import get_db
from app.dependencies.auth import require_admin
from app.schemas.schedule import ScheduleCreate, ScheduleResponse, ScheduleUpdate
from app.services import schedule_service

router = APIRouter(prefix="/api/v1/schedule", tags=["schedule"])


@router.get("/", response_model=list[ScheduleResponse])
async def list_schedule(
    group_id: int | None = None,
    teacher_id: int | None = None,
    day_of_week: int | None = None,
    db: AsyncSession = Depends(get_db),
):
    return await schedule_service.list_schedules(db, group_id=group_id, teacher_id=teacher_id, day_of_week=day_of_week)


@router.get("/{schedule_id}", response_model=ScheduleResponse)
async def get_schedule(schedule_id: int, db: AsyncSession = Depends(get_db)):
    return await schedule_service.get_schedule(db, schedule_id)


@router.get("/group/{group_id}", response_model=list[ScheduleResponse])
async def get_by_group(group_id: int, db: AsyncSession = Depends(get_db)):
    return await schedule_service.get_by_group(db, group_id)


@router.get("/teacher/{teacher_id}", response_model=list[ScheduleResponse])
async def get_by_teacher(teacher_id: int, db: AsyncSession = Depends(get_db)):
    return await schedule_service.get_by_teacher(db, teacher_id)


@router.post("/", response_model=ScheduleResponse, status_code=status.HTTP_201_CREATED, dependencies=[Depends(require_admin)])
async def create_schedule(data: ScheduleCreate, db: AsyncSession = Depends(get_db)):
    return await schedule_service.create_schedule(db, data)


@router.put("/{schedule_id}", response_model=ScheduleResponse, dependencies=[Depends(require_admin)])
async def update_schedule(schedule_id: int, data: ScheduleUpdate, db: AsyncSession = Depends(get_db)):
    return await schedule_service.update_schedule(db, schedule_id, data)


@router.delete("/{schedule_id}", status_code=status.HTTP_204_NO_CONTENT, dependencies=[Depends(require_admin)])
async def delete_schedule(schedule_id: int, db: AsyncSession = Depends(get_db)):
    await schedule_service.delete_schedule(db, schedule_id)


@router.post("/seed", dependencies=[Depends(require_admin)])
async def seed(db: AsyncSession = Depends(get_db)):
    return await schedule_service.generate_seed_data(db)

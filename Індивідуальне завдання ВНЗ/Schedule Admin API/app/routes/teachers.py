from fastapi import APIRouter, Depends, status
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import get_db
from app.dependencies.auth import require_admin
from app.schemas.teacher import TeacherCreate, TeacherResponse, TeacherUpdate
from app.services import teacher_service

router = APIRouter(prefix="/api/v1/teachers", tags=["teachers"])


@router.get("/", response_model=list[TeacherResponse])
async def list_teachers(db: AsyncSession = Depends(get_db)):
    return await teacher_service.list_teachers(db)


@router.get("/{teacher_id}", response_model=TeacherResponse)
async def get_teacher(teacher_id: int, db: AsyncSession = Depends(get_db)):
    return await teacher_service.get_teacher(db, teacher_id)


@router.post("/", response_model=TeacherResponse, status_code=status.HTTP_201_CREATED, dependencies=[Depends(require_admin)])
async def create_teacher(data: TeacherCreate, db: AsyncSession = Depends(get_db)):
    return await teacher_service.create_teacher(db, data)


@router.put("/{teacher_id}", response_model=TeacherResponse, dependencies=[Depends(require_admin)])
async def update_teacher(teacher_id: int, data: TeacherUpdate, db: AsyncSession = Depends(get_db)):
    return await teacher_service.update_teacher(db, teacher_id, data)


@router.delete("/{teacher_id}", status_code=status.HTTP_204_NO_CONTENT, dependencies=[Depends(require_admin)])
async def delete_teacher(teacher_id: int, db: AsyncSession = Depends(get_db)):
    await teacher_service.delete_teacher(db, teacher_id)

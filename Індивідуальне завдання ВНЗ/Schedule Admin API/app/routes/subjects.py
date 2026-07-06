from fastapi import APIRouter, Depends, status
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import get_db
from app.dependencies.auth import require_admin
from app.schemas.subject import SubjectCreate, SubjectResponse, SubjectUpdate
from app.services import subject_service

router = APIRouter(prefix="/api/v1/subjects", tags=["subjects"])


@router.get("/", response_model=list[SubjectResponse])
async def list_subjects(db: AsyncSession = Depends(get_db)):
    return await subject_service.list_subjects(db)


@router.get("/{subject_id}", response_model=SubjectResponse)
async def get_subject(subject_id: int, db: AsyncSession = Depends(get_db)):
    return await subject_service.get_subject(db, subject_id)


@router.post("/", response_model=SubjectResponse, status_code=status.HTTP_201_CREATED, dependencies=[Depends(require_admin)])
async def create_subject(data: SubjectCreate, db: AsyncSession = Depends(get_db)):
    return await subject_service.create_subject(db, data)


@router.put("/{subject_id}", response_model=SubjectResponse, dependencies=[Depends(require_admin)])
async def update_subject(subject_id: int, data: SubjectUpdate, db: AsyncSession = Depends(get_db)):
    return await subject_service.update_subject(db, subject_id, data)


@router.delete("/{subject_id}", status_code=status.HTTP_204_NO_CONTENT, dependencies=[Depends(require_admin)])
async def delete_subject(subject_id: int, db: AsyncSession = Depends(get_db)):
    await subject_service.delete_subject(db, subject_id)

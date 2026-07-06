from fastapi import HTTPException, status
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.subject import Subject
from app.repositories import subject_repository
from app.schemas.subject import SubjectCreate, SubjectUpdate


async def list_subjects(db: AsyncSession) -> list[Subject]:
    return await subject_repository.get_all(db)


async def get_subject(db: AsyncSession, subject_id: int) -> Subject:
    subject = await subject_repository.get_by_id(db, subject_id)
    if not subject:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="Subject not found")
    return subject


async def create_subject(db: AsyncSession, data: SubjectCreate) -> Subject:
    if await subject_repository.get_by_name(db, data.name):
        raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail="Subject name already exists")
    return await subject_repository.create(db, name=data.name, description=data.description)


async def update_subject(db: AsyncSession, subject_id: int, data: SubjectUpdate) -> Subject:
    subject = await get_subject(db, subject_id)
    fields = data.model_dump(exclude_unset=True)
    if fields.get("name") and fields["name"] != subject.name:
        if await subject_repository.get_by_name(db, fields["name"]):
            raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail="Subject name already exists")
    return await subject_repository.update(db, subject, **fields)


async def delete_subject(db: AsyncSession, subject_id: int) -> None:
    subject = await get_subject(db, subject_id)
    await subject_repository.delete(db, subject)

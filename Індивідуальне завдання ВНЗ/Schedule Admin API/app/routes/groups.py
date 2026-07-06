from fastapi import APIRouter, Depends, status
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import get_db
from app.dependencies.auth import require_admin
from app.schemas.group import GroupCreate, GroupResponse, GroupUpdate
from app.services import group_service

router = APIRouter(prefix="/api/v1/groups", tags=["groups"])


@router.get("/", response_model=list[GroupResponse])
async def list_groups(db: AsyncSession = Depends(get_db)):
    return await group_service.list_groups(db)


@router.get("/{group_id}", response_model=GroupResponse)
async def get_group(group_id: int, db: AsyncSession = Depends(get_db)):
    return await group_service.get_group(db, group_id)


@router.post("/", response_model=GroupResponse, status_code=status.HTTP_201_CREATED, dependencies=[Depends(require_admin)])
async def create_group(data: GroupCreate, db: AsyncSession = Depends(get_db)):
    return await group_service.create_group(db, data)


@router.put("/{group_id}", response_model=GroupResponse, dependencies=[Depends(require_admin)])
async def update_group(group_id: int, data: GroupUpdate, db: AsyncSession = Depends(get_db)):
    return await group_service.update_group(db, group_id, data)


@router.delete("/{group_id}", status_code=status.HTTP_204_NO_CONTENT, dependencies=[Depends(require_admin)])
async def delete_group(group_id: int, db: AsyncSession = Depends(get_db)):
    await group_service.delete_group(db, group_id)

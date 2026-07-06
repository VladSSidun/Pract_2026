from fastapi import HTTPException, status
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.group import Group
from app.repositories import group_repository
from app.schemas.group import GroupCreate, GroupUpdate


async def list_groups(db: AsyncSession) -> list[Group]:
    return await group_repository.get_all(db)


async def get_group(db: AsyncSession, group_id: int) -> Group:
    group = await group_repository.get_by_id(db, group_id)
    if not group:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="Group not found")
    return group


async def create_group(db: AsyncSession, data: GroupCreate) -> Group:
    if await group_repository.get_by_name(db, data.name):
        raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail="Group name already exists")
    return await group_repository.create(db, name=data.name)


async def update_group(db: AsyncSession, group_id: int, data: GroupUpdate) -> Group:
    group = await get_group(db, group_id)
    fields = data.model_dump(exclude_unset=True)
    if fields.get("name") and fields["name"] != group.name:
        if await group_repository.get_by_name(db, fields["name"]):
            raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail="Group name already exists")
    return await group_repository.update(db, group, **fields)


async def delete_group(db: AsyncSession, group_id: int) -> None:
    group = await get_group(db, group_id)
    await group_repository.delete(db, group)

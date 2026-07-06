from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.group import Group


async def get_all(db: AsyncSession) -> list[Group]:
    result = await db.execute(select(Group).order_by(Group.id))
    return list(result.scalars().all())


async def get_by_id(db: AsyncSession, group_id: int) -> Group | None:
    result = await db.execute(select(Group).where(Group.id == group_id))
    return result.scalar_one_or_none()


async def get_by_name(db: AsyncSession, name: str) -> Group | None:
    result = await db.execute(select(Group).where(Group.name == name))
    return result.scalar_one_or_none()


async def create(db: AsyncSession, name: str) -> Group:
    group = Group(name=name)
    db.add(group)
    await db.commit()
    await db.refresh(group)
    return group


async def update(db: AsyncSession, group: Group, **fields) -> Group:
    for key, value in fields.items():
        if value is not None:
            setattr(group, key, value)
    await db.commit()
    await db.refresh(group)
    return group


async def delete(db: AsyncSession, group: Group) -> None:
    await db.delete(group)
    await db.commit()

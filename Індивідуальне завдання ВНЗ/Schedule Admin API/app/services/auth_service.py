from fastapi import HTTPException, status
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.security import create_access_token, hash_password, verify_password
from app.models.user import User
from app.repositories import user_repository
from app.schemas.token import Token
from app.schemas.user import UserCreate


async def register(db: AsyncSession, data: UserCreate) -> Token:
    if await user_repository.get_by_username(db, data.username):
        raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail="Username already taken")
    if await user_repository.get_by_email(db, data.email):
        raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail="Email already registered")

    user = await user_repository.create(
        db,
        username=data.username,
        email=data.email,
        hashed_password=hash_password(data.password),
        role=data.role,
    )
    return Token(access_token=create_access_token(user.id))


async def login(db: AsyncSession, username: str, password: str) -> Token:
    user = await user_repository.get_by_username(db, username)
    if not user or not verify_password(password, user.hashed_password):
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Invalid username or password")
    return Token(access_token=create_access_token(user.id))


async def get_me(current_user: User) -> User:
    return current_user

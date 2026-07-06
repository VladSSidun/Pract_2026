import pytest
import pytest_asyncio
from httpx import ASGITransport, AsyncClient
from sqlalchemy.ext.asyncio import async_sessionmaker, create_async_engine
from sqlalchemy.pool import StaticPool

from app.core.database import Base, get_db
from app.main import app

TEST_DATABASE_URL = "sqlite+aiosqlite://"


@pytest_asyncio.fixture
async def db_engine():
    engine = create_async_engine(TEST_DATABASE_URL, connect_args={"check_same_thread": False}, poolclass=StaticPool)
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
    yield engine
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.drop_all)
    await engine.dispose()


@pytest_asyncio.fixture
async def async_client(db_engine):
    TestingSessionLocal = async_sessionmaker(bind=db_engine, expire_on_commit=False)

    async def override_get_db():
        async with TestingSessionLocal() as session:
            yield session

    app.dependency_overrides[get_db] = override_get_db
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        yield client
    app.dependency_overrides.clear()


async def _register(client: AsyncClient, username: str, email: str, password: str, role: str) -> str:
    resp = await client.post(
        "/api/v1/auth/register",
        json={"username": username, "email": email, "password": password, "role": role},
    )
    assert resp.status_code == 201, resp.text
    return resp.json()["access_token"]


@pytest_asyncio.fixture
async def admin_token(async_client):
    return await _register(async_client, "admin", "admin@test.com", "password123", "admin")


@pytest_asyncio.fixture
async def student_token(async_client):
    return await _register(async_client, "student", "student@test.com", "password123", "student")


def auth_headers(token: str) -> dict:
    return {"Authorization": f"Bearer {token}"}

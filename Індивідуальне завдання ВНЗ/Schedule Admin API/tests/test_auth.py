import pytest

from tests.conftest import auth_headers


@pytest.mark.asyncio
async def test_register(async_client):
    resp = await async_client.post(
        "/api/v1/auth/register",
        json={"username": "user1", "email": "user1@test.com", "password": "password123"},
    )
    assert resp.status_code == 201
    assert "access_token" in resp.json()


@pytest.mark.asyncio
async def test_register_duplicate_username(async_client):
    await async_client.post(
        "/api/v1/auth/register",
        json={"username": "dup", "email": "a@test.com", "password": "password123"},
    )
    resp = await async_client.post(
        "/api/v1/auth/register",
        json={"username": "dup", "email": "b@test.com", "password": "password123"},
    )
    assert resp.status_code == 400


@pytest.mark.asyncio
async def test_login(async_client):
    await async_client.post(
        "/api/v1/auth/register",
        json={"username": "loginuser", "email": "login@test.com", "password": "password123"},
    )
    resp = await async_client.post(
        "/api/v1/auth/login",
        json={"username": "loginuser", "password": "password123"},
    )
    assert resp.status_code == 200
    assert "access_token" in resp.json()


@pytest.mark.asyncio
async def test_login_invalid_credentials(async_client):
    resp = await async_client.post(
        "/api/v1/auth/login",
        json={"username": "nouser", "password": "wrong"},
    )
    assert resp.status_code == 401


@pytest.mark.asyncio
async def test_me(async_client, student_token):
    resp = await async_client.get("/api/v1/auth/me", headers=auth_headers(student_token))
    assert resp.status_code == 200
    assert resp.json()["username"] == "student"


@pytest.mark.asyncio
async def test_me_unauthenticated(async_client):
    resp = await async_client.get("/api/v1/auth/me")
    assert resp.status_code == 401

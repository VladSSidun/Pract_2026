import pytest

from tests.conftest import auth_headers


@pytest.mark.asyncio
async def test_list_teachers(async_client, admin_token):
    await async_client.post(
        "/api/v1/teachers/",
        json={"first_name": "Олена", "last_name": "Коваленко"},
        headers=auth_headers(admin_token),
    )
    resp = await async_client.get("/api/v1/teachers/")
    assert resp.status_code == 200
    assert len(resp.json()) == 1


@pytest.mark.asyncio
async def test_get_teacher_by_id(async_client, admin_token):
    create_resp = await async_client.post(
        "/api/v1/teachers/",
        json={"first_name": "Ігор", "last_name": "Шевченко"},
        headers=auth_headers(admin_token),
    )
    teacher_id = create_resp.json()["id"]
    resp = await async_client.get(f"/api/v1/teachers/{teacher_id}")
    assert resp.status_code == 200
    assert resp.json()["last_name"] == "Шевченко"


@pytest.mark.asyncio
async def test_get_teacher_not_found(async_client):
    resp = await async_client.get("/api/v1/teachers/999")
    assert resp.status_code == 404


@pytest.mark.asyncio
async def test_create_teacher_admin(async_client, admin_token):
    resp = await async_client.post(
        "/api/v1/teachers/",
        json={"first_name": "Марія", "last_name": "Бондаренко"},
        headers=auth_headers(admin_token),
    )
    assert resp.status_code == 201
    assert resp.json()["first_name"] == "Марія"


@pytest.mark.asyncio
async def test_create_teacher_student_forbidden(async_client, student_token):
    resp = await async_client.post(
        "/api/v1/teachers/",
        json={"first_name": "Андрій", "last_name": "Мельник"},
        headers=auth_headers(student_token),
    )
    assert resp.status_code == 403


@pytest.mark.asyncio
async def test_create_teacher_unauthenticated(async_client):
    resp = await async_client.post(
        "/api/v1/teachers/", json={"first_name": "Тест", "last_name": "Тестенко"}
    )
    assert resp.status_code == 401


@pytest.mark.asyncio
async def test_update_teacher_admin(async_client, admin_token):
    create_resp = await async_client.post(
        "/api/v1/teachers/",
        json={"first_name": "Наталія", "last_name": "Гончар"},
        headers=auth_headers(admin_token),
    )
    teacher_id = create_resp.json()["id"]
    resp = await async_client.put(
        f"/api/v1/teachers/{teacher_id}", json={"department": "Кафедра фізики"}, headers=auth_headers(admin_token)
    )
    assert resp.status_code == 200
    assert resp.json()["department"] == "Кафедра фізики"


@pytest.mark.asyncio
async def test_delete_teacher_admin(async_client, admin_token):
    create_resp = await async_client.post(
        "/api/v1/teachers/",
        json={"first_name": "Петро", "last_name": "Іваненко"},
        headers=auth_headers(admin_token),
    )
    teacher_id = create_resp.json()["id"]
    resp = await async_client.delete(f"/api/v1/teachers/{teacher_id}", headers=auth_headers(admin_token))
    assert resp.status_code == 204

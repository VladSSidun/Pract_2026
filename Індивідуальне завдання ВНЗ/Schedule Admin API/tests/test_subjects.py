import pytest

from tests.conftest import auth_headers


@pytest.mark.asyncio
async def test_list_subjects(async_client, admin_token):
    await async_client.post(
        "/api/v1/subjects/", json={"name": "Бази даних", "description": "СУБД"}, headers=auth_headers(admin_token)
    )
    resp = await async_client.get("/api/v1/subjects/")
    assert resp.status_code == 200
    assert len(resp.json()) == 1


@pytest.mark.asyncio
async def test_get_subject_by_id(async_client, admin_token):
    create_resp = await async_client.post(
        "/api/v1/subjects/", json={"name": "Веб-програмування"}, headers=auth_headers(admin_token)
    )
    subject_id = create_resp.json()["id"]
    resp = await async_client.get(f"/api/v1/subjects/{subject_id}")
    assert resp.status_code == 200
    assert resp.json()["name"] == "Веб-програмування"


@pytest.mark.asyncio
async def test_get_subject_not_found(async_client):
    resp = await async_client.get("/api/v1/subjects/999")
    assert resp.status_code == 404


@pytest.mark.asyncio
async def test_create_subject_admin(async_client, admin_token):
    resp = await async_client.post(
        "/api/v1/subjects/", json={"name": "Алгоритми"}, headers=auth_headers(admin_token)
    )
    assert resp.status_code == 201
    assert resp.json()["name"] == "Алгоритми"


@pytest.mark.asyncio
async def test_create_subject_student_forbidden(async_client, student_token):
    resp = await async_client.post(
        "/api/v1/subjects/", json={"name": "Операційні системи"}, headers=auth_headers(student_token)
    )
    assert resp.status_code == 403


@pytest.mark.asyncio
async def test_create_subject_unauthenticated(async_client):
    resp = await async_client.post("/api/v1/subjects/", json={"name": "Математика"})
    assert resp.status_code == 401


@pytest.mark.asyncio
async def test_update_subject_admin(async_client, admin_token):
    create_resp = await async_client.post(
        "/api/v1/subjects/", json={"name": "Фізика"}, headers=auth_headers(admin_token)
    )
    subject_id = create_resp.json()["id"]
    resp = await async_client.put(
        f"/api/v1/subjects/{subject_id}", json={"description": "Оновлений опис"}, headers=auth_headers(admin_token)
    )
    assert resp.status_code == 200
    assert resp.json()["description"] == "Оновлений опис"


@pytest.mark.asyncio
async def test_delete_subject_admin(async_client, admin_token):
    create_resp = await async_client.post(
        "/api/v1/subjects/", json={"name": "Хімія"}, headers=auth_headers(admin_token)
    )
    subject_id = create_resp.json()["id"]
    resp = await async_client.delete(f"/api/v1/subjects/{subject_id}", headers=auth_headers(admin_token))
    assert resp.status_code == 204

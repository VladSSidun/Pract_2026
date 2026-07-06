import pytest

from tests.conftest import auth_headers


@pytest.mark.asyncio
async def test_list_groups(async_client, admin_token):
    await async_client.post("/api/v1/groups/", json={"name": "ІПЗ-31"}, headers=auth_headers(admin_token))
    resp = await async_client.get("/api/v1/groups/")
    assert resp.status_code == 200
    assert len(resp.json()) == 1


@pytest.mark.asyncio
async def test_get_group_by_id(async_client, admin_token):
    create_resp = await async_client.post("/api/v1/groups/", json={"name": "ІПЗ-32"}, headers=auth_headers(admin_token))
    group_id = create_resp.json()["id"]
    resp = await async_client.get(f"/api/v1/groups/{group_id}")
    assert resp.status_code == 200
    assert resp.json()["name"] == "ІПЗ-32"


@pytest.mark.asyncio
async def test_get_group_not_found(async_client):
    resp = await async_client.get("/api/v1/groups/999")
    assert resp.status_code == 404


@pytest.mark.asyncio
async def test_create_group_admin(async_client, admin_token):
    resp = await async_client.post("/api/v1/groups/", json={"name": "ІПЗ-41"}, headers=auth_headers(admin_token))
    assert resp.status_code == 201
    assert resp.json()["name"] == "ІПЗ-41"


@pytest.mark.asyncio
async def test_create_group_student_forbidden(async_client, student_token):
    resp = await async_client.post("/api/v1/groups/", json={"name": "ІПЗ-42"}, headers=auth_headers(student_token))
    assert resp.status_code == 403


@pytest.mark.asyncio
async def test_create_group_unauthenticated(async_client):
    resp = await async_client.post("/api/v1/groups/", json={"name": "ІПЗ-43"})
    assert resp.status_code == 401


@pytest.mark.asyncio
async def test_update_group_admin(async_client, admin_token):
    create_resp = await async_client.post("/api/v1/groups/", json={"name": "ІПЗ-51"}, headers=auth_headers(admin_token))
    group_id = create_resp.json()["id"]
    resp = await async_client.put(
        f"/api/v1/groups/{group_id}", json={"name": "ІПЗ-51-оновлено"}, headers=auth_headers(admin_token)
    )
    assert resp.status_code == 200
    assert resp.json()["name"] == "ІПЗ-51-оновлено"


@pytest.mark.asyncio
async def test_delete_group_admin(async_client, admin_token):
    create_resp = await async_client.post("/api/v1/groups/", json={"name": "ІПЗ-61"}, headers=auth_headers(admin_token))
    group_id = create_resp.json()["id"]
    resp = await async_client.delete(f"/api/v1/groups/{group_id}", headers=auth_headers(admin_token))
    assert resp.status_code == 204

    get_resp = await async_client.get(f"/api/v1/groups/{group_id}")
    assert get_resp.status_code == 404

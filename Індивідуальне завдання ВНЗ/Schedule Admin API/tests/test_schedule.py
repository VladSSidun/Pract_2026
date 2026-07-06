import pytest

from tests.conftest import auth_headers


async def _create_group(client, token, name="ІПЗ-31"):
    resp = await client.post("/api/v1/groups/", json={"name": name}, headers=auth_headers(token))
    return resp.json()["id"]


async def _create_subject(client, token, name="Бази даних"):
    resp = await client.post("/api/v1/subjects/", json={"name": name}, headers=auth_headers(token))
    return resp.json()["id"]


async def _create_teacher(client, token, first_name="Олена", last_name="Коваленко"):
    resp = await client.post(
        "/api/v1/teachers/",
        json={"first_name": first_name, "last_name": last_name},
        headers=auth_headers(token),
    )
    return resp.json()["id"]


async def _base_entities(client, token):
    group_id = await _create_group(client, token)
    subject_id = await _create_subject(client, token)
    teacher_id = await _create_teacher(client, token)
    return group_id, subject_id, teacher_id


@pytest.mark.asyncio
async def test_list_schedule(async_client, admin_token):
    group_id, subject_id, teacher_id = await _base_entities(async_client, admin_token)
    await async_client.post(
        "/api/v1/schedule/",
        json={
            "subject_id": subject_id,
            "teacher_id": teacher_id,
            "group_id": group_id,
            "day_of_week": 1,
            "time_slot": "1",
            "room": "101",
        },
        headers=auth_headers(admin_token),
    )
    resp = await async_client.get("/api/v1/schedule/")
    assert resp.status_code == 200
    body = resp.json()
    assert len(body) == 1
    assert body[0]["subject"]["id"] == subject_id
    assert body[0]["teacher"]["id"] == teacher_id
    assert body[0]["group"]["id"] == group_id


@pytest.mark.asyncio
async def test_get_schedule_by_id(async_client, admin_token):
    group_id, subject_id, teacher_id = await _base_entities(async_client, admin_token)
    create_resp = await async_client.post(
        "/api/v1/schedule/",
        json={
            "subject_id": subject_id,
            "teacher_id": teacher_id,
            "group_id": group_id,
            "day_of_week": 2,
            "time_slot": "1",
            "room": "102",
        },
        headers=auth_headers(admin_token),
    )
    schedule_id = create_resp.json()["id"]
    resp = await async_client.get(f"/api/v1/schedule/{schedule_id}")
    assert resp.status_code == 200
    assert resp.json()["room"] == "102"


@pytest.mark.asyncio
async def test_get_schedule_not_found(async_client):
    resp = await async_client.get("/api/v1/schedule/999")
    assert resp.status_code == 404


@pytest.mark.asyncio
async def test_create_schedule_admin(async_client, admin_token):
    group_id, subject_id, teacher_id = await _base_entities(async_client, admin_token)
    resp = await async_client.post(
        "/api/v1/schedule/",
        json={
            "subject_id": subject_id,
            "teacher_id": teacher_id,
            "group_id": group_id,
            "day_of_week": 3,
            "time_slot": "1",
            "room": "103",
        },
        headers=auth_headers(admin_token),
    )
    assert resp.status_code == 201


@pytest.mark.asyncio
async def test_create_schedule_student_forbidden(async_client, admin_token, student_token):
    group_id, subject_id, teacher_id = await _base_entities(async_client, admin_token)
    resp = await async_client.post(
        "/api/v1/schedule/",
        json={
            "subject_id": subject_id,
            "teacher_id": teacher_id,
            "group_id": group_id,
            "day_of_week": 3,
            "time_slot": "1",
            "room": "104",
        },
        headers=auth_headers(student_token),
    )
    assert resp.status_code == 403


@pytest.mark.asyncio
async def test_create_schedule_unauthenticated(async_client, admin_token):
    group_id, subject_id, teacher_id = await _base_entities(async_client, admin_token)
    resp = await async_client.post(
        "/api/v1/schedule/",
        json={
            "subject_id": subject_id,
            "teacher_id": teacher_id,
            "group_id": group_id,
            "day_of_week": 3,
            "time_slot": "1",
            "room": "105",
        },
    )
    assert resp.status_code == 401


@pytest.mark.asyncio
async def test_update_schedule_admin(async_client, admin_token):
    group_id, subject_id, teacher_id = await _base_entities(async_client, admin_token)
    create_resp = await async_client.post(
        "/api/v1/schedule/",
        json={
            "subject_id": subject_id,
            "teacher_id": teacher_id,
            "group_id": group_id,
            "day_of_week": 4,
            "time_slot": "1",
            "room": "106",
        },
        headers=auth_headers(admin_token),
    )
    schedule_id = create_resp.json()["id"]
    resp = await async_client.put(
        f"/api/v1/schedule/{schedule_id}", json={"room": "201"}, headers=auth_headers(admin_token)
    )
    assert resp.status_code == 200
    assert resp.json()["room"] == "201"


@pytest.mark.asyncio
async def test_delete_schedule_admin(async_client, admin_token):
    group_id, subject_id, teacher_id = await _base_entities(async_client, admin_token)
    create_resp = await async_client.post(
        "/api/v1/schedule/",
        json={
            "subject_id": subject_id,
            "teacher_id": teacher_id,
            "group_id": group_id,
            "day_of_week": 5,
            "time_slot": "1",
            "room": "107",
        },
        headers=auth_headers(admin_token),
    )
    schedule_id = create_resp.json()["id"]
    resp = await async_client.delete(f"/api/v1/schedule/{schedule_id}", headers=auth_headers(admin_token))
    assert resp.status_code == 204


@pytest.mark.asyncio
async def test_schedule_conflict_room(async_client, admin_token):
    group_id, subject_id, teacher_id = await _base_entities(async_client, admin_token)
    other_teacher_id = await _create_teacher(async_client, admin_token, "Ігор", "Шевченко")

    await async_client.post(
        "/api/v1/schedule/",
        json={
            "subject_id": subject_id,
            "teacher_id": teacher_id,
            "group_id": group_id,
            "day_of_week": 1,
            "time_slot": "2",
            "room": "301",
        },
        headers=auth_headers(admin_token),
    )
    resp = await async_client.post(
        "/api/v1/schedule/",
        json={
            "subject_id": subject_id,
            "teacher_id": other_teacher_id,
            "group_id": group_id,
            "day_of_week": 1,
            "time_slot": "2",
            "room": "301",
        },
        headers=auth_headers(admin_token),
    )
    assert resp.status_code == 409


@pytest.mark.asyncio
async def test_schedule_conflict_teacher(async_client, admin_token):
    group_id, subject_id, teacher_id = await _base_entities(async_client, admin_token)
    other_group_id = await _create_group(async_client, admin_token, "ІПЗ-32")

    await async_client.post(
        "/api/v1/schedule/",
        json={
            "subject_id": subject_id,
            "teacher_id": teacher_id,
            "group_id": group_id,
            "day_of_week": 1,
            "time_slot": "3",
            "room": "302",
        },
        headers=auth_headers(admin_token),
    )
    resp = await async_client.post(
        "/api/v1/schedule/",
        json={
            "subject_id": subject_id,
            "teacher_id": teacher_id,
            "group_id": other_group_id,
            "day_of_week": 1,
            "time_slot": "3",
            "room": "303",
        },
        headers=auth_headers(admin_token),
    )
    assert resp.status_code == 409


@pytest.mark.asyncio
async def test_filter_by_group(async_client, admin_token):
    group_a = await _create_group(async_client, admin_token, "ІПЗ-31")
    group_b = await _create_group(async_client, admin_token, "ІПЗ-32")
    subject_id = await _create_subject(async_client, admin_token)
    teacher_id = await _create_teacher(async_client, admin_token)

    await async_client.post(
        "/api/v1/schedule/",
        json={
            "subject_id": subject_id, "teacher_id": teacher_id, "group_id": group_a,
            "day_of_week": 1, "time_slot": "1", "room": "401",
        },
        headers=auth_headers(admin_token),
    )
    await async_client.post(
        "/api/v1/schedule/",
        json={
            "subject_id": subject_id, "teacher_id": teacher_id, "group_id": group_b,
            "day_of_week": 1, "time_slot": "2", "room": "402",
        },
        headers=auth_headers(admin_token),
    )

    resp = await async_client.get(f"/api/v1/schedule/?group_id={group_a}")
    assert resp.status_code == 200
    body = resp.json()
    assert len(body) == 1
    assert body[0]["group"]["id"] == group_a

    group_resp = await async_client.get(f"/api/v1/schedule/group/{group_a}")
    assert group_resp.status_code == 200
    assert len(group_resp.json()) == 1


@pytest.mark.asyncio
async def test_filter_by_day(async_client, admin_token):
    group_id, subject_id, teacher_id = await _base_entities(async_client, admin_token)
    other_teacher_id = await _create_teacher(async_client, admin_token, "Ігор", "Шевченко")

    await async_client.post(
        "/api/v1/schedule/",
        json={
            "subject_id": subject_id, "teacher_id": teacher_id, "group_id": group_id,
            "day_of_week": 1, "time_slot": "1", "room": "501",
        },
        headers=auth_headers(admin_token),
    )
    await async_client.post(
        "/api/v1/schedule/",
        json={
            "subject_id": subject_id, "teacher_id": other_teacher_id, "group_id": group_id,
            "day_of_week": 2, "time_slot": "1", "room": "502",
        },
        headers=auth_headers(admin_token),
    )

    resp = await async_client.get("/api/v1/schedule/?day_of_week=1")
    assert resp.status_code == 200
    body = resp.json()
    assert len(body) == 1
    assert body[0]["day_of_week"] == 1


@pytest.mark.asyncio
async def test_seed(async_client, admin_token):
    resp = await async_client.post("/api/v1/schedule/seed", headers=auth_headers(admin_token))
    assert resp.status_code == 200
    assert resp.json()["message"] == "Seed data generated"

    resp2 = await async_client.post("/api/v1/schedule/seed", headers=auth_headers(admin_token))
    assert resp2.status_code == 200
    assert resp2.json()["message"] == "Already seeded"

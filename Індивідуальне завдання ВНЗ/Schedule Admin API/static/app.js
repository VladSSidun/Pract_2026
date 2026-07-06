'use strict';

/* =============================================
   CONSTANTS
   ============================================= */

const DAYS = ['', 'Понеділок', 'Вівторок', 'Середа', 'Четвер', 'П\'ятниця', 'Субота', 'Неділя'];

const TIME_SLOTS = {
  '1': '08:30 – 10:05',
  '2': '10:15 – 11:50',
  '3': '12:20 – 13:55',
  '4': '14:05 – 15:40',
  '5': '15:50 – 17:25',
  '6': '17:35 – 19:10',
};

/* =============================================
   STATE
   ============================================= */

const state = {
  token:   localStorage.getItem('token'),
  user:    null,
  groups:   [],
  subjects: [],
  teachers: [],
  scheduleFilters: { group_id: '', teacher_id: '', day_of_week: '' },
};

/* =============================================
   API UTILITY
   ============================================= */

async function apiFetch(method, path, body) {
  const headers = { 'Content-Type': 'application/json' };
  if (state.token) headers['Authorization'] = 'Bearer ' + state.token;

  const opts = { method, headers };
  if (body !== undefined) opts.body = JSON.stringify(body);

  let resp;
  try {
    resp = await fetch(path, opts);
  } catch {
    showToast('Помилка мережі. Перевірте з\'єднання з сервером.', 'error');
    return null;
  }

  if (resp.status === 401) {
    logout();
    return null;
  }

  if (resp.status === 204) return true;

  let data;
  try { data = await resp.json(); } catch { data = {}; }

  if (!resp.ok) {
    let msg = data.detail || ('Помилка ' + resp.status);
    if (typeof msg !== 'string') msg = JSON.stringify(msg);
    showToast(msg, 'error');
    return null;
  }

  return data;
}

/* =============================================
   UTILITIES
   ============================================= */

function esc(v) {
  if (v === null || v === undefined) return '';
  return String(v)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function fmtDate(iso) {
  if (!iso) return '—';
  return new Date(iso).toLocaleDateString('uk-UA', { day: '2-digit', month: '2-digit', year: 'numeric' });
}

function dayLabel(n) { return DAYS[n] || ('День ' + n); }

function slotLabel(slot) {
  const time = TIME_SLOTS[slot];
  return time ? (slot + ' пара (' + time + ')') : ('Пара ' + slot);
}

function isAdmin() { return state.user && state.user.role === 'admin'; }

/* =============================================
   TOAST
   ============================================= */

function showToast(msg, type) {
  type = type || 'success';
  const el = document.createElement('div');
  el.className = 'toast toast-' + type;
  el.textContent = msg;
  document.getElementById('toast-container').appendChild(el);
  setTimeout(function() { el.remove(); }, 3800);
}

/* =============================================
   MODAL
   ============================================= */

function openModal(title, html) {
  document.getElementById('modal-title').textContent = title;
  document.getElementById('modal-body').innerHTML = html;
  document.getElementById('modal-overlay').classList.remove('hidden');
}

function closeModal() {
  document.getElementById('modal-overlay').classList.add('hidden');
  document.getElementById('modal-body').innerHTML = '';
}

function bindModalForm(onSubmit) {
  const form = document.querySelector('#modal-body form');
  if (!form) return;
  form.addEventListener('submit', function(e) {
    e.preventDefault();
    onSubmit(e);
  });
}

/* =============================================
   AUTH
   ============================================= */

async function doLogin(username, password) {
  const data = await apiFetch('POST', '/api/v1/auth/login', { username, password });
  if (!data) return;
  state.token = data.access_token;
  localStorage.setItem('token', state.token);
  await initApp();
}

async function doRegister(payload) {
  const data = await apiFetch('POST', '/api/v1/auth/register', payload);
  if (!data) return;
  state.token = data.access_token;
  localStorage.setItem('token', state.token);
  await initApp();
}

function logout() {
  state.token = null;
  state.user = null;
  state.groups = [];
  state.subjects = [];
  state.teachers = [];
  localStorage.removeItem('token');
  showAuthScreen();
}

async function loadCurrentUser() {
  const data = await apiFetch('GET', '/api/v1/auth/me');
  if (!data) return false;
  state.user = data;
  return true;
}

/* =============================================
   AUTH SCREEN
   ============================================= */

function showAuthScreen() {
  document.getElementById('auth-screen').hidden = false;
  document.getElementById('app-screen').hidden = true;
}

function showAppScreen() {
  document.getElementById('auth-screen').hidden = true;
  document.getElementById('app-screen').hidden = false;
}

function renderUserPanel() {
  const u = state.user;
  const initial = u.username.charAt(0).toUpperCase();
  const badgeClass = u.role === 'admin' ? 'badge-admin' : 'badge-student';
  const badgeLabel = u.role === 'admin' ? 'Адміністратор' : 'Студент';

  document.getElementById('user-panel').innerHTML =
    '<div class="user-avatar">' + esc(initial) + '</div>' +
    '<div class="user-name">' + esc(u.username) + '</div>' +
    '<div class="user-email" title="' + esc(u.email) + '">' + esc(u.email) + '</div>' +
    '<span class="badge ' + badgeClass + '">' + badgeLabel + '</span>';

  document.querySelectorAll('.admin-only').forEach(function(el) {
    if (u.role === 'admin') el.classList.remove('hidden');
    else el.classList.add('hidden');
  });
}

/* =============================================
   NAVIGATION
   ============================================= */

function showSection(name) {
  document.querySelectorAll('.section').forEach(function(s) { s.classList.remove('active'); });
  document.querySelectorAll('.nav-item').forEach(function(a) { a.classList.remove('active'); });

  const sec = document.getElementById('section-' + name);
  if (sec) sec.classList.add('active');

  const nav = document.querySelector('.nav-item[data-section="' + name + '"]');
  if (nav) nav.classList.add('active');

  switch (name) {
    case 'schedule': loadSchedule(); break;
    case 'groups':   loadGroups();   break;
    case 'subjects': loadSubjects(); break;
    case 'teachers': loadTeachers(); break;
    case 'admin':    loadAdminSection(); break;
  }
}

/* =============================================
   GROUPS
   ============================================= */

async function loadGroups() {
  const c = document.getElementById('groups-list');
  c.innerHTML = '<div class="loading">Завантаження...</div>';

  const groups = await apiFetch('GET', '/api/v1/groups/');
  if (!groups) return;
  state.groups = groups;
  refreshGroupFilter();

  if (!groups.length) {
    c.innerHTML = '<div class="empty-state"><span class="empty-icon">&#128101;</span><p>Груп ще немає. Адміністратор може додати нові групи.</p></div>';
    return;
  }

  const admin = isAdmin();
  let rows = '';
  groups.forEach(function(g) {
    rows +=
      '<tr>' +
        '<td class="text-muted">' + esc(g.id) + '</td>' +
        '<td class="fw-semibold">' + esc(g.name) + '</td>' +
        '<td class="text-muted">' + fmtDate(g.created_at) + '</td>' +
        '<td><div class="actions">' +
          '<button class="btn btn-secondary btn-sm" onclick="viewGroupSchedule(' + g.id + ',\'' + esc(g.name) + '\')">Розклад</button>' +
          (admin
            ? '<button class="btn btn-secondary btn-sm" onclick="openEditGroup(' + g.id + ')">Редагувати</button>' +
              '<button class="btn btn-danger btn-sm" onclick="deleteGroup(' + g.id + ')">Видалити</button>'
            : '') +
        '</div></td>' +
      '</tr>';
  });

  c.innerHTML =
    '<table><thead><tr>' +
      '<th>#</th><th>Назва групи</th><th>Дата створення</th><th>Дії</th>' +
    '</tr></thead><tbody>' + rows + '</tbody></table>';
}

function refreshGroupFilter() {
  const sel = document.getElementById('filter-group');
  const cur = sel.value;
  sel.innerHTML = '<option value="">Всі групи</option>';
  state.groups.forEach(function(g) {
    const opt = document.createElement('option');
    opt.value = g.id;
    opt.textContent = g.name;
    if (String(g.id) === String(cur)) opt.selected = true;
    sel.appendChild(opt);
  });
}

function openCreateGroup() {
  openModal('Додати групу',
    '<form>' +
      '<div class="form-group"><label>Назва групи *</label>' +
        '<input type="text" id="gf-name" placeholder="Наприклад: ІПЗ-31" required></div>' +
      '<div class="modal-footer">' +
        '<button type="button" class="btn btn-secondary" onclick="closeModal()">Скасувати</button>' +
        '<button type="submit" class="btn btn-primary">Створити</button>' +
      '</div>' +
    '</form>');

  bindModalForm(async function() {
    const name = document.getElementById('gf-name').value.trim();
    if (!name) return;
    const res = await apiFetch('POST', '/api/v1/groups/', { name });
    if (!res) return;
    showToast('Групу "' + name + '" створено', 'success');
    closeModal();
    await loadGroups();
  });
}

async function openEditGroup(id) {
  const g = await apiFetch('GET', '/api/v1/groups/' + id);
  if (!g) return;

  openModal('Редагувати групу',
    '<form>' +
      '<div class="form-group"><label>Назва групи *</label>' +
        '<input type="text" id="gf-name" value="' + esc(g.name) + '" required></div>' +
      '<div class="modal-footer">' +
        '<button type="button" class="btn btn-secondary" onclick="closeModal()">Скасувати</button>' +
        '<button type="submit" class="btn btn-primary">Зберегти</button>' +
      '</div>' +
    '</form>');

  bindModalForm(async function() {
    const name = document.getElementById('gf-name').value.trim();
    if (!name) return;
    const res = await apiFetch('PUT', '/api/v1/groups/' + id, { name });
    if (!res) return;
    showToast('Групу оновлено', 'success');
    closeModal();
    await loadGroups();
  });
}

async function deleteGroup(id) {
  const g = state.groups.find(function(x) { return x.id === id; });
  if (!confirm('Видалити групу "' + (g ? g.name : id) + '"?\nЦе також видалить усі пов\'язані записи розкладу.')) return;
  const res = await apiFetch('DELETE', '/api/v1/groups/' + id);
  if (res === null) return;
  showToast('Групу видалено', 'success');
  await loadGroups();
}

async function viewGroupSchedule(groupId, groupName) {
  const items = await apiFetch('GET', '/api/v1/schedule/group/' + groupId);
  if (!items) return;
  showScheduleModal('Розклад групи: ' + groupName, items);
}

/* =============================================
   SUBJECTS
   ============================================= */

async function loadSubjects() {
  const c = document.getElementById('subjects-list');
  c.innerHTML = '<div class="loading">Завантаження...</div>';

  const subjects = await apiFetch('GET', '/api/v1/subjects/');
  if (!subjects) return;
  state.subjects = subjects;

  if (!subjects.length) {
    c.innerHTML = '<div class="empty-state"><span class="empty-icon">&#128218;</span><p>Предметів ще немає.</p></div>';
    return;
  }

  const admin = isAdmin();
  let rows = '';
  subjects.forEach(function(s) {
    rows +=
      '<tr>' +
        '<td class="text-muted">' + esc(s.id) + '</td>' +
        '<td class="fw-semibold">' + esc(s.name) + '</td>' +
        '<td class="text-muted">' + (s.description ? esc(s.description) : '—') + '</td>' +
        '<td class="text-muted">' + fmtDate(s.created_at) + '</td>' +
        '<td><div class="actions">' +
          (admin
            ? '<button class="btn btn-secondary btn-sm" onclick="openEditSubject(' + s.id + ')">Редагувати</button>' +
              '<button class="btn btn-danger btn-sm" onclick="deleteSubject(' + s.id + ')">Видалити</button>'
            : '<span class="text-muted">—</span>') +
        '</div></td>' +
      '</tr>';
  });

  c.innerHTML =
    '<table><thead><tr>' +
      '<th>#</th><th>Назва предмета</th><th>Опис</th><th>Дата створення</th><th>Дії</th>' +
    '</tr></thead><tbody>' + rows + '</tbody></table>';
}

function openCreateSubject() {
  openModal('Додати предмет',
    '<form>' +
      '<div class="form-group"><label>Назва предмета *</label>' +
        '<input type="text" id="sf-name" placeholder="Наприклад: Бази даних" required></div>' +
      '<div class="form-group"><label>Опис</label>' +
        '<textarea id="sf-desc" placeholder="Короткий опис курсу"></textarea></div>' +
      '<div class="modal-footer">' +
        '<button type="button" class="btn btn-secondary" onclick="closeModal()">Скасувати</button>' +
        '<button type="submit" class="btn btn-primary">Створити</button>' +
      '</div>' +
    '</form>');

  bindModalForm(async function() {
    const name = document.getElementById('sf-name').value.trim();
    const description = document.getElementById('sf-desc').value.trim() || null;
    if (!name) return;
    const res = await apiFetch('POST', '/api/v1/subjects/', { name, description });
    if (!res) return;
    showToast('Предмет "' + name + '" створено', 'success');
    closeModal();
    await loadSubjects();
  });
}

async function openEditSubject(id) {
  const s = await apiFetch('GET', '/api/v1/subjects/' + id);
  if (!s) return;

  openModal('Редагувати предмет',
    '<form>' +
      '<div class="form-group"><label>Назва предмета *</label>' +
        '<input type="text" id="sf-name" value="' + esc(s.name) + '" required></div>' +
      '<div class="form-group"><label>Опис</label>' +
        '<textarea id="sf-desc">' + esc(s.description || '') + '</textarea></div>' +
      '<div class="modal-footer">' +
        '<button type="button" class="btn btn-secondary" onclick="closeModal()">Скасувати</button>' +
        '<button type="submit" class="btn btn-primary">Зберегти</button>' +
      '</div>' +
    '</form>');

  bindModalForm(async function() {
    const name = document.getElementById('sf-name').value.trim();
    const description = document.getElementById('sf-desc').value.trim() || null;
    if (!name) return;
    const res = await apiFetch('PUT', '/api/v1/subjects/' + id, { name, description });
    if (!res) return;
    showToast('Предмет оновлено', 'success');
    closeModal();
    await loadSubjects();
  });
}

async function deleteSubject(id) {
  const s = state.subjects.find(function(x) { return x.id === id; });
  if (!confirm('Видалити предмет "' + (s ? s.name : id) + '"?\nЦе також видалить усі пов\'язані записи розкладу.')) return;
  const res = await apiFetch('DELETE', '/api/v1/subjects/' + id);
  if (res === null) return;
  showToast('Предмет видалено', 'success');
  await loadSubjects();
}

/* =============================================
   TEACHERS
   ============================================= */

async function loadTeachers() {
  const c = document.getElementById('teachers-list');
  c.innerHTML = '<div class="loading">Завантаження...</div>';

  const teachers = await apiFetch('GET', '/api/v1/teachers/');
  if (!teachers) return;
  state.teachers = teachers;
  refreshTeacherFilter();

  if (!teachers.length) {
    c.innerHTML = '<div class="empty-state"><span class="empty-icon">&#127891;</span><p>Викладачів ще немає.</p></div>';
    return;
  }

  const admin = isAdmin();
  let rows = '';
  teachers.forEach(function(t) {
    rows +=
      '<tr>' +
        '<td class="text-muted">' + esc(t.id) + '</td>' +
        '<td class="fw-semibold">' + esc(t.last_name) + ' ' + esc(t.first_name) + '</td>' +
        '<td class="text-muted">' + (t.email ? esc(t.email) : '—') + '</td>' +
        '<td class="text-muted">' + (t.department ? esc(t.department) : '—') + '</td>' +
        '<td><div class="actions">' +
          '<button class="btn btn-secondary btn-sm" onclick="viewTeacherSchedule(' + t.id + ',\'' + esc(t.last_name) + ' ' + esc(t.first_name) + '\')">Розклад</button>' +
          (admin
            ? '<button class="btn btn-secondary btn-sm" onclick="openEditTeacher(' + t.id + ')">Редагувати</button>' +
              '<button class="btn btn-danger btn-sm" onclick="deleteTeacher(' + t.id + ')">Видалити</button>'
            : '') +
        '</div></td>' +
      '</tr>';
  });

  c.innerHTML =
    '<table><thead><tr>' +
      '<th>#</th><th>ПІБ</th><th>Email</th><th>Кафедра</th><th>Дії</th>' +
    '</tr></thead><tbody>' + rows + '</tbody></table>';
}

function refreshTeacherFilter() {
  const sel = document.getElementById('filter-teacher');
  const cur = sel.value;
  sel.innerHTML = '<option value="">Всі викладачі</option>';
  state.teachers.forEach(function(t) {
    const opt = document.createElement('option');
    opt.value = t.id;
    opt.textContent = t.last_name + ' ' + t.first_name;
    if (String(t.id) === String(cur)) opt.selected = true;
    sel.appendChild(opt);
  });
}

function openCreateTeacher() {
  openModal('Додати викладача',
    '<form>' +
      '<div class="form-row">' +
        '<div class="form-group"><label>Ім\'я *</label><input type="text" id="tf-first" placeholder="Ім\'я" required></div>' +
        '<div class="form-group"><label>Прізвище *</label><input type="text" id="tf-last" placeholder="Прізвище" required></div>' +
      '</div>' +
      '<div class="form-group"><label>Email</label><input type="email" id="tf-email" placeholder="teacher@university.edu"></div>' +
      '<div class="form-group"><label>Кафедра</label><input type="text" id="tf-dept" placeholder="Кафедра програмної інженерії"></div>' +
      '<div class="modal-footer">' +
        '<button type="button" class="btn btn-secondary" onclick="closeModal()">Скасувати</button>' +
        '<button type="submit" class="btn btn-primary">Створити</button>' +
      '</div>' +
    '</form>');

  bindModalForm(async function() {
    const first = document.getElementById('tf-first').value.trim();
    const last  = document.getElementById('tf-last').value.trim();
    const email = document.getElementById('tf-email').value.trim() || null;
    const dept  = document.getElementById('tf-dept').value.trim()  || null;
    if (!first || !last) { showToast('Ім\'я і прізвище обов\'язкові', 'error'); return; }
    const res = await apiFetch('POST', '/api/v1/teachers/', { first_name: first, last_name: last, email, department: dept });
    if (!res) return;
    showToast('Викладача "' + last + ' ' + first + '" додано', 'success');
    closeModal();
    await loadTeachers();
  });
}

async function openEditTeacher(id) {
  const t = await apiFetch('GET', '/api/v1/teachers/' + id);
  if (!t) return;

  openModal('Редагувати викладача',
    '<form>' +
      '<div class="form-row">' +
        '<div class="form-group"><label>Ім\'я *</label><input type="text" id="tf-first" value="' + esc(t.first_name) + '" required></div>' +
        '<div class="form-group"><label>Прізвище *</label><input type="text" id="tf-last" value="' + esc(t.last_name) + '" required></div>' +
      '</div>' +
      '<div class="form-group"><label>Email</label><input type="email" id="tf-email" value="' + esc(t.email || '') + '"></div>' +
      '<div class="form-group"><label>Кафедра</label><input type="text" id="tf-dept" value="' + esc(t.department || '') + '"></div>' +
      '<div class="modal-footer">' +
        '<button type="button" class="btn btn-secondary" onclick="closeModal()">Скасувати</button>' +
        '<button type="submit" class="btn btn-primary">Зберегти</button>' +
      '</div>' +
    '</form>');

  bindModalForm(async function() {
    const first = document.getElementById('tf-first').value.trim();
    const last  = document.getElementById('tf-last').value.trim();
    const email = document.getElementById('tf-email').value.trim() || null;
    const dept  = document.getElementById('tf-dept').value.trim()  || null;
    if (!first || !last) { showToast('Ім\'я і прізвище обов\'язкові', 'error'); return; }
    const res = await apiFetch('PUT', '/api/v1/teachers/' + id, { first_name: first, last_name: last, email, department: dept });
    if (!res) return;
    showToast('Викладача оновлено', 'success');
    closeModal();
    await loadTeachers();
  });
}

async function deleteTeacher(id) {
  const t = state.teachers.find(function(x) { return x.id === id; });
  const name = t ? (t.last_name + ' ' + t.first_name) : id;
  if (!confirm('Видалити викладача "' + name + '"?\nЦе також видалить усі пов\'язані записи розкладу.')) return;
  const res = await apiFetch('DELETE', '/api/v1/teachers/' + id);
  if (res === null) return;
  showToast('Викладача видалено', 'success');
  await loadTeachers();
}

async function viewTeacherSchedule(teacherId, teacherName) {
  const items = await apiFetch('GET', '/api/v1/schedule/teacher/' + teacherId);
  if (!items) return;
  showScheduleModal('Розклад викладача: ' + teacherName, items);
}

/* =============================================
   SCHEDULE — SHARED MODAL VIEW
   ============================================= */

function buildScheduleRows(items) {
  if (!items.length) {
    return '<tr><td colspan="7" class="empty-state" style="padding:2rem;text-align:center;color:var(--text-muted)">Записів не знайдено</td></tr>';
  }
  return items.map(function(s) {
    return (
      '<tr>' +
        '<td class="day-cell">' + esc(dayLabel(s.day_of_week)) + '</td>' +
        '<td class="time-cell">' + esc(slotLabel(s.time_slot)) + '</td>' +
        '<td><span class="tag tag-gray room-cell">' + esc(s.room) + '</span></td>' +
        '<td>' + esc(s.subject.name) + '</td>' +
        '<td>' + esc(s.teacher.last_name) + ' ' + esc(s.teacher.first_name) + '</td>' +
        '<td><span class="tag tag-blue">' + esc(s.group.name) + '</span></td>' +
        '<td class="text-muted">' + (s.notes ? esc(s.notes) : '—') + '</td>' +
      '</tr>'
    );
  }).join('');
}

function showScheduleModal(title, items) {
  const html =
    '<div class="modal-schedule-table">' +
      '<table><thead><tr>' +
        '<th>День</th><th>Пара / час</th><th>Аудиторія</th><th>Предмет</th><th>Викладач</th><th>Група</th><th>Нотатки</th>' +
      '</tr></thead><tbody>' +
        buildScheduleRows(items) +
      '</tbody></table>' +
    '</div>' +
    '<div class="modal-footer"><button class="btn btn-secondary" onclick="closeModal()">Закрити</button></div>';
  openModal(title, html);
}

/* =============================================
   SCHEDULE — MAIN SECTION
   ============================================= */

async function loadSchedule() {
  const c = document.getElementById('schedule-list');
  c.innerHTML = '<div class="loading">Завантаження...</div>';

  const p = new URLSearchParams();
  if (state.scheduleFilters.group_id)   p.set('group_id',   state.scheduleFilters.group_id);
  if (state.scheduleFilters.teacher_id) p.set('teacher_id', state.scheduleFilters.teacher_id);
  if (state.scheduleFilters.day_of_week) p.set('day_of_week', state.scheduleFilters.day_of_week);
  const qs = p.toString() ? '?' + p.toString() : '';

  const items = await apiFetch('GET', '/api/v1/schedule/' + qs);
  if (!items) return;

  if (!items.length) {
    c.innerHTML = '<div class="empty-state"><span class="empty-icon">&#128197;</span><p>Записів розкладу не знайдено. Спробуйте змінити фільтри або додайте нові записи.</p></div>';
    return;
  }

  const admin = isAdmin();
  const actionHeader = admin ? '<th>Дії</th>' : '';

  let rows = items.map(function(s) {
    const actionCell = admin
      ? '<td><div class="actions">' +
          '<button class="btn btn-secondary btn-sm" onclick="openEditSchedule(' + s.id + ')">Ред.</button>' +
          '<button class="btn btn-danger btn-sm" onclick="deleteSchedule(' + s.id + ')">Вид.</button>' +
        '</div></td>'
      : '';
    return (
      '<tr>' +
        '<td class="day-cell">' + esc(dayLabel(s.day_of_week)) + '</td>' +
        '<td class="time-cell">' + esc(slotLabel(s.time_slot)) + '</td>' +
        '<td><span class="tag tag-gray room-cell">' + esc(s.room) + '</span></td>' +
        '<td>' + esc(s.subject.name) + '</td>' +
        '<td>' + esc(s.teacher.last_name) + ' ' + esc(s.teacher.first_name) + '</td>' +
        '<td><span class="tag tag-blue">' + esc(s.group.name) + '</span></td>' +
        '<td class="text-muted">' + s.max_students + '</td>' +
        '<td class="text-muted">' + (s.notes ? esc(s.notes) : '—') + '</td>' +
        actionCell +
      '</tr>'
    );
  }).join('');

  c.innerHTML =
    '<table><thead><tr>' +
      '<th>День</th><th>Пара / час</th><th>Аудиторія</th><th>Предмет</th>' +
      '<th>Викладач</th><th>Група</th><th>Макс.</th><th>Нотатки</th>' +
      actionHeader +
    '</tr></thead><tbody>' + rows + '</tbody></table>';
}

/* =============================================
   SCHEDULE FORM (shared create / edit)
   ============================================= */

async function ensureRefData() {
  if (!state.groups.length) {
    const g = await apiFetch('GET', '/api/v1/groups/');
    if (g) state.groups = g;
  }
  if (!state.subjects.length) {
    const s = await apiFetch('GET', '/api/v1/subjects/');
    if (s) state.subjects = s;
  }
  if (!state.teachers.length) {
    const t = await apiFetch('GET', '/api/v1/teachers/');
    if (t) state.teachers = t;
  }
}

function buildScheduleForm(s) {
  function opts(arr, valFn, labelFn, selId) {
    return arr.map(function(x) {
      const v = valFn(x), l = labelFn(x);
      const sel = (s && selId(s) === v) ? ' selected' : '';
      return '<option value="' + v + '"' + sel + '>' + esc(l) + '</option>';
    }).join('');
  }

  const groupOpts   = opts(state.groups,   function(g) { return g.id; }, function(g) { return g.name; },                               function(sc) { return sc.group.id;   });
  const subjectOpts = opts(state.subjects, function(s2){ return s2.id;}, function(s2){ return s2.name; },                              function(sc) { return sc.subject.id; });
  const teacherOpts = opts(state.teachers, function(t) { return t.id; }, function(t) { return t.last_name + ' ' + t.first_name; }, function(sc) { return sc.teacher.id; });

  const dayOpts = DAYS.slice(1).map(function(d, i) {
    const n = i + 1;
    const sel = (s && s.day_of_week === n) ? ' selected' : '';
    return '<option value="' + n + '"' + sel + '>' + esc(d) + '</option>';
  }).join('');

  const slotOpts = ['1','2','3','4','5','6'].map(function(n) {
    const sel = (s && s.time_slot === n) ? ' selected' : '';
    return '<option value="' + n + '"' + sel + '>' + esc(slotLabel(n)) + '</option>';
  }).join('');

  return (
    '<form>' +
      '<div class="form-group"><label>Предмет *</label>' +
        '<select id="scf-subject" required><option value="">— Оберіть предмет —</option>' + subjectOpts + '</select></div>' +
      '<div class="form-group"><label>Викладач *</label>' +
        '<select id="scf-teacher" required><option value="">— Оберіть викладача —</option>' + teacherOpts + '</select></div>' +
      '<div class="form-group"><label>Група *</label>' +
        '<select id="scf-group" required><option value="">— Оберіть групу —</option>' + groupOpts + '</select></div>' +
      '<div class="form-row">' +
        '<div class="form-group"><label>День тижня *</label>' +
          '<select id="scf-day" required><option value="">— Оберіть день —</option>' + dayOpts + '</select></div>' +
        '<div class="form-group"><label>Номер пари *</label>' +
          '<select id="scf-slot" required><option value="">— Оберіть пару —</option>' + slotOpts + '</select></div>' +
      '</div>' +
      '<div class="form-row">' +
        '<div class="form-group"><label>Аудиторія *</label>' +
          '<input type="text" id="scf-room" value="' + esc((s && s.room) || '') + '" placeholder="101" required></div>' +
        '<div class="form-group"><label>Макс. студентів</label>' +
          '<input type="number" id="scf-max" value="' + ((s && s.max_students) || 30) + '" min="1" max="999"></div>' +
      '</div>' +
      '<div class="form-group"><label>Нотатки</label>' +
        '<textarea id="scf-notes" placeholder="Додаткова інформація...">' + esc((s && s.notes) || '') + '</textarea></div>' +
      '<div class="modal-footer">' +
        '<button type="button" class="btn btn-secondary" onclick="closeModal()">Скасувати</button>' +
        '<button type="submit" class="btn btn-primary">' + (s ? 'Зберегти' : 'Створити') + '</button>' +
      '</div>' +
    '</form>'
  );
}

function collectScheduleBody() {
  return {
    subject_id:   parseInt(document.getElementById('scf-subject').value) || null,
    teacher_id:   parseInt(document.getElementById('scf-teacher').value) || null,
    group_id:     parseInt(document.getElementById('scf-group').value)   || null,
    day_of_week:  parseInt(document.getElementById('scf-day').value)     || null,
    time_slot:    document.getElementById('scf-slot').value,
    room:         document.getElementById('scf-room').value.trim(),
    max_students: parseInt(document.getElementById('scf-max').value)     || 30,
    notes:        document.getElementById('scf-notes').value.trim() || null,
  };
}

async function openCreateSchedule() {
  await ensureRefData();
  if (!state.groups.length || !state.subjects.length || !state.teachers.length) {
    showToast('Спочатку додайте групи, предмети та викладачів', 'error');
    return;
  }
  openModal('Додати запис розкладу', buildScheduleForm(null));
  bindModalForm(async function() {
    const body = collectScheduleBody();
    if (!body.subject_id || !body.teacher_id || !body.group_id || !body.day_of_week || !body.time_slot || !body.room) {
      showToast('Заповніть усі обов\'язкові поля', 'error');
      return;
    }
    const res = await apiFetch('POST', '/api/v1/schedule/', body);
    if (!res) return;
    showToast('Запис розкладу створено', 'success');
    closeModal();
    await loadSchedule();
  });
}

async function openEditSchedule(id) {
  await ensureRefData();
  const s = await apiFetch('GET', '/api/v1/schedule/' + id);
  if (!s) return;
  openModal('Редагувати запис розкладу', buildScheduleForm(s));
  bindModalForm(async function() {
    const body = collectScheduleBody();
    const res = await apiFetch('PUT', '/api/v1/schedule/' + id, body);
    if (!res) return;
    showToast('Запис розкладу оновлено', 'success');
    closeModal();
    await loadSchedule();
  });
}

async function deleteSchedule(id) {
  if (!confirm('Видалити цей запис розкладу?')) return;
  const res = await apiFetch('DELETE', '/api/v1/schedule/' + id);
  if (res === null) return;
  showToast('Запис видалено', 'success');
  await loadSchedule();
}

/* =============================================
   ADMIN SECTION
   ============================================= */

function loadAdminSection() {
  const u = state.user;
  if (!u) return;
  const badgeClass = u.role === 'admin' ? 'badge-admin' : 'badge-student';
  const badgeLabel = u.role === 'admin' ? 'Адміністратор' : 'Студент';

  document.getElementById('admin-profile').innerHTML =
    '<div class="profile-info">' +
      '<div class="profile-row"><span class="profile-label">ID:</span><span class="text-muted">' + esc(u.id) + '</span></div>' +
      '<div class="profile-row"><span class="profile-label">Логін:</span><span>' + esc(u.username) + '</span></div>' +
      '<div class="profile-row"><span class="profile-label">Email:</span><span>' + esc(u.email) + '</span></div>' +
      '<div class="profile-row"><span class="profile-label">Роль:</span><span class="badge ' + badgeClass + '">' + badgeLabel + '</span></div>' +
      '<div class="profile-row"><span class="profile-label">Зареєстрований:</span><span class="text-muted">' + fmtDate(u.created_at) + '</span></div>' +
    '</div>';
}

async function seedData() {
  const btn = document.getElementById('seed-btn');
  const resultEl = document.getElementById('seed-result');

  btn.disabled = true;
  btn.textContent = 'Виконання...';

  const data = await apiFetch('POST', '/api/v1/schedule/seed');

  btn.disabled = false;
  btn.textContent = 'Згенерувати seed-дані';

  if (!data) return;

  if (data.message === 'Seed data generated') {
    resultEl.className = 'seed-result success';
    resultEl.textContent = '✓ Тестові дані успішно згенеровано! Групи, предмети, викладачі та записи розкладу додані до бази.';
  } else {
    resultEl.className = 'seed-result info';
    resultEl.textContent = 'ℹ База вже містить дані (Already seeded). Повторна генерація пропущена.';
  }

  // Refresh reference data after seed
  const [groups, subjects, teachers] = await Promise.all([
    apiFetch('GET', '/api/v1/groups/'),
    apiFetch('GET', '/api/v1/subjects/'),
    apiFetch('GET', '/api/v1/teachers/'),
  ]);
  if (groups)   { state.groups   = groups;   refreshGroupFilter();   }
  if (subjects)  state.subjects  = subjects;
  if (teachers) { state.teachers = teachers; refreshTeacherFilter(); }
}

/* =============================================
   INITIALIZATION
   ============================================= */

async function initApp() {
  const ok = await loadCurrentUser();
  if (!ok) { logout(); return; }

  showAppScreen();
  renderUserPanel();

  // Pre-load all reference data in parallel
  const [groups, subjects, teachers] = await Promise.all([
    apiFetch('GET', '/api/v1/groups/'),
    apiFetch('GET', '/api/v1/subjects/'),
    apiFetch('GET', '/api/v1/teachers/'),
  ]);
  if (groups)   { state.groups   = groups;   refreshGroupFilter();   }
  if (subjects)  state.subjects  = subjects;
  if (teachers) { state.teachers = teachers; refreshTeacherFilter(); }

  showSection('schedule');
}

/* =============================================
   EVENT LISTENERS
   ============================================= */

document.addEventListener('DOMContentLoaded', async function() {

  // --- Auth tabs ---
  document.querySelectorAll('.auth-tab').forEach(function(tab) {
    tab.addEventListener('click', function() {
      document.querySelectorAll('.auth-tab').forEach(function(t) { t.classList.remove('active'); });
      tab.classList.add('active');
      const which = tab.dataset.tab;
      document.getElementById('login-form').hidden    = (which !== 'login');
      document.getElementById('register-form').hidden = (which !== 'register');
    });
  });

  // --- Login form ---
  document.getElementById('login-form').addEventListener('submit', async function(e) {
    e.preventDefault();
    await doLogin(
      document.getElementById('login-username').value,
      document.getElementById('login-password').value
    );
  });

  // --- Register form ---
  document.getElementById('register-form').addEventListener('submit', async function(e) {
    e.preventDefault();
    await doRegister({
      username: document.getElementById('reg-username').value,
      email:    document.getElementById('reg-email').value,
      password: document.getElementById('reg-password').value,
      role:     document.getElementById('reg-role').value,
    });
  });

  // --- Sidebar navigation ---
  document.querySelectorAll('.nav-item').forEach(function(item) {
    item.addEventListener('click', function(e) {
      e.preventDefault();
      showSection(item.dataset.section);
    });
  });

  // --- Logout ---
  document.getElementById('logout-btn').addEventListener('click', logout);

  // --- Modal close ---
  document.getElementById('modal-close').addEventListener('click', closeModal);
  document.getElementById('modal-overlay').addEventListener('click', function(e) {
    if (e.target === this) closeModal();
  });
  document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape') closeModal();
  });

  // --- Schedule filters ---
  document.getElementById('filter-group').addEventListener('change', function(e) {
    state.scheduleFilters.group_id = e.target.value;
    loadSchedule();
  });
  document.getElementById('filter-teacher').addEventListener('change', function(e) {
    state.scheduleFilters.teacher_id = e.target.value;
    loadSchedule();
  });
  document.getElementById('filter-day').addEventListener('change', function(e) {
    state.scheduleFilters.day_of_week = e.target.value;
    loadSchedule();
  });
  document.getElementById('reset-filters-btn').addEventListener('click', function() {
    state.scheduleFilters = { group_id: '', teacher_id: '', day_of_week: '' };
    document.getElementById('filter-group').value   = '';
    document.getElementById('filter-teacher').value = '';
    document.getElementById('filter-day').value     = '';
    loadSchedule();
  });

  // --- Add buttons ---
  document.getElementById('add-schedule-btn').addEventListener('click', openCreateSchedule);
  document.getElementById('add-group-btn').addEventListener('click', openCreateGroup);
  document.getElementById('add-subject-btn').addEventListener('click', openCreateSubject);
  document.getElementById('add-teacher-btn').addEventListener('click', openCreateTeacher);
  document.getElementById('seed-btn').addEventListener('click', seedData);

  // --- Bootstrap ---
  if (state.token) {
    await initApp();
  } else {
    showAuthScreen();
  }
});

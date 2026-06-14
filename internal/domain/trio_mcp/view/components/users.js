(function () {
  'use strict';

  // ── Render table ────────────────────────────────────────────────────────────
  window.loadUsers = async function () {
    const tbody = document.getElementById('users-body');
    tbody.innerHTML = '<tr><td colspan="6" class="empty">Loading…</td></tr>';

    const { ok, body } = await api('/admin/users');
    if (!ok) {
      tbody.innerHTML = '<tr><td colspan="6" class="empty" style="color:var(--error);">Failed to load users.</td></tr>';
      return;
    }

    if (!body || body.length === 0) {
      tbody.innerHTML = '<tr><td colspan="6" class="empty">No users found.</td></tr>';
      return;
    }

    tbody.innerHTML = '';
    body.forEach(function (u) {
      const projects = (u.allowed_projects || []).join(', ') || '—';
      const tr = document.createElement('tr');
      tr.innerHTML = [
        '<td><strong>' + esc(u.username) + '</strong></td>',
        '<td>' + esc(u.display_name || '—') + '</td>',
        '<td><span class="badge ' + esc(u.role) + '">' + esc(u.role) + '</span></td>',
        '<td style="max-width:200px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;" title="' + esc(projects) + '"><code>' + esc(projects) + '</code></td>',
        '<td><span class="badge ' + (u.is_active ? 'active' : 'inactive') + '">' + (u.is_active ? 'active' : 'inactive') + '</span></td>',
        '<td><button class="btn ghost sm" onclick="editUser(' + JSON.stringify(u).replace(/"/g, '&quot;') + ')">Edit</button></td>',
      ].join('');
      tbody.appendChild(tr);
    });
  };

  // ── Edit modal ──────────────────────────────────────────────────────────────
  window.editUser = function (u) {
    document.getElementById('user-modal-title').textContent = 'Edit @' + u.username;
    document.getElementById('u-id').value       = u.id;
    document.getElementById('u-role').value     = u.role || 'user';
    document.getElementById('u-projects').value = (u.allowed_projects || []).join(', ');
    document.getElementById('user-form-error').classList.add('hidden');
    openModal('user-modal');
  };

  // ── Form submit ─────────────────────────────────────────────────────────────
  document.getElementById('user-form').addEventListener('submit', async function (e) {
    e.preventDefault();
    const err = document.getElementById('user-form-error');
    err.classList.add('hidden');

    const id       = document.getElementById('u-id').value;
    const role     = document.getElementById('u-role').value;
    const projects = document.getElementById('u-projects').value
      .split(',').map(function (p) { return p.trim(); }).filter(Boolean);

    const [roleRes, projRes] = await Promise.all([
      api('/admin/users/' + id + '/role',     { method: 'PUT', body: JSON.stringify({ role }) }),
      api('/admin/users/' + id + '/projects', { method: 'PUT', body: JSON.stringify({ projects }) }),
    ]);

    if (!roleRes.ok || !projRes.ok) {
      err.textContent = 'Save failed — check the API.';
      err.classList.remove('hidden');
      return;
    }

    toast('User updated');
    closeModal('user-modal');
    loadUsers();
  });

  function esc(str) {
    return String(str || '').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
  }
})();

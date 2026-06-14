(function () {
  'use strict';

  let editingId = null; // null = create, uuid = update

  // ── Render table ─────────────────────────────────────────────────────────────
  window.loadSources = async function () {
    const tbody = document.getElementById('sources-body');
    tbody.innerHTML = '<tr><td colspan="6" class="empty">Loading…</td></tr>';

    const { ok, body } = await api('/admin/sources');
    if (!ok) {
      tbody.innerHTML = '<tr><td colspan="6" class="empty" style="color:var(--error);">Failed to load sources.</td></tr>';
      return;
    }
    if (!body || body.length === 0) {
      tbody.innerHTML = '<tr><td colspan="6" class="empty">No sources registered. Add one above.</td></tr>';
      return;
    }

    tbody.innerHTML = '';
    body.forEach(function (s) {
      const active = s.is_active !== false;
      const tr = document.createElement('tr');
      tr.innerHTML = [
        '<td><strong>' + esc(s.name) + '</strong></td>',
        '<td><code>' + esc(s.platform_domain) + '</code></td>',
        '<td>' + esc(s.type || 'api') + '</td>',
        '<td style="max-width:180px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;" title="' + esc(s.base_url) + '">' + esc(s.base_url) + '</td>',
        '<td><span class="badge ' + (active ? 'active' : 'inactive') + '">' + (active ? 'active' : 'inactive') + '</span></td>',
        '<td><div class="actions">',
        '  <button class="btn ghost sm" onclick="editSource(' + JSON.stringify(s).replace(/"/g, '&quot;') + ')">Edit</button>',
        '  <button class="btn danger sm" onclick="deleteSource(\'' + s.id + '\',\'' + esc(s.name) + '\')">Delete</button>',
        '</div></td>',
      ].join('');
      tbody.appendChild(tr);
    });
  };

  // ── Add button ───────────────────────────────────────────────────────────────
  document.getElementById('add-source-btn').addEventListener('click', function () {
    editingId = null;
    resetForm();
    document.getElementById('source-modal-title').textContent = 'Add source';
    document.getElementById('source-form-submit').textContent = 'Create';
    openModal('source-modal');
  });

  // ── Edit ─────────────────────────────────────────────────────────────────────
  window.editSource = function (s) {
    editingId = s.id;
    resetForm();
    document.getElementById('source-modal-title').textContent = 'Edit source';
    document.getElementById('source-form-submit').textContent = 'Save';
    document.getElementById('s-name').value     = s.name        || '';
    document.getElementById('s-domain').value   = s.platform_domain || '';
    document.getElementById('s-desc').value     = s.description || '';
    document.getElementById('s-type').value     = s.type        || 'api';
    document.getElementById('s-base-url').value = s.base_url    || '';
    document.getElementById('s-swagger').value  = s.swagger_path || '/swagger/doc.json';
    document.getElementById('s-context').value  = s.context_path || '/mcp/context';
    openModal('source-modal');
  };

  // ── Delete ───────────────────────────────────────────────────────────────────
  window.deleteSource = async function (id, name) {
    const yes = await confirm('Delete source "' + name + '"? This cannot be undone.', 'Delete');
    if (!yes) return;
    const { ok } = await api('/admin/sources/' + id, { method: 'DELETE' });
    if (ok) {
      toast('Source deleted');
      loadSources();
      window.reloadMCP && reloadMCP();
    } else {
      toast('Delete failed', 'error');
    }
  };

  // ── Form submit ──────────────────────────────────────────────────────────────
  document.getElementById('source-form').addEventListener('submit', async function (e) {
    e.preventDefault();
    const errEl = document.getElementById('source-form-error');
    errEl.classList.add('hidden');

    const payload = {
      name:            document.getElementById('s-name').value.trim(),
      platform_domain: document.getElementById('s-domain').value.trim(),
      description:     document.getElementById('s-desc').value.trim(),
      type:            document.getElementById('s-type').value || 'api',
      base_url:        document.getElementById('s-base-url').value.trim(),
      swagger_path:    document.getElementById('s-swagger').value.trim() || '/swagger/doc.json',
      context_path:    document.getElementById('s-context').value.trim() || '/mcp/context',
    };

    const isEdit = !!editingId;
    const { ok, body } = await api(
      isEdit ? '/admin/sources/' + editingId : '/admin/sources',
      { method: isEdit ? 'PUT' : 'POST', body: JSON.stringify(payload) }
    );

    if (!ok) {
      errEl.textContent = (body && body.error) || 'Save failed';
      errEl.classList.remove('hidden');
      return;
    }

    toast(isEdit ? 'Source updated' : 'Source created');
    closeModal('source-modal');
    loadSources();
    window.reloadMCP && reloadMCP();
  });

  // ── Reset form ───────────────────────────────────────────────────────────────
  function resetForm() {
    document.getElementById('source-form').reset();
    document.getElementById('source-form-error').classList.add('hidden');
    document.getElementById('s-swagger').value = '/swagger/doc.json';
    document.getElementById('s-context').value = '/mcp/context';
  }

  function esc(str) {
    return String(str || '')
      .replace(/&/g, '&amp;').replace(/</g, '&lt;')
      .replace(/>/g, '&gt;').replace(/"/g, '&quot;');
  }
})();

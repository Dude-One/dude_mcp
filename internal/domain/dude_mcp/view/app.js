(function () {
  'use strict';

  // ── Shared API helper ──────────────────────────────────────────────────────
  window.api = async function (path, opts) {
    const res = await fetch(path, {
      credentials: 'include',
      headers: { 'Content-Type': 'application/json', ...(opts && opts.headers) },
      ...opts,
    });
    let body = null;
    try { body = await res.json(); } catch (_) {}
    return { ok: res.ok, status: res.status, body };
  };

  // ── Views ──────────────────────────────────────────────────────────────────
  const views = ['loading-view', 'setup-view', 'login-view', 'forbidden-view', 'main-view'];

  function show(id) {
    views.forEach(function (v) {
      document.getElementById(v).classList.toggle('hidden', v !== id);
    });
  }

  window.showLogin = function () { show('login-view'); };

  // ── Modal helpers ──────────────────────────────────────────────────────────
  window.openModal = function (id) {
    document.getElementById(id).classList.add('open');
  };
  window.closeModal = function (id) {
    document.getElementById(id).classList.remove('open');
  };

  document.querySelectorAll('.modal-overlay').forEach(function (overlay) {
    // Source modal stays open until Save or Cancel is explicitly clicked.
    if (overlay.id === 'source-modal') return;
    overlay.addEventListener('click', function (e) {
      if (e.target === overlay) overlay.classList.remove('open');
    });
  });
  document.querySelectorAll('[data-close]').forEach(function (btn) {
    btn.addEventListener('click', function () { closeModal(btn.dataset.close); });
  });

  // ── Confirm dialog ─────────────────────────────────────────────────────────
  window.confirm = function (message, label) {
    return new Promise(function (resolve) {
      var resolved   = false;
      var modal      = document.getElementById('confirm-modal');
      var ok         = document.getElementById('confirm-ok');
      var cancelBtns = modal.querySelectorAll('[data-close]');

      function settle(result) {
        if (resolved) return;
        resolved = true;
        ok.removeEventListener('click', onOk);
        modal.removeEventListener('click', onOverlay);
        cancelBtns.forEach(function (b) { b.removeEventListener('click', onCancel); });
        closeModal('confirm-modal');
        resolve(result);
      }

      function onOk()       { settle(true);  }
      function onCancel()   { settle(false); }
      function onOverlay(e) { if (e.target === modal) settle(false); }

      document.getElementById('confirm-message').textContent = message;
      ok.textContent = label || 'Confirm';
      ok.className   = label === 'Delete' ? 'btn danger' : 'btn primary';

      ok.addEventListener('click', onOk);
      cancelBtns.forEach(function (b) { b.addEventListener('click', onCancel); });
      modal.addEventListener('click', onOverlay);

      openModal('confirm-modal');
    });
  };

  // ── Toast ──────────────────────────────────────────────────────────────────
  window.toast = function (msg, type) {
    const t = document.createElement('div');
    t.textContent = msg;
    t.style.cssText = [
      'position:fixed;bottom:1.5rem;right:1.5rem;z-index:999',
      'padding:.65rem 1.1rem;border-radius:8px;font-size:.875rem',
      'background:' + (type === 'error' ? '#7f1d1d' : '#14532d'),
      'color:'      + (type === 'error' ? '#fca5a5' : '#86efac'),
      'border:1px solid ' + (type === 'error' ? '#991b1b' : '#166534'),
    ].join(';');
    document.body.appendChild(t);
    setTimeout(function () { t.remove(); }, 3000);
  };

  // ── Tab routing ────────────────────────────────────────────────────────────
  function activateTab(name) {
    document.querySelectorAll('.tab').forEach(function (b) {
      b.classList.toggle('active', b.dataset.tab === name);
    });
    document.querySelectorAll('.tab-panel').forEach(function (p) {
      p.classList.toggle('active', p.id === 'tab-' + name);
    });
    if (name === 'sources') window.loadSources && loadSources();
    if (name === 'users')   window.loadUsers   && loadUsers();
    if (name === 'health')  window.loadHealth  && loadHealth();
    location.hash = name;
  }

  document.querySelectorAll('.tab').forEach(function (btn) {
    btn.addEventListener('click', function () { activateTab(btn.dataset.tab); });
  });

  // ── MCP status + reload ────────────────────────────────────────────────────
  async function checkMCPStatus() {
    const dot  = document.getElementById('mcp-dot');
    const text = document.getElementById('mcp-status-text');
    const hdot = document.getElementById('health-mcp-dot');
    const htxt = document.getElementById('health-mcp-text');
    if (!dot) return;
    try {
      const { ok } = await api('/admin/mcp/status');
      dot.className    = ok ? 'dot ok'      : 'dot fail';
      text.textContent = ok ? 'MCP online'  : 'MCP offline';
      if (hdot) hdot.className   = ok ? 'dot ok'    : 'dot fail';
      if (htxt) htxt.textContent = ok ? '● Running' : '● Stopped';
      if (htxt) htxt.style.color = ok ? 'var(--success, #4ade80)' : 'var(--error, #f87171)';
    } catch (_) {
      dot.className    = 'dot fail';
      text.textContent = 'MCP offline';
      if (hdot) hdot.className   = 'dot fail';
      if (htxt) htxt.textContent = '● Stopped';
      if (htxt) htxt.style.color = 'var(--error, #f87171)';
    }
  }

  window.reloadMCP = async function () {
    const dot  = document.getElementById('mcp-dot');
    const text = document.getElementById('mcp-status-text');
    if (dot) { dot.className = 'dot checking'; text.textContent = 'Reloading…'; }
    const { ok, body } = await api('/admin/mcp/reload', { method: 'POST' });
    if (ok) {
      toast('MCP registry reloaded');
    } else {
      toast((body && body.error) || 'Reload failed', 'error');
    }
    checkMCPStatus();
  };

  document.getElementById('reload-mcp-btn').addEventListener('click', reloadMCP);

  // ── Logout ─────────────────────────────────────────────────────────────────
  document.getElementById('logout-btn').addEventListener('click', async function () {
    await api('/panel/logout', { method: 'POST' });
    show('login-view');
  });

  // ── Login form ─────────────────────────────────────────────────────────────
  document.getElementById('login-form').addEventListener('submit', async function (e) {
    e.preventDefault();
    const err  = document.getElementById('login-error');
    const username = document.getElementById('login-username').value.trim();
    const credential = document.getElementById('login-password').value;

    err.classList.add('hidden');

    const { ok, body } = await api('/panel/login', {
      method: 'POST',
      body: JSON.stringify({ username, credential }),
    });

    if (!ok) {
      err.textContent = (body && body.error) || 'Invalid credentials';
      err.classList.remove('hidden');
      return;
    }

    // Check role after login.
    await checkSession();
  });

  // ── Session check ──────────────────────────────────────────────────────────
  async function checkSession() {
    const { ok, body } = await api('/panel/me');

    if (!ok || !body) {
      show('login-view');
      return;
    }

    if (body.role !== 'admin') {
      show('forbidden-view');
      return;
    }

    show('main-view');

    // Route to hash or default.
    const tab = (location.hash || '#sources').replace('#', '');
    activateTab(['sources', 'users', 'health'].includes(tab) ? tab : 'sources');

    // Poll MCP status every 30s.
    checkMCPStatus();
    setInterval(checkMCPStatus, 30000);
  }

  // ── Setup form ─────────────────────────────────────────────────────────────
  document.getElementById('setup-form').addEventListener('submit', async function (e) {
    e.preventDefault();
    var err = document.getElementById('setup-error');
    err.classList.add('hidden');

    var username    = document.getElementById('setup-username').value.trim();
    var credential  = document.getElementById('setup-password').value;
    var displayName = document.getElementById('setup-display').value.trim();

    var res = await api('/panel/setup', {
      method: 'POST',
      body: JSON.stringify({ username, credential, display_name: displayName }),
    });

    if (!res.ok) {
      err.textContent = (res.body && res.body.error) || 'Setup failed';
      err.classList.remove('hidden');
      return;
    }

    await checkSession();
  });

  // ── Boot ───────────────────────────────────────────────────────────────────
  show('loading-view');

  (async function boot() {
    var setupRes = await api('/panel/setup/check');
    if (setupRes.ok && setupRes.body && setupRes.body.needs_setup) {
      show('setup-view');
      return;
    }
    checkSession();
  })();
})();

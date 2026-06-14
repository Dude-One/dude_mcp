(function () {
  'use strict';

  window.loadHealth = async function () {
    const tbody = document.getElementById('health-body');
    tbody.innerHTML = '<tr><td colspan="5" class="empty">Loading sources…</td></tr>';

    const { ok, body } = await api('/admin/sources');
    if (!ok || !body || body.length === 0) {
      tbody.innerHTML = '<tr><td colspan="5" class="empty">No sources to check.</td></tr>';
      return;
    }

    // Render rows with "checking" state first
    tbody.innerHTML = '';
    body.forEach(function (s) {
      const tr = document.createElement('tr');
      tr.id = 'health-row-' + s.id;
      tr.innerHTML = [
        '<td><strong>' + esc(s.name) + '</strong></td>',
        '<td><code>' + esc(s.platform_domain) + '</code></td>',
        '<td style="max-width:200px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;">' + esc(s.base_url) + '</td>',
        '<td id="health-status-' + s.id + '"><span class="dot checking"></span> Checking…</td>',
        '<td><button class="btn ghost sm" onclick="pingSource(' + JSON.stringify(s).replace(/"/g,'&quot;') + ')">Ping</button></td>',
      ].join('');
      tbody.appendChild(tr);
    });

    // Ping all concurrently
    body.forEach(function (s) { pingSource(s); });
  };

  window.pingSource = async function (s) {
    const cell = document.getElementById('health-status-' + s.id);
    if (!cell) return;
    cell.innerHTML = '<span class="dot checking"></span> Checking…';

    // Ping the swagger endpoint as the health probe
    const url = s.base_url + (s.swagger_path || '/swagger/doc.json');

    try {
      const start = Date.now();
      const res = await fetch(url, { method: 'GET', mode: 'cors', cache: 'no-cache', signal: AbortSignal.timeout(5000) });
      const ms  = Date.now() - start;
      if (res.ok) {
        cell.innerHTML = '<span class="dot ok"></span> Reachable <span style="color:var(--muted);font-size:.8rem;">(' + ms + ' ms)</span>';
      } else {
        cell.innerHTML = '<span class="dot fail"></span> HTTP ' + res.status;
      }
    } catch (err) {
      cell.innerHTML = '<span class="dot fail"></span> Unreachable';
    }
  };

  // Refresh all button
  document.getElementById('refresh-health-btn').addEventListener('click', function () {
    loadHealth();
  });


  function esc(str) {
    return String(str || '').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
  }
})();

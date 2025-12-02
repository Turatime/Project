const f = document.getElementById('f');
const statusEl = document.getElementById('status');
const btnIngest = document.getElementById('btnIngest');
const btnAnalyze = document.getElementById('btnAnalyze');

const infoView = document.getElementById('infoview');
const elTitle = document.getElementById('title');
const elSummary = document.getElementById('summary');
const elBullets = document.getElementById('bullets');
const elMetrics = document.getElementById('metrics');
let chart;

btnIngest.addEventListener('click', async (e) => {
  e.preventDefault();
  const fd = new FormData(f);
  const res = await fetch('/api/rai/ingest', { method: 'POST', body: fd });
  const js = await res.json();
  statusEl.textContent = js.ok ? 'บันทึกแล้ว' : 'มีปัญหา';
});

btnAnalyze.addEventListener('click', async () => {
  const fd = new FormData(f);
  const res = await fetch('/api/rai/analyze', { method:'POST', body: fd });
  if (!res.ok) { statusEl.textContent = 'วิเคราะห์ล้มเหลว'; return; }
  const data = await res.json();
  renderInfographic(data);
  infoView.style.display = 'block';
});

function renderInfographic(d) {
  elTitle.textContent = d.title || 'ผลลัพธ์';
  elSummary.textContent = d.summary || '';

  elBullets.innerHTML = '';
  (d.bullets || []).forEach(x => {
    const li = document.createElement('li');
    li.textContent = x;
    elBullets.appendChild(li);
  });

  elMetrics.innerHTML = '';
  (d.metrics || []).forEach(m => {
    const div = document.createElement('div');
    div.className = 'metric';
    div.innerHTML = `<div style="color:#6b7280">${m.name}</div><div style="font-size:1.6rem;font-weight:700">${m.value} ${m.unit||''}</div>`;
    elMetrics.appendChild(div);
  });

  const ctx = document.getElementById('chart');
  if (chart) chart.destroy();
  chart = new Chart(ctx, {
    type: 'doughnut',
    data: { labels: d.chart?.labels || [], datasets: [{ data: d.chart?.series || [] }] },
    options: { responsive: true, plugins: { legend: { position: 'bottom' } } }
  });
}

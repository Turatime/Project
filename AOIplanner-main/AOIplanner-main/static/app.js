let lastFieldId = null;

function out(id, data) {
  const el = document.getElementById(id);
  el.textContent = (typeof data === 'string') ? data : JSON.stringify(data, null, 2);
}

async function _devLogin() {
  const uid = document.getElementById('uid').value || 'U_DEV_DEFAULT';
  const res = await fetch(`/devlogin?uid=${encodeURIComponent(uid)}`).catch(e=>({ok:false, statusText:String(e)}));
  if (!res.ok) return out('authOut', `devlogin failed: ${res.status} ${res.statusText}`);
  const who = await fetch('/whoami');
  out('authOut', await who.json());
}

async function _whoAmI() {
  const res = await fetch('/whoami');
  out('authOut', await res.json());
}

async function _createField() {
  const payload = {
    variety: document.getElementById('variety').value,
    crop_type: document.getElementById('crop_type').value,
    area_rai: parseFloat(document.getElementById('area_rai').value || 0),
    province: document.getElementById('province').value,
    district: document.getElementById('district').value,
    soil_texture: document.getElementById('soil_texture').value,
    pump_m3h: parseFloat(document.getElementById('pump_m3h').value || 0),
    irrigation_src: document.getElementById('irrigation_src').value,
    budget_tier: document.getElementById('budget_tier').value,
    fert_base: document.getElementById('fert_base').value,
    planting_date: document.getElementById('planting_date').value
  };
  const res = await fetch('/fields', {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify(payload)
  });
  const json = await res.json();
  if (res.ok) { lastFieldId = json.field_id; }
  out('fieldOut', json);
}

// ===== NEW: inject plan.calendar into month view immediately =====
function injectCalendarFromPlan(planCalendar) {
  if (!planCalendar || typeof planCalendar !== 'object') return;
  calDataByDate = {};
  const dates = Object.keys(planCalendar).sort();
  for (const ds of dates) {
    const items = planCalendar[ds] || [];
    calDataByDate[ds] = items.map(t => ({ ...t, date: ds }));
  }
  // jump calendar to the month of first task
  if (dates.length) {
    const d = new Date(dates[0]);
    calYear = d.getFullYear();
    calMonth = d.getMonth();
  }
  document.getElementById('calMonthLabel').textContent =
    `${calYear}-${String(calMonth+1).padStart(2,'0')}`;
  renderCalendarGrid();
  pickFirstTaskDay();
}

// ===== Lock Delivery Date to last day of current plan =====
function lockDeliveryDateToLastPlan() {
  try {
    const dateInput = document.getElementById('d_date');
    if (!dateInput) return;

    const keys = Object.keys(calDataByDate || {}).sort(); // 'YYYY-MM-DD' ASC
    if (!keys.length) return;

    const last = keys[keys.length - 1];      // วันสุดท้ายของแผน
    dateInput.value = last;                  // ตั้งค่า
    dateInput.min = last;                    // ล็อกไม่ให้เลือกก่อนหน้า
    dateInput.max = last;                    // ล็อกไม่ให้เลือกหลังจากนั้น
    dateInput.readOnly = true;               // กันพิมพ์เอง (เสริมความแน่น)
    dateInput.title = 'Locked to last plan date';
  } catch (e) {}
}


// REPLACE _generatePlan WITH THIS VERSION
// ===== Generate Plan (ตอน 3 → แสดง success + แสดงปฏิทิน) =====
async function _generatePlan() {
  if (!lastFieldId) return out('planOut', 'Create a field first.');

  // ต้องกรอกช่วงวันก่อน
  const from = (document.getElementById('from')||{}).value;
  const to   = (document.getElementById('to')||{}).value;
  if (!from || !to) {
    out('planOut', 'Please fill in the date fields before clicking Generate Plan.');
    return;
  }

  // ตั้งเดือนของปฏิทินตาม From
  try { 
    const d = new Date(from); 
    if (!isNaN(d)) { calYear = d.getFullYear(); calMonth = d.getMonth(); } 
  } catch(e){}

  const res = await fetch(`/fields/${lastFieldId}/plan?format=calendar`, { method:'POST' });
  let json = null;
  try { json = await res.json(); } catch(e) {}

  if (res.ok) {
    out('planOut', ' Plan generated successfully');
  } else {
    out('planOut', ` Failed: ${res.status} ${res.statusText}`);
    return;
  }

  // inject calendar
  if (json && json.calendar) {
    calDataByDate = {};
    const dates = Object.keys(json.calendar).sort();
    for (const ds of dates) {
      const items = json.calendar[ds] || [];
      calDataByDate[ds] = items.map(t => ({ ...t, date: ds }));
    }
    if (dates.length) {
      const d = new Date(dates[0]);
      calYear = d.getFullYear();
      calMonth = d.getMonth();
    }
  } else {
    await calReload();
  }

  // ย้าย calendar section (ตอน 5) มาอยู่ใต้ตอน 3
  try {
    const calSection = document.getElementById('calSection');
    const calMount3 = document.getElementById('calMount3');
    if (calSection && calMount3 && !calMount3.contains(calSection)) {
      calMount3.appendChild(calSection);
    }
  } catch(e){}

   document.getElementById('calMonthLabel').textContent =
    `${calYear}-${String(calMonth+1).padStart(2,'0')}`;
  renderCalendarGrid();
  pickFirstTaskDay();

  // === เพิ่ม: แสดงวันสุดท้ายของแผน ===
  const keys = Object.keys(calDataByDate || {}).sort();
  if (keys.length) {
    const last = keys[keys.length - 1];
    out('planOut', `Plan generated successfully. Last day of plan: ${last}`);
  } else {
    out('planOut', 'Plan generated successfully. (No tasks found)');
  }


}

// ===== Get Schedule (ตอน 3 → JSON output) =====
async function _getSchedule() {
  if (!lastFieldId) return out('schedOut', 'Create a field first.');
  const from = document.getElementById('from').value;
  const to   = document.getElementById('to').value;
  if (!from || !to) {
    out('schedOut', 'Please fill in the date before.');
    return;
  }
  const q = new URLSearchParams();
  q.set('from', from);
  q.set('to', to);
  const res = await fetch(`/fields/${lastFieldId}/schedule?`+q.toString());
  out('schedOut', await res.json());
}


async function _logMeasurement() {
   if (!lastFieldId) return out('measOut', 'Create a field first.');
  const payload = {
    date: document.getElementById('m_date').value,
    cane_height_cm: safeFloat('m_height'),
    soil_moist_pct: safeFloat('m_moist'),
    moist_state: document.getElementById('m_state').value,
    rainfall_mm: safeFloat('m_rain'),
    pest_scale: safeInt('m_pest'),
    problem: document.getElementById('m_problem').value   // ← เพิ่มตรงนี้
  };
  const res = await fetch(`/fields/${lastFieldId}/measurements`, {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify(payload)
  });
  out('measOut', await res.json());
}


async function _replan() {
  if (!lastFieldId) return out('replanOut', 'Create a field first.');
  const res = await fetch(`/fields/${lastFieldId}/replan`, { method:'POST' });
  out('replanOut', await res.json());
}

function safeFloat(id) {
  const v = document.getElementById(id).value;
  const n = parseFloat(v);
  return isFinite(n) ? n : undefined;
}
function safeInt(id) {
  const v = document.getElementById(id).value;
  const n = parseInt(v,10);
  return isFinite(n) ? n : undefined;
}

// expose to global for inline onclick=""
window.devLogin = _devLogin;
window.whoAmI = _whoAmI;
window.createField = _createField;
window.generatePlan = _generatePlan;
window.getSchedule = _getSchedule;
window.logMeasurement = _logMeasurement;
window.replan = _replan;

// set default planting date & show whoami
document.addEventListener('DOMContentLoaded', async () => {
  const d = new Date();
  const y = d.getFullYear(), m = (''+(d.getMonth()+1)).padStart(2,'0'), da = (''+d.getDate()).padStart(2,'0');
  const today = `${y}-${m}-${da}`;
  const pd = document.getElementById('planting_date');
  if (pd && !pd.value) pd.value = today;
  try { const res = await fetch('/whoami'); out('authOut', await res.json()); } catch(e){}
});

// ===== Calendar state =====
let calYear, calMonth; // JS month: 0..11
let calDataByDate = {}; // { 'YYYY-MM-DD': [tasks...] }
let selectedDateStr = null;

window.calPrevMonth = function() {
  if (calMonth === 0) { calMonth = 11; calYear--; } else { calMonth--; }
  calReload();
};
window.calNextMonth = function() {
  if (calMonth === 11) { calMonth = 0; calYear++; } else { calMonth++; }
  calReload();
};
window.calToday = function() {
  const d = new Date();
  calYear = d.getFullYear();
  calMonth = d.getMonth();
  calReload();
};



window.calReload = async function() {
  if (!lastFieldId) { out('schedOut', 'Create a field and generate plan first.'); }
  document.getElementById('calMonthLabel').textContent = `${calYear}-${String(calMonth+1).padStart(2,'0')}`;
  await loadMonthSchedule();
  renderCalendarGrid();
  if (selectedDateStr) selectDay(selectedDateStr); else pickFirstTaskDay();
  setDeliveryDateToLastPlan(); // อัปเดตวันที่ส่งให้เป็นวันสุดท้ายอีกครั้ง
};

function pad(n){ return String(n).padStart(2,'0'); }
function ymd(y,m,d){ return `${y}-${String(m).padStart(2,'0')}-${String(d).padStart(2,'0')}`; }

async function loadMonthSchedule() {
  calDataByDate = {};
  // from first day to last day of calendar month
  const from = ymd(calYear, calMonth+1, 1);
  const lastDate = new Date(calYear, calMonth+1, 0).getDate();
  const to = ymd(calYear, calMonth+1, lastDate);
  const res = await fetch(`/fields/${lastFieldId}/schedule?from=${from}&to=${to}`);
  if (!res.ok) { return; }
  const tasks = await res.json();
  for (const t of tasks) {
    const d = (t.date || '').slice(0,10);
    if (!calDataByDate[d]) calDataByDate[d] = [];
    calDataByDate[d].push(t);
  }
}

function renderCalendarGrid() {
  const cal = document.getElementById('calendar');
  cal.innerHTML = '';

  const first = new Date(calYear, calMonth, 1);
  const startDay = first.getDay(); // 0..6
  const daysInMonth = new Date(calYear, calMonth+1, 0).getDate();

  const label = `${calYear}-${String(calMonth+1).padStart(2,'0')}`;
  document.getElementById('calMonthLabel').textContent = label;

  const wd = ['Sun','Mon','Tue','Wed','Thu','Fri','Sat'];

  let html = '<table class="cal"><thead><tr>';
  for (const d of wd) html += `<th>${d}</th>`;
  html += '</tr></thead><tbody><tr>';

  let cell = 0;
  // ช่องว่างก่อนวันที่ 1
  for (let i=0;i<startDay;i++){ html += '<td class="empty"></td>'; cell++; }

  const todayStr = ymd(new Date().getFullYear(), new Date().getMonth()+1, new Date().getDate());

  for (let d=1; d<=daysInMonth; d++){
    const ds = ymd(calYear, calMonth+1, d);
    const tasks = calDataByDate[ds] || [];
    const isToday = (ds === todayStr);

    html += `<td id="cal-${ds}" class="${isToday?'today':''}" onclick="pickDay('${ds}')">
      <div class="daynum">${d}</div>
      ${
        tasks.map(t=>`<div class="task-line"><span class="task-dot"></span>${t.title}</div>`).join('')
      }
    </td>`;

    cell++;
    if (cell % 7 === 0 && d<daysInMonth) html += '</tr><tr>';
  }

  // ช่องว่างท้ายเดือนให้ครบแถว
  while (cell % 7 !== 0){ html += '<td class="empty"></td>'; cell++; }

  html += '</tr></tbody></table>';
  cal.innerHTML = html;
}


  function renderCalendarGrid() {
  const cal = document.getElementById('calendar');

  // คำนวณจำนวนวันของเดือนที่กำลังแสดง
  const first = new Date(calYear, calMonth, 1);
  const startDay = first.getDay(); // 0..6
  const daysInMonth = new Date(calYear, calMonth + 1, 0).getDate();

  // เฮดเดอร์ตาราง
  const wd = ['Sun','Mon','Tue','Wed','Thu','Fri','Sat'];
  let html = '<table class="cal"><tr>';
  for (const d of wd) html += `<th>${d}</th>`;
  html += '</tr><tr>';

  // ช่องว่างก่อนวันที่ 1
  let cellCount = 0;
  for (let i = 0; i < startDay; i++) {
    html += '<td class="cal-cell empty"></td>';
    cellCount++;
  }

  // วาดวันในเดือน
  for (let d = 1; d <= daysInMonth; d++) {
    const dateStr = `${calYear}-${String(calMonth + 1).padStart(2,'0')}-${String(d).padStart(2,'0')}`;
    const tasks = calDataByDate[dateStr] || [];

    html += `<td class="cal-cell" id="cal-${dateStr}" onclick="pickDay('${dateStr}')">
               <div class="daynum">${d}</div>
               ${tasks.map(t => `<div class="small muted">${t.title}</div>`).join('')}
             </td>`;

    cellCount++;
    if (cellCount % 7 === 0 && d < daysInMonth) html += '</tr><tr>';
  }

  // เติมช่องว่างท้ายเดือนให้ครบสัปดาห์
  while (cellCount % 7 !== 0) {
    html += '<td class="cal-cell empty"></td>';
    cellCount++;
  }

  html += '</tr></table>';
  cal.innerHTML = html;
}


function pickDay(dateStr){
  // ล้าง highlight เก่า
  document.querySelectorAll('.cal td.selected').forEach(el => el.classList.remove('selected'));
  // ใส่ให้ cell ใหม่
  const cell = document.getElementById('cal-' + dateStr);
  if (cell) cell.classList.add('selected');

  // แสดงรายการงานด้านขวา
  const tasks = calDataByDate[dateStr] || [];
  document.getElementById('dayTitle').textContent = dateStr;
  document.getElementById('dayTasks').innerHTML =
    tasks.length
      ? tasks.map(t => `<div>• ${t.title}${t.qty?` – ${t.qty} ${t.unit||''}`:''}</div>`).join('')
      : '<div class="muted">No tasks</div>';
}



function pickFirstTaskDay() {
  // choose today if in month, else first date that has tasks
  const today = new Date();
  const inThisMonth = today.getFullYear() === calYear && today.getMonth() === calMonth;
  const dsToday = ymd(calYear, calMonth+1, today.getDate());
  if (inThisMonth && calDataByDate[dsToday]) { selectDay(dsToday); return; }
  const keys = Object.keys(calDataByDate).sort();
  if (keys.length) selectDay(keys[0]); else showDayTasks(null);
}

function selectDay(ds) {
  selectedDateStr = ds;
  showDayTasks(ds);
}

function showDayTasks(ds) {
  const title = document.getElementById('dayTitle');
  const pane = document.getElementById('dayTasks');
  pane.innerHTML = '';
  if (!ds) {
    title.textContent = 'Selected Day';
    return;
  }
  title.textContent = `Tasks on ${ds}`;
  const tasks = calDataByDate[ds] || [];
  if (!tasks.length) {
    pane.textContent = 'No tasks';
    return;
  }
  // list
  for (const t of tasks) {
    const row = document.createElement('div');
    row.style.border = '1px solid #ddd';
    row.style.borderRadius = '8px';
    row.style.padding = '8px';
    row.style.marginBottom = '8px';
    const chk = document.createElement('input');
    chk.type = 'checkbox';
    chk.checked = (t.status === 'done');
    chk.addEventListener('change', async () => {
      await fetch(`/schedule/${t.task_id}`, {
        method: 'PATCH',
        headers: {'Content-Type':'application/json'},
        body: JSON.stringify({ status: chk.checked ? 'done' : 'todo' })
      });
      // reflect immediately
      t.status = chk.checked ? 'done' : 'todo';
    });
    const title = document.createElement('div');
    title.textContent = `${t.type.toUpperCase()}: ${t.title}`;
    const meta = document.createElement('div');
    meta.className = 'muted';
    const qty = (t.qty != null) ? `${Number(t.qty).toFixed(1)} ${t.unit || ''}` : '';
    meta.textContent = [qty, t.notes].filter(Boolean).join(' • ');
    row.appendChild(chk);
    row.appendChild(document.createTextNode(' '));
    row.appendChild(title);
    row.appendChild(meta);
    pane.appendChild(row);
  }
}

// initialize calendar after page load & whoami check
document.addEventListener('DOMContentLoaded', () => {
  const d = new Date();
  calYear = d.getFullYear();
  calMonth = d.getMonth();
  // try to render calendar after field creation / plan generation too
  const hook = async () => {
    if (!lastFieldId) return;
    await calReload();
  };
  // re-run calendar loading after actions
  window.generatePlan = (orig => async function(){ await orig(); await hook(); })(window.generatePlan);
  window.createField = (orig => async function(){ await orig(); /* plan not yet; wait */ })(window.createField);
  window.getSchedule = (orig => async function(){ await orig(); await hook(); })(window.getSchedule);

  // initial try (in case a field already exists)
  setTimeout(hook, 400);
    const keys = Object.keys(calDataByDate || {}).sort();
  if (keys.length) {
    const last = keys[keys.length - 1];
    document.getElementById('lastPlanDate').textContent = last;
  } else {
    document.getElementById('lastPlanDate').textContent = '-';
  }

});
// ===== end Calendar state =====

window.kbIngestURL = async function() {
  const url = document.getElementById('kb_url').value.trim();
  const tags = document.getElementById('kb_tags').value.trim();
  if (!url) return out('kbIngestOut', 'Enter a URL');
  const res = await fetch('/kb/ingest/url', {
    method:'POST', headers:{'Content-Type':'application/json'},
    body: JSON.stringify({ url, tags })
  });
  out('kbIngestOut', await res.json());
};

window.kbSearch = async function() {
  const q = document.getElementById('kb_q').value.trim();
  if (!q) return out('kbSearchOut', 'Enter a query');
  const res = await fetch('/kb/search?q=' + encodeURIComponent(q));
  out('kbSearchOut', await res.json());
};
window.lastFieldId = window.lastFieldId || 1;

function out(el, data){
  const n = document.getElementById(el);
  if (!n) return console.log(el, data);
  n.textContent = typeof data === 'string' ? data : JSON.stringify(data, null, 2);
}

async function createDelivery(){
  const body = {
    date: document.getElementById('d_date').value,
    mill_name: document.getElementById('d_mill').value,
    mill_quota_ton: parseFloat(document.getElementById('d_quota').value||0),
    time_window_from: document.getElementById('d_from').value || null,
    time_window_to: document.getElementById('d_to').value || null,
    notes: document.getElementById('d_notes').value || ''
  };
  const res = await fetch(`/fields/${lastFieldId}/deliveries`, {
    method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify(body)
  });
  out('deliveryOut', await res.json());
}

async function listDeliveries(){
  const now=new Date(), y=now.getFullYear(), m=now.getMonth();
  const from = new Date(y,m,1).toISOString().slice(0,10);
  const to   = new Date(y,m+1,0).toISOString().slice(0,10);
  const res = await fetch(`/fields/${lastFieldId}/deliveries?from=${from}&to=${to}`);
  out('deliveryOut', await res.json());
}

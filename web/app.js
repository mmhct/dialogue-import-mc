'use strict';
const $ = id => document.getElementById(id);
const PAGE_SIZE = 100;
let lines = [], pendingFiles = [], layout = null, selected = -1, partIndex = 0, page = 0;
let zoom = 1, revision = 0, timer = null, paintScale = 10, importing = false;
const example = '# 第一幕：此行只作备注，不进入游戏\nA：别出声，前面有人。\nB：你确定是这条路吗？\n/title @a title {"text":"遗迹入口","color":"gold"}\nA：拿好这颗钻石，跟我来。\n/give @a minecraft:diamond 1\n# 同一角色可以连续说多句\nA：看见那扇门了吗？\nA：等我发出信号，我们一起进去。\nB：明白！\n';
const templates = {
  dialogue: '', tp: '/tp @a ', give: '/give @a minecraft:diamond 1',
  title: '/title @a title {"text":"新的篇章","color":"gold"}',
  effect: '/effect give @a minecraft:night_vision 30 0 true', custom: '/'
};
function error(message) { $('error').textContent = message || ''; $('error').hidden = !message; }
function notice(message) { $('notice').textContent = message || ''; $('notice').hidden = !message; }
function interval(value) {
  const n = Number(value);
  if (!Number.isFinite(n) || n < .1 || n > 600 || Math.abs(n * 10 - Math.round(n * 10)) > 1e-5)
    throw Error('间隔应为 0.1～600 秒，最多一位小数。');
  return Math.round(n * 10);
}
function kind(line) { return line.kind || (line.text.startsWith('/') ? 'command' : 'dialogue'); }
function safeName(value) {
  let name = Array.from(value || 'dialogue').slice(0, 80).join('').replace(/[<>:"/\\|?*\u0000-\u001f\u007f-\u009f]/g, '_').replace(/[. ]+$/, '');
  if (!name || /^(con|prn|aux|nul|com[1-9]|lpt[1-9])(\.|$)/i.test(name)) name = 'dialogue_' + name;
  return name;
}
async function api(name, data, binary = false) {
  const response = await fetch('api/' + name, {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(data)});
  if (!response.ok) {
    const text = await response.text();
    let message = text;
    try { message = JSON.parse(text).error; } catch {}
    throw Error(message || '操作失败');
  }
  if (!binary) return response.json();
  const disposition = response.headers.get('Content-Disposition') || '';
  const utf = disposition.match(/filename\*=utf-8''([^;]+)/i), plain = disposition.match(/filename="?([^";]+)"?/i);
  return {blob: await response.blob(), name: utf ? decodeURIComponent(utf[1]) : plain ? plain[1] : 'dialogue.schem'};
}
function request() {
  for (const input of document.querySelectorAll('#lines input, #width, #gap, #splitevery')) {
    if (input.id === 'splitevery' && $('splitmode').value !== 'count') continue;
    if (!input.checkValidity()) throw Error((input.getAttribute('aria-label') || '设置') + '：请输入范围内的有效数字。');
  }
  const ends = new Set(partitions().map(p => p.end - 1));
  lines.forEach((line, i) => {
    if (!ends.has(i) && (!Number.isInteger(line.delay_tenths) || line.delay_tenths < 1 || line.delay_tenths > 6000))
      throw Error('第 ' + (i + 1) + ' 条的间隔应为 0.1～600 秒。');
  });
  return {
    schema_version: 2, lines: lines.map(l => ({...l, delay_tenths:Number.isInteger(l.delay_tenths) ? l.delay_tenths : 20})), name: $('projectname').value || 'dialogue',
    max_width: Number($('width').value), lane_gap: Number($('gap').value), target: $('target').value.trim(),
    split_mode: $('splitmode').value, split_every: Number($('splitevery').value)
  };
}
function download(blob, name) {
  const url = URL.createObjectURL(blob), a = document.createElement('a');
  a.href = url; a.download = name; document.body.append(a); a.click(); a.remove();
  setTimeout(() => URL.revokeObjectURL(url), 10000);
}
// This mirrors only the partition boundaries for editor labels; the Go builder
// remains authoritative for limits, geometry, timing and export.
function partitions() {
  if (!lines.length) return [];
  const mode = $('splitmode').value, every = Number($('splitevery').value), starts = [0];
  for (let i = 1; i < lines.length; i++) {
    const a = lines[i], b = lines[i - 1];
    if ((mode === 'count' && every > 0 && i % every === 0) ||
        (mode === 'manual' && a.break_before) ||
        (mode === 'files' && (a.source_group !== b.source_group || a.source_file !== b.source_file))) starts.push(i);
  }
  return starts.map((start, i) => ({start, end: starts[i + 1] ?? lines.length}));
}
function splitHelp() {
  $('splitcountlabel').hidden = $('splitmode').value !== 'count';
  $('splithelp').textContent = {
    none: '所有条目按顺序连成一条线路。',
    count: '对白与指令都计入条数，注释与已忽略的空行不计入。',
    files: '每段连续的来源文件独立成块。新插入条目继承相邻条目的来源。',
    manual: '点击任意条目的“分块”按钮，从该条开始新结构；再次点击可取消。'
  }[$('splitmode').value];
}
function button(text, label, action, disabled = false) {
  const b = document.createElement('button');
  b.type = 'button'; b.textContent = text; b.title = label; b.setAttribute('aria-label', label); b.disabled = disabled;
  b.addEventListener('click', e => { e.stopPropagation(); action(); });
  return b;
}
function renderQueue() {
  $('filequeue').replaceChildren();
  pendingFiles.forEach((file, i) => {
    const li = document.createElement('li'), name = document.createElement('span'), actions = document.createElement('div');
    name.textContent = (i + 1) + '. ' + file.name;
    actions.append(
      button('↑', '上移文件 ' + file.name, () => { [pendingFiles[i - 1], pendingFiles[i]] = [pendingFiles[i], pendingFiles[i - 1]]; renderQueue(); }, i === 0 || importing),
      button('↓', '下移文件 ' + file.name, () => { [pendingFiles[i + 1], pendingFiles[i]] = [pendingFiles[i], pendingFiles[i + 1]]; renderQueue(); }, i === pendingFiles.length - 1 || importing),
      button('移除', '移除文件 ' + file.name, () => { pendingFiles.splice(i, 1); renderQueue(); }, importing));
    li.append(name, actions); $('filequeue').append(li);
  });
  $('importfiles').disabled = !pendingFiles.length || importing;
  $('importfiles').textContent = importing ? '正在读取…' : '按顺序导入';
}
function queueFiles(files) {
  if (importing) return;
  const added = Array.from(files);
  if (added.some(f => f.size > 4 * 1024 * 1024)) { error('每份 TXT 最大为 4 MB。'); return; }
  if (pendingFiles.length + added.length > 256) { error('一次最多导入 256 份文件。'); return; }
  pendingFiles.push(...added); renderQueue(); error('');
}
function decodeBytes(bytes, choice) {
  const v = new Uint8Array(bytes);
  let encoding = choice;
  if (choice === 'auto') {
    if (v[0] === 255 && v[1] === 254) encoding = 'utf-16le';
    else if (v[0] === 254 && v[1] === 255) encoding = 'utf-16be';
    else {
      try { return new TextDecoder('utf-8', {fatal:true}).decode(v); }
      catch { encoding = 'gb18030'; }
    }
  }
  return new TextDecoder(encoding, {fatal:true}).decode(v);
}
async function importFiles() {
  if (importing || !pendingFiles.length) return;
  importing = true; renderQueue();
  const files = [...pendingFiles], replace = $('importmode').value === 'replace';
  const encoding = $('encoding').value, skipBlank = $('skipblank').checked;
  // Keep editor changes from racing a "replace" import.
  document.querySelector('.dialogue-panel').inert = true;
  $('example').disabled = true; $('projectfile').disabled = true;
  try {
    const delay = interval($('defaultdelay').value);
    let group = Math.max(0, ...lines.map(l => l.source_group || 0)), comments = 0, blanks = 0;
    const imported = [];
    for (const file of files) {
      const text = decodeBytes(await file.arrayBuffer(), encoding);
      const p = await api('parse', {text, skip_blank: skipBlank, delay_tenths: delay});
      group++;
      imported.push(...p.lines.map(l => ({...l, source_file: file.name, source_group: group})));
      comments += p.comment_lines; blanks += p.blank_lines;
      if (imported.length + (replace ? 0 : lines.length) > 10000) throw Error('一个项目最多 10000 条对白或指令。');
    }
    if (!imported.length) throw Error('这些文件没有可导入的条目（只有注释或已忽略的空行）。');
    const wasEmpty = !lines.length;
    lines = replace ? imported : [...lines, ...imported];
    if (replace || wasEmpty) $('projectname').value = files.length === 1 ? files[0].name.replace(/\.txt$/i, '') : '合并剧情';
    pendingFiles = []; selected = -1; partIndex = 0; page = 0;
    const detail = '已导入 ' + files.length + ' 份文件、' + imported.length + ' 条；跳过 ' + comments + ' 行注释' + (skipBlank ? '、' + blanks + ' 个空行' : '。空行已保留');
    $('filedetail').textContent = detail; notice(detail); error('');
    renderLines(); schedule(0);
  } catch (e) { error('导入失败：' + e.message); }
  finally {
    importing = false; renderQueue(); document.querySelector('.dialogue-panel').inert = false;
    $('example').disabled = false; $('projectfile').disabled = false;
  }
}
function renderLines() {
  const body = $('lines'), parts = partitions(), starts = new Set(parts.map(p => p.start)), ends = new Set(parts.map(p => p.end - 1));
  page = Math.max(0, Math.min(page, Math.ceil(lines.length / PAGE_SIZE) - 1));
  body.replaceChildren(); $('empty').hidden = lines.length > 0;
  $('linecount').textContent = lines.length + ' 条'; $('applydelay').disabled = !lines.length; $('saveproject').disabled = !lines.length;
  $('pagenumber').textContent = (page + 1) + ' / ' + Math.max(1, Math.ceil(lines.length / PAGE_SIZE));
  $('prevpage').disabled = page === 0; $('nextpage').disabled = (page + 1) * PAGE_SIZE >= lines.length;
  const fragment = document.createDocumentFragment();
  lines.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE).forEach((line, offset) => {
    const i = page * PAGE_SIZE + offset, tr = document.createElement('tr');
    tr.dataset.index = i; tr.classList.toggle('selected', i === selected); tr.classList.toggle('part-start', i > 0 && starts.has(i));
    const num = document.createElement('td'); num.textContent = i + 1;
    const text = document.createElement('td'), meta = document.createElement('div'), type = document.createElement('select');
    text.className = 'line-text'; meta.className = 'line-meta';
    type.setAttribute('aria-label', '第 ' + (i + 1) + ' 条类型');
    type.append(new Option('对白', 'dialogue'), new Option('指令', 'command')); type.value = kind(line);
    const source = document.createElement('span');
    source.textContent = (line.source_file || '手动输入') + (line.source_line ? ' · 源行 ' + line.source_line : '');
    source.title = source.textContent;
    meta.append(type, source);
    const editor = document.createElement('textarea'); editor.rows = 2; editor.value = line.text; editor.spellcheck = false;
    editor.setAttribute('aria-label', '第 ' + (i + 1) + ' 条内容');
    editor.classList.toggle('command-editor', kind(line) === 'command');
    editor.addEventListener('input', () => { line.text = editor.value; schedule(); });
    editor.addEventListener('focus', () => selectLine(i, true));
    type.addEventListener('change', () => { line.kind = type.value; editor.classList.toggle('command-editor', type.value === 'command'); schedule(); });
    text.append(meta, editor);
    const delay = document.createElement('td'), actions = document.createElement('td'); actions.className = 'line-actions';
    if (ends.has(i)) { delay.textContent = '块末条'; delay.className = 'muted'; }
    else {
      const input = document.createElement('input'), caption = document.createElement('div');
      input.type = 'number'; input.min = '.1'; input.max = '600'; input.step = '.1'; input.required = true;
      input.value = Number.isFinite(line.delay_tenths) ? (line.delay_tenths / 10).toFixed(1) : '';
      input.setAttribute('aria-label', '第 ' + (i + 1) + ' 条后的间隔秒数');
      caption.className = 'after';
      const update = () => { caption.textContent = Number.isFinite(line.delay_tenths) ? '秒 · ' + Math.ceil(line.delay_tenths / 4) + ' 个中继器' : '请输入有效间隔'; };
      input.addEventListener('input', () => {
        try { line.delay_tenths = interval(input.value); input.setCustomValidity(''); }
        catch (e) { line.delay_tenths = NaN; input.setCustomValidity(e.message); }
        update(); schedule();
      });
      update(); delay.append(input, caption);
    }
    actions.append(
      button('↑', '上移第 ' + (i + 1) + ' 条', () => moveLine(i, -1), i === 0),
      button('↓', '下移第 ' + (i + 1) + ' 条', () => moveLine(i, 1), i === lines.length - 1),
      button(i > 0 && starts.has(i) ? '取消分块' : '分块', '从第 ' + (i + 1) + ' 条开始分块', () => toggleBreak(i), i === 0),
      button('删除', '删除第 ' + (i + 1) + ' 条', () => { lines.splice(i, 1); selected = -1; renderLines(); schedule(0); }));
    tr.append(num, text, delay, actions);
    tr.addEventListener('click', () => selectLine(i, true)); fragment.append(tr);
  });
  body.append(fragment);
}
function moveLine(i, delta) {
  [lines[i], lines[i + delta]] = [lines[i + delta], lines[i]];
  selected = i + delta; page = Math.floor(selected / PAGE_SIZE); renderLines(); schedule(0);
}
function toggleBreak(i) {
  if ($('splitmode').value !== 'manual') {
    const starts = new Set(partitions().map(p => p.start));
    lines.forEach((l, n) => l.break_before = n > 0 && starts.has(n));
  }
  lines[i].break_before = !lines[i].break_before;
  $('splitmode').value = 'manual'; splitHelp(); renderLines(); schedule(0);
}
function insertLine(template) {
  try {
    if (lines.length >= 10000) throw Error('一个项目最多 10000 条。');
    const where = $('insertwhere').value;
    const i = selected < 0 || where === 'end' ? lines.length : selected + (where === 'after' ? 1 : 0);
    const source = (where === 'before' ? lines[i] : lines[i - 1]) || lines[i] || {};
    const line = {text:templates[template], kind:template === 'dialogue' ? 'dialogue' : 'command', delay_tenths:interval($('defaultdelay').value), source_line:0, source_file:source.source_file || '', source_group:source.source_group || 0};
    // Inserting before the first item in a manual part belongs to that part.
    if (where === 'before' && lines[i]?.break_before) { line.break_before = true; lines[i].break_before = false; }
    lines.splice(i, 0, line); selected = i; page = Math.floor(i / PAGE_SIZE); renderLines(); schedule(0);
    const editor = document.querySelector('tr[data-index="' + i + '"] textarea');
    editor.focus(); editor.setSelectionRange(editor.value.length, editor.value.length);
    notice(template === 'tp' ? '已插入 /tp @a ，请填写坐标，例如 100 64 200。' : template === 'custom' ? '已插入指令条目，请补全 / 后的内容。' : '');
  } catch (e) { error(e.message); }
}
function selectLine(i, scroll) {
  selected = i;
  document.querySelectorAll('#lines tr').forEach(row => row.classList.toggle('selected', Number(row.dataset.index) === i));
  const nextPart = partitions().findIndex(p => p.start <= i && i < p.end);
  if (nextPart >= 0 && partIndex !== nextPart) { partIndex = nextPart; schedule(0); return; }
  draw();
  const event = layout?.events[i - layout.parts[partIndex].start];
  if (event) {
    $('previewhint').textContent = '第 ' + (i + 1) + ' 条 · ' + (event.kind === 'command' ? '指令' : '对白') + ' · 坐标 ' + [event.x, event.y, event.z].join(', ') + ' · 本块首条后 ' + (event.at_tenths / 10).toFixed(1) + ' 秒';
    if (scroll) $('canvaswrap').scrollTo({top:Math.max(0, event.z * paintScale - 90), behavior:'smooth'});
  }
}
function disableExport(value) { $('export').disabled = value; $('exportpart').disabled = value; }
function schedule(delay = 180) {
  clearTimeout(timer); revision++; disableExport(true);
  const parts = partitions();
  partIndex = Math.max(0, Math.min(partIndex, parts.length - 1));
  if (selected >= 0) {
    const found = parts.findIndex(p => p.start <= selected && selected < p.end);
    if (found >= 0) partIndex = found;
  }
  layout = null; draw();
  $('size').textContent = $('repeatercount').textContent = $('duration').textContent = '—';
  $('rowcount').textContent = '';
  $('previewhint').textContent = '俯视图 · 命令方块位于各自的红石粉下方';
  $('partselect').disabled = true;
  if (!lines.length) { $('partselect').replaceChildren(new Option('尚无结构')); error(''); return; }
  const rev = revision; timer = setTimeout(() => updatePreview(rev), delay);
}
async function updatePreview(rev) {
  try {
    const result = await api('preview', {...request(), part_index:partIndex});
    if (rev !== revision) return;
    layout = result; error('');
    $('size').textContent = [result.width, result.length, result.height].join(' × ');
    $('repeatercount').textContent = result.repeaters.toLocaleString(); $('duration').textContent = (result.duration_tenths / 10).toFixed(1);
    $('rowcount').textContent = result.rows + ' 排 · 间距 ' + result.lane_gap + ' 格';
    $('partselect').replaceChildren(...result.parts.map(p => new Option('结构 ' + (p.index + 1) + ' · 第 ' + (p.start + 1) + '～' + p.end + ' 条', p.index)));
    $('partselect').value = result.part_index; $('partselect').disabled = false;
    const multi = result.parts.length > 1;
    $('exportpart').hidden = !multi; $('exportext').textContent = multi ? '.zip' : '.schem';
    $('exportlabel').textContent = multi ? '导出全部 ' + result.parts.length + ' 个结构' : '导出结构文件';
    $('exporthint').textContent = lines.length + ' 个命令方块 · ' + result.parts.length + ' 个独立结构' + (multi ? ' · ZIP 含清单和可编辑项目' : '');
    disableExport(false); draw(); if (selected >= 0) selectLine(selected, false);
  } catch (e) {
    if (rev !== revision) return;
    layout = null; error(e.message); disableExport(true); draw();
  }
}
function draw() {
  const canvas = $('canvas'), wrap = $('canvaswrap');
  $('previewempty').hidden = !!layout;
  if (!layout) { canvas.width = 1; canvas.height = 1; canvas.style.width = '1px'; canvas.style.height = '1px'; return; }
  const ratio = Math.min(window.devicePixelRatio || 1, 2), available = wrap.clientWidth - 34;
  let scale = Math.max(2, Math.min(20, available / layout.width)) * zoom;
  scale = Math.min(scale, 15000 / layout.length); paintScale = scale;
  const w = Math.max(1, Math.ceil(layout.width * scale)), h = Math.max(1, Math.ceil(layout.length * scale));
  canvas.width = Math.ceil(w * ratio); canvas.height = Math.ceil(h * ratio); canvas.style.width = w + 'px'; canvas.style.height = h + 'px';
  const ctx = canvas.getContext('2d'); ctx.scale(ratio, ratio); ctx.fillStyle = '#17251d'; ctx.fillRect(0, 0, w, h);
  if (scale >= 5) {
    ctx.strokeStyle = '#283b2d'; ctx.lineWidth = .4; ctx.beginPath();
    for (let x = 0; x <= layout.width; x++) { ctx.moveTo(x * scale, 0); ctx.lineTo(x * scale, h); }
    for (let z = 0; z <= layout.length; z++) { ctx.moveTo(0, z * scale); ctx.lineTo(w, z * scale); }
    ctx.stroke();
  }
  const colors = {button:'#e5ca73', dialogue:'#dfa163', command:'#7eb8d3', repeater:'#d3766d', wire:'#815347'};
  for (const c of layout.cells) {
    const x = c.x * scale, y = c.z * scale, pad = scale > 5 ? 1 : 0;
    ctx.fillStyle = colors[c.kind]; ctx.fillRect(x + pad, y + pad, Math.max(.7, scale - 2 * pad), Math.max(.7, scale - 2 * pad));
    if (c.kind === 'repeater' && scale >= 8) {
      ctx.strokeStyle = '#321d1b'; ctx.lineWidth = 1;
      const dx = {east:1,west:-1,north:0,south:0}[c.facing], dy = {east:0,west:0,north:-1,south:1}[c.facing], cx = x + scale / 2, cy = y + scale / 2;
      ctx.beginPath(); ctx.moveTo(cx - dx * scale * .25, cy - dy * scale * .25); ctx.lineTo(cx + dx * scale * .25, cy + dy * scale * .25);
      ctx.lineTo(cx + dx * scale * .08 - dy * scale * .14, cy + dy * scale * .08 + dx * scale * .14);
      ctx.moveTo(cx + dx * scale * .25, cy + dy * scale * .25); ctx.lineTo(cx + dx * scale * .08 + dy * scale * .14, cy + dy * scale * .08 - dx * scale * .14); ctx.stroke();
    }
    if ((c.kind === 'dialogue' || c.kind === 'command') && scale >= 11) {
      ctx.fillStyle = '#2d2419'; ctx.font = Math.max(7, scale * .51) + 'px Segoe UI, sans-serif'; ctx.textAlign = 'center'; ctx.textBaseline = 'middle';
      ctx.fillText(c.line + layout.parts[partIndex].start, x + scale / 2, y + scale / 2);
    }
  }
  const event = layout.events[selected - layout.parts[partIndex].start];
  if (event) { ctx.strokeStyle = '#ebfbdc'; ctx.lineWidth = 2; ctx.strokeRect(event.x * scale - 2, event.z * scale - 2, scale + 4, scale + 4); }
}
async function exportStructure(onlyPart) {
  const rev = revision;
  try {
    disableExport(true);
    const result = await api(onlyPart ? 'export-part' : 'export', {...request(), part_index:partIndex}, true);
    download(result.blob, result.name); notice('已生成 ' + result.name + '。多个结构请解压后分别导入；可用“保存项目”继续编辑。');
  } catch (e) { error(e.message); }
  finally { if (rev === revision) disableExport(!layout); }
}
$('txtfile').addEventListener('change', e => { queueFiles(e.target.files); e.target.value = ''; });
$('importfiles').addEventListener('click', importFiles);
for (const event of ['dragenter', 'dragover']) $('dropzone').addEventListener(event, e => { e.preventDefault(); $('dropzone').classList.add('drag-over'); });
for (const event of ['dragleave', 'drop']) $('dropzone').addEventListener(event, e => { e.preventDefault(); $('dropzone').classList.remove('drag-over'); if (event === 'drop') queueFiles(e.dataTransfer.files); });
$('example').addEventListener('click', async () => {
  try {
    const p = await api('parse', {text:example, skip_blank:true, delay_tenths:20});
    if (lines.length + p.lines.length > 10000) throw Error('一个项目最多 10000 条。');
    const group = Math.max(0, ...lines.map(l => l.source_group || 0)) + 1, start = lines.length;
    lines.push(...p.lines.map(l => ({...l, source_file:'混合示例.txt', source_group:group})));
    if (start === 0) $('projectname').value = '混合示例';
    selected = start; page = Math.floor(start / PAGE_SIZE); renderLines(); schedule(0);
    notice('已追加示例：含对白、标题和给予指令；2 行 # 注释已跳过。');
  } catch (e) { error(e.message); }
});
document.querySelectorAll('[data-template]').forEach(b => b.addEventListener('click', () => insertLine(b.dataset.template)));
$('applydelay').addEventListener('click', () => {
  try { const d = interval($('defaultdelay').value); lines.forEach(l => l.delay_tenths = d); renderLines(); schedule(0); }
  catch (e) { error(e.message); }
});
for (const id of ['width', 'gap', 'target', 'projectname']) $(id).addEventListener('input', () => schedule());
$('splitmode').addEventListener('change', () => { splitHelp(); renderLines(); schedule(0); });
$('splitevery').addEventListener('input', () => { renderLines(); schedule(); });
$('partselect').addEventListener('change', () => { partIndex = Number($('partselect').value); selected = -1; renderLines(); schedule(0); });
$('prevpage').addEventListener('click', () => { page--; renderLines(); });
$('nextpage').addEventListener('click', () => { page++; renderLines(); });
$('zoomout').addEventListener('click', () => { zoom = Math.max(.3, zoom / 1.3); draw(); });
$('zoomin').addEventListener('click', () => { zoom = Math.min(4, zoom * 1.3); draw(); });
$('zoomreset').addEventListener('click', () => { zoom = 1; draw(); });
$('canvas').addEventListener('click', e => {
  if (!layout) return;
  const rect = e.currentTarget.getBoundingClientRect(), x = Math.floor((e.clientX - rect.left) / paintScale), z = Math.floor((e.clientY - rect.top) / paintScale);
  const local = layout.events.findIndex(p => p.x === x && p.z === z);
  if (local >= 0) {
    const i = local + layout.parts[partIndex].start; page = Math.floor(i / PAGE_SIZE); renderLines(); selectLine(i, false);
    document.querySelector('tr[data-index="' + i + '"]').scrollIntoView({block:'nearest',behavior:'smooth'});
  }
});
$('export').addEventListener('click', () => exportStructure(false));
$('exportpart').addEventListener('click', () => exportStructure(true));
$('saveproject').addEventListener('click', () => {
  try { download(new Blob([JSON.stringify(request(), null, 2)], {type:'application/json'}), safeName($('projectname').value) + '.dialogue.json'); notice('已保存可编辑项目（含分块、指令与来源信息）。'); }
  catch (e) { error(e.message); }
});
$('projectfile').addEventListener('change', async e => {
  const file = e.target.files[0]; if (!file) return;
  try {
    if (file.size > 8 * 1024 * 1024) throw Error('项目文件超过 8 MB');
    const data = JSON.parse(await file.text());
    if (!Array.isArray(data.lines) || !data.lines.length || data.lines.length > 10000 ||
        !data.lines.every(l => l && typeof l.text === 'string' && Number.isInteger(l.delay_tenths) && (l.kind === undefined || ['command', 'dialogue'].includes(l.kind))) ||
        (data.schema_version !== undefined && ![0, 1, 2].includes(data.schema_version)) ||
        (data.split_mode !== undefined && !['none', 'count', 'files', 'manual'].includes(data.split_mode)))
      throw Error('不支持的项目格式或版本');
    // Draft commands may be incomplete. Load them for editing; preview/export
    // still run full server validation. Older single-file projects remain valid.
    lines = data.lines.map(l => ({text:l.text, delay_tenths:l.delay_tenths, kind:kind(l), source_line:Number.isInteger(l.source_line) ? l.source_line : 0, source_file:typeof l.source_file === 'string' ? l.source_file : '', source_group:Number.isInteger(l.source_group) ? l.source_group : 0, break_before:l.break_before === true}));
    $('width').value = data.max_width; $('gap').value = data.lane_gap ?? 3; $('target').value = data.target ?? '@a';
    $('projectname').value = data.name || file.name.replace(/\.dialogue\.json$/i, '');
    $('splitmode').value = data.split_mode || 'none'; $('splitevery').value = data.split_every || 100;
    selected = -1; page = 0; partIndex = 0; splitHelp(); renderLines(); schedule(0); notice('项目已载入。旧版项目使用 3 格掉头间距。');
  } catch (err) { error('载入失败：' + err.message); }
  finally { e.target.value = ''; }
});
$('quit').addEventListener('click', async () => {
  if (lines.length && !confirm('退出后未保存的剧情编辑将丢失。确定退出软件吗？')) return;
  try { await api('shutdown', {}); document.querySelector('.workspace').classList.add('offline'); $('quit').disabled = true; notice('软件已退出，可以关闭本页。'); }
  catch (e) { error(e.message); }
});
new ResizeObserver(() => draw()).observe($('canvaswrap'));
splitHelp(); renderLines();

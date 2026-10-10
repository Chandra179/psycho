// Browser glue for the WebAssembly build: loads psycho.wasm, runs the analysis
// on the visitor's device and keeps optional history in IndexedDB.
(function () {
  'use strict';

  var form = document.getElementById('form');
  var textArea = document.getElementById('text');
  var fileInput = document.getElementById('file');
  var keep = document.getElementById('keep');
  var submit = document.getElementById('submit');
  var submitLabel = document.getElementById('submit-label');
  var status = document.getElementById('status');
  var result = document.getElementById('result');
  var intake = document.getElementById('intake');
  var historyBox = document.getElementById('history');
  var historyList = document.getElementById('history-list');
  var current = null; // {json} of the report on screen
  var MAX_BYTES = 1 << 20; // replaced by psycho.maxBytes once the analyzer loads
  var MAX_SAVED = 50; // saved readings kept on this device; the oldest are dropped
  var encoder = new TextEncoder();
  var lengthNote = document.getElementById('length-note');
  var overLimit = false;

  function setStatus(msg) { status.textContent = msg; }

  function formatBytes(n) {
    return n >= 1048576 ? parseFloat((n / 1048576).toFixed(2)) + ' MB' : Math.max(1, Math.round(n / 1024)) + ' KB';
  }

  var TEXTAREA_OK = ['border-stone-300', 'focus:border-teal-500', 'focus:ring-teal-500/20'];
  var TEXTAREA_BAD = ['border-rose-500', 'focus:border-rose-500', 'focus:ring-rose-500/20'];

  // Shows the size against the limit: a quiet amber note when close, a red
  // alert, a red textarea and a blocked button when over.
  function checkLength() {
    var bytes = encoder.encode(textArea.value).length;
    var ready = !!(window.psycho && window.psycho.ready);
    overLimit = bytes > MAX_BYTES;
    lengthNote.replaceChildren();
    if (overLimit) {
      var title = document.createElement('strong');
      title.textContent = 'Too long to analyze. ';
      lengthNote.append(title, 'This text is ' + formatBytes(bytes) + ' and the limit is ' + formatBytes(MAX_BYTES) +
        ' (about 170,000 words). Shorten it or analyze one part at a time.');
      lengthNote.className = 'mt-3 rounded-xl border border-rose-300 bg-rose-50 px-4 py-3 text-sm text-rose-800';
      lengthNote.setAttribute('role', 'alert');
    } else if (bytes > MAX_BYTES * 0.8) {
      lengthNote.textContent = 'Getting long: ' + formatBytes(bytes) + ' of ' + formatBytes(MAX_BYTES) + ' used.';
      lengthNote.className = 'mt-3 rounded-xl border border-amber-300 bg-amber-50 px-4 py-2 text-sm text-amber-800';
      lengthNote.removeAttribute('role');
    } else {
      lengthNote.className = 'hidden';
      lengthNote.removeAttribute('role');
    }
    textArea.setAttribute('aria-invalid', overLimit ? 'true' : 'false');
    textArea.classList.remove.apply(textArea.classList, overLimit ? TEXTAREA_OK : TEXTAREA_BAD);
    textArea.classList.add.apply(textArea.classList, overLimit ? TEXTAREA_BAD : TEXTAREA_OK);
    if (ready) submitLabel.textContent = overLimit ? 'Text is too long' : 'Analyze my writing';
    submit.disabled = overLimit || !ready;
  }

  function showError(msg) {
    var box = document.createElement('div');
    box.setAttribute('role', 'alert');
    box.className = 'max-w-[60rem] mx-auto px-4 sm:px-6 lg:px-8';
    var inner = document.createElement('div');
    inner.className = 'rounded-2xl border border-rose-200 bg-rose-50 p-5 text-sm leading-relaxed text-rose-800';
    var title = document.createElement('p');
    title.className = 'font-semibold';
    title.textContent = 'We could not analyze this writing.';
    var body = document.createElement('p');
    body.className = 'mt-1';
    body.textContent = msg || 'Please try again.';
    inner.append(title, body);
    box.append(inner);
    result.replaceChildren(box);
  }

  // ---- WebAssembly loading ----
  // scripts/build-site.sh rewrites this to the content-hashed file name.
  var WASM_URL = 'psycho.wasm';

  function loadWasm() {
    var go = new Go();
    var load = WebAssembly.instantiateStreaming
      ? WebAssembly.instantiateStreaming(fetch(WASM_URL), go.importObject)
      : Promise.reject(new Error('no streaming'));
    return load.catch(function () {
      // Hosts that do not serve .wasm as application/wasm need the slow path.
      return fetch(WASM_URL).then(function (r) { return r.arrayBuffer(); })
        .then(function (b) { return WebAssembly.instantiate(b, go.importObject); });
    }).then(function (res) {
      go.run(res.instance); // runs for the page's lifetime
      return waitForPsycho();
    });
  }

  function waitForPsycho() {
    return new Promise(function (resolve, reject) {
      var tries = 0;
      (function check() {
        var p = window.psycho;
        if (p && p.error) { reject(new Error(p.error)); return; }
        if (p && p.ready) { resolve(p); return; }
        if (++tries > 200) { reject(new Error('The analyzer did not start.')); return; }
        setTimeout(check, 25);
      })();
    });
  }

  // ---- History in IndexedDB (opt-in, local only) ----
  var DB_NAME = 'psycho', STORE = 'readings';

  function openDB() {
    return new Promise(function (resolve, reject) {
      var req;
      try { req = indexedDB.open(DB_NAME, 1); } catch (e) { reject(e); return; }
      req.onupgradeneeded = function () { req.result.createObjectStore(STORE, { keyPath: 'id' }); };
      req.onsuccess = function () { resolve(req.result); };
      req.onerror = function () { reject(req.error); };
    });
  }

  function tx(mode, fn) {
    return openDB().then(function (db) {
      return new Promise(function (resolve, reject) {
        var t = db.transaction(STORE, mode);
        var out = fn(t.objectStore(STORE));
        t.oncomplete = function () { db.close(); resolve(out && out.result); };
        t.onerror = t.onabort = function () { db.close(); reject(t.error); };
      });
    });
  }

  // A short label for a saved reading: the first line of the text, trimmed.
  function titleFrom(text) {
    var line = '';
    text.split('\n').some(function (l) { line = l.replace(/^[#>*\-\s]+/, '').replace(/\s+/g, ' ').trim(); return line !== ''; });
    if (line.length <= 60) return line;
    var cut = line.slice(0, 60);
    return cut.slice(0, Math.max(cut.lastIndexOf(' '), 30)).replace(/[\s.,;:]+$/, '') + '...';
  }

  function saveReading(json, title) {
    var data;
    try { data = JSON.parse(json); } catch (e) { return Promise.resolve(); }
    var rec = {
      id: data.analysis_id || String(Date.now()),
      savedAt: Date.now(),
      title: title || '',
      wordCount: data.word_count || 0,
      json: json
    };
    return tx('readwrite', function (s) { return s.put(rec); }).then(pruneReadings).catch(function () {
      setStatus('The reading could not be saved on this device.');
    });
  }

  // Keeps only the newest MAX_SAVED readings so history cannot grow without bound.
  function pruneReadings() {
    return listReadings().then(function (rows) {
      var extra = rows.slice(MAX_SAVED);
      if (!extra.length) return;
      return tx('readwrite', function (s) { extra.forEach(function (r) { s.delete(r.id); }); });
    });
  }

  function listReadings() {
    return tx('readonly', function (s) { return s.getAll(); }).then(function (rows) {
      return (rows || []).sort(function (a, b) { return b.savedAt - a.savedAt; });
    }).catch(function () { return []; });
  }

  function renderHistory() {
    listReadings().then(function (rows) {
      historyList.replaceChildren();
      historyBox.classList.toggle('hidden', rows.length === 0);
      rows.forEach(function (r) {
        var li = document.createElement('li');
        li.className = 'flex flex-wrap items-center justify-between gap-3 px-4 py-3 text-sm';
        var label = document.createElement('span');
        label.className = 'min-w-0';
        var name = document.createElement('span');
        name.className = 'block font-medium truncate';
        name.textContent = r.title || 'Untitled reading';
        var meta = document.createElement('span');
        meta.className = 'block text-stone-500';
        meta.textContent = new Date(r.savedAt).toLocaleString() + ' \u00b7 ' +
          r.wordCount.toLocaleString() + ' words';
        label.append(name, meta);
        var actions = document.createElement('span');
        actions.className = 'flex gap-3';
        var open = document.createElement('button');
        open.type = 'button';
        open.textContent = 'Open';
        open.className = 'underline hover:text-stone-900';
        open.addEventListener('click', function () { show(r.json, false); });
        var rename = document.createElement('button');
        rename.type = 'button';
        rename.textContent = 'Rename';
        rename.className = 'underline text-stone-600 hover:text-stone-900';
        rename.addEventListener('click', function () {
          var t = window.prompt('Title for this reading', r.title || '');
          if (t === null) return;
          r.title = t.trim().slice(0, 100);
          tx('readwrite', function (s) { return s.put(r); }).then(renderHistory);
        });
        var del = document.createElement('button');
        del.type = 'button';
        del.textContent = 'Delete';
        del.className = 'underline text-stone-600 hover:text-stone-900';
        del.addEventListener('click', function () {
          tx('readwrite', function (s) { return s.delete(r.id); }).then(renderHistory);
        });
        actions.append(open, rename, del);
        li.append(label, actions);
        historyList.append(li);
      });
    });
  }

  document.getElementById('history-clear').addEventListener('click', function () {
    if (!window.confirm('Delete every reading saved on this device?')) return;
    tx('readwrite', function (s) { return s.clear(); }).then(renderHistory);
  });

  // ---- Showing and exporting reports ----
  function show(json, scroll) {
    var out = window.psycho.render(json, false);
    if (!out.ok) { showError(out.error); return; }
    current = { json: json };
    result.innerHTML = out.html; // markup comes from the compiled Go templates, which escape all text
    intake.classList.add('print:hidden');
    if (scroll !== false) {
      result.scrollIntoView({ behavior: 'smooth', block: 'start' });
      result.focus({ preventScroll: true });
    }
  }

  function download(name, mime, content) {
    var url = URL.createObjectURL(new Blob([content], { type: mime }));
    var a = document.createElement('a');
    a.href = url;
    a.download = name;
    document.body.append(a);
    a.click();
    a.remove();
    setTimeout(function () { URL.revokeObjectURL(url); }, 1000);
  }

  result.addEventListener('click', function (e) {
    var btn = e.target.closest('[data-psycho-action]');
    if (!btn) return;
    switch (btn.getAttribute('data-psycho-action')) {
      case 'new':
        result.replaceChildren();
        current = null;
        window.scrollTo({ top: 0, behavior: 'smooth' });
        textArea.focus({ preventScroll: true });
        break;
      case 'print':
        window.print();
        break;
      case 'download-html':
        if (!current) return;
        var page = window.psycho.render(current.json, true);
        if (page.ok) {
          download('psycho-report-' + new Date().toISOString().slice(0, 10) + '.html', 'text/html', page.html);
        }
        break;
    }
  });

  // ---- Form ----
  fileInput.addEventListener('change', function () {
    var f = fileInput.files && fileInput.files[0];
    if (!f) return;
    function reject(msg) { setStatus(msg); fileInput.value = ''; }
    if (!/\.(txt|md|markdown)$/i.test(f.name)) { reject('Please choose a .txt or .md file.'); return; }
    if (f.size > MAX_BYTES) { reject('That file is ' + formatBytes(f.size) + ', over the ' + formatBytes(MAX_BYTES) + ' limit.'); return; }
    var reader = new FileReader();
    reader.onload = function () {
      var text = String(reader.result);
      if (text.indexOf('\u0000') !== -1) { reject('That file does not look like plain text.'); return; }
      textArea.value = text;
      setStatus('Loaded ' + f.name + '.');
      checkLength();
    };
    reader.onerror = function () { reject('That file could not be read.'); };
    reader.readAsText(f);
  });

  textArea.addEventListener('input', checkLength);

  form.addEventListener('submit', function (e) {
    e.preventDefault();
    if (!window.psycho || !window.psycho.ready) return;
    checkLength();
    if (overLimit) { setStatus(lengthNote.textContent); return; }
    submit.disabled = true;
    setStatus('Analyzing...');
    // Let the status paint before the synchronous analysis runs.
    setTimeout(function () {
      var out = window.psycho.analyze(textArea.value);
      submit.disabled = false;
      setStatus('');
      if (!out.ok) { showError(out.error); return; }
      current = { json: out.json };
      result.innerHTML = out.html;
      intake.classList.add('print:hidden');
      result.scrollIntoView({ behavior: 'smooth', block: 'start' });
      result.focus({ preventScroll: true });
      if (keep.checked) saveReading(out.json, titleFrom(textArea.value)).then(renderHistory);
    }, 30);
  });

  loadWasm().then(function () {
    if (window.psycho.maxBytes) MAX_BYTES = window.psycho.maxBytes;
    checkLength();
    submitLabel.textContent = 'Analyze my writing';
    renderHistory();
  }).catch(function (err) {
    submitLabel.textContent = 'Analyzer unavailable';
    setStatus('The analyzer could not load: ' + err.message +
      ' This page must be opened from a web address (http or https), not as a local file.');
  });
})();

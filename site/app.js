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

  function setStatus(msg) { status.textContent = msg; }

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
  function loadWasm() {
    var go = new Go();
    var load = WebAssembly.instantiateStreaming
      ? WebAssembly.instantiateStreaming(fetch('psycho.wasm'), go.importObject)
      : Promise.reject(new Error('no streaming'));
    return load.catch(function () {
      // Hosts that do not serve .wasm as application/wasm need the slow path.
      return fetch('psycho.wasm').then(function (r) { return r.arrayBuffer(); })
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

  function saveReading(json) {
    var data;
    try { data = JSON.parse(json); } catch (e) { return Promise.resolve(); }
    var rec = {
      id: data.analysis_id || String(Date.now()),
      savedAt: Date.now(),
      wordCount: data.word_count || 0,
      json: json
    };
    return tx('readwrite', function (s) { return s.put(rec); }).catch(function () {
      setStatus('The reading could not be saved on this device.');
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
        label.textContent = new Date(r.savedAt).toLocaleString() + ' · ' +
          r.wordCount.toLocaleString() + ' words';
        var actions = document.createElement('span');
        actions.className = 'flex gap-3';
        var open = document.createElement('button');
        open.type = 'button';
        open.textContent = 'Open';
        open.className = 'underline hover:text-stone-900';
        open.addEventListener('click', function () { show(r.json, false); });
        var del = document.createElement('button');
        del.type = 'button';
        del.textContent = 'Delete';
        del.className = 'underline text-stone-600 hover:text-stone-900';
        del.addEventListener('click', function () {
          tx('readwrite', function (s) { return s.delete(r.id); }).then(renderHistory);
        });
        actions.append(open, del);
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
    if (f.size > 1000000) { setStatus('That file is larger than 1 MB.'); fileInput.value = ''; return; }
    var reader = new FileReader();
    reader.onload = function () { textArea.value = String(reader.result); setStatus('Loaded ' + f.name + '.'); };
    reader.onerror = function () { setStatus('That file could not be read.'); };
    reader.readAsText(f);
  });

  form.addEventListener('submit', function (e) {
    e.preventDefault();
    if (!window.psycho || !window.psycho.ready) return;
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
      if (keep.checked) saveReading(out.json).then(renderHistory);
    }, 30);
  });

  loadWasm().then(function () {
    submit.disabled = false;
    submitLabel.textContent = 'Analyze my writing';
    renderHistory();
  }).catch(function (err) {
    submitLabel.textContent = 'Analyzer unavailable';
    setStatus('The analyzer could not load: ' + err.message +
      ' This page must be opened from a web address (http or https), not as a local file.');
  });
})();

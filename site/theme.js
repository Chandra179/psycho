// Light and dark theme. Loaded in <head> so a saved choice applies before first
// paint. With no saved choice the page follows the system setting (see app.css).
(function () {
  'use strict';
  var KEY = 'psycho-theme';
  var root = document.documentElement;

  function saved() {
    try { var v = localStorage.getItem(KEY); return v === 'light' || v === 'dark' ? v : ''; } catch (e) { return ''; }
  }
  function systemDark() {
    return !!(window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches);
  }
  function current() { return root.getAttribute('data-theme') || (systemDark() ? 'dark' : 'light'); }

  if (saved()) root.setAttribute('data-theme', saved());

  function sync(btn) {
    var dark = current() === 'dark';
    btn.setAttribute('aria-pressed', dark ? 'true' : 'false');
    var opts = btn.querySelectorAll('[data-theme-option]');
    for (var i = 0; i < opts.length; i++) {
      if ((opts[i].getAttribute('data-theme-option') === 'dark') === dark) opts[i].setAttribute('data-on', '');
      else opts[i].removeAttribute('data-on');
    }
  }

  document.addEventListener('DOMContentLoaded', function () {
    var btn = document.getElementById('theme-toggle');
    if (!btn) return;
    sync(btn);
    btn.addEventListener('click', function () {
      var next = current() === 'dark' ? 'light' : 'dark';
      root.setAttribute('data-theme', next);
      try { localStorage.setItem(KEY, next); } catch (e) {}
      sync(btn);
    });
    if (window.matchMedia) {
      var mq = window.matchMedia('(prefers-color-scheme: dark)');
      var onChange = function () { if (!saved()) sync(btn); };
      if (mq.addEventListener) mq.addEventListener('change', onChange);
    }
  });
})();

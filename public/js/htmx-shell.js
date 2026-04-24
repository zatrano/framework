(function () {
  'use strict';

  function normalizePath(path) {
    var p = (path || '').replace(/\/$/, '') || '/';
    if (p === '/dashboard/home') {
      p = '/dashboard';
    }
    return p;
  }

  function scrollAfterNav() {
    window.scrollTo(0, 0);
    document.documentElement.scrollTop = 0;
    document.body.scrollTop = 0;
    var mc = document.getElementById('main-content');
    if (mc) {
      mc.scrollTop = 0;
    }
    var mw = document.querySelector('.main-wrapper');
    if (mw) {
      mw.scrollTop = 0;
    }
  }

  function syncActiveMenu() {
    var path = normalizePath(window.location.pathname);
    var best = null;
    var bestLen = -1;
    document.querySelectorAll('.sidebar .menu-item[href]').forEach(function (a) {
      try {
        var u = new URL(a.getAttribute('href'), window.location.origin);
        var p = normalizePath(u.pathname);
        var len = p.length;
        if ((path === p || path.startsWith(p + '/')) && len > bestLen) {
          best = a;
          bestLen = len;
        }
      } catch (e) {
        /* ignore */
      }
    });
    document.querySelectorAll('.sidebar .menu-item[href]').forEach(function (a) {
      a.classList.remove('active');
    });
    if (best) {
      best.classList.add('active');
    }
    document.querySelectorAll('.sidebar .submenu').forEach(function (sub) {
      var hasActive = !!sub.querySelector('a.menu-item.active');
      sub.classList.toggle('show', hasActive);
      var drop = sub.previousElementSibling;
      if (drop && drop.classList.contains('menu-dropdown')) {
        drop.classList.toggle('open', hasActive);
      }
    });
  }

  function syncWebsiteNav() {
    var nav = document.querySelector('.website-nav');
    if (!nav) {
      return;
    }
    var path = normalizePath(window.location.pathname);
    nav.querySelectorAll('a[href]').forEach(function (a) {
      try {
        var u = new URL(a.getAttribute('href'), window.location.origin);
        var p = normalizePath(u.pathname);
        var isHome = p === '/' || p === '';
        var active = path === p || (isHome && (path === '/' || path === ''));
        a.classList.toggle('active', active);
      } catch (e) {
        /* ignore */
      }
    });
  }

  function initToastsIn(root) {
    if (!root || !window.bootstrap) {
      return;
    }
    root.querySelectorAll('.toast').forEach(function (el) {
      var toast = new bootstrap.Toast(el);
      toast.show();
    });
  }

  function maybeRenderTurnstile(root) {
    if (!root || !window.turnstile) {
      return;
    }
    root.querySelectorAll('.cf-turnstile').forEach(function (el) {
      if (el.getAttribute('data-cf-widget-id')) {
        return;
      }
      var key = el.getAttribute('data-sitekey');
      if (!key) {
        return;
      }
      try {
        window.turnstile.render(el, {
          sitekey: key,
          theme: el.getAttribute('data-theme') || 'light',
          size: el.getAttribute('data-size') || 'normal'
        });
      } catch (e) {
        /* ignore */
      }
    });
  }

  function pageTitleFromHtmxXhr(xhr) {
    if (!xhr) {
      return '';
    }
    var b64 = xhr.getResponseHeader('X-Page-Title-B64');
    if (!b64) {
      return '';
    }
    try {
      b64 = b64.replace(/\s/g, '');
      var bin = atob(b64);
      var bytes = new Uint8Array(bin.length);
      for (var i = 0; i < bin.length; i++) {
        bytes[i] = bin.charCodeAt(i);
      }
      return new TextDecoder('utf-8').decode(bytes);
    } catch (e) {
      return '';
    }
  }

  function scheduleTurnstileRender(root, attempts) {
    if (!root || !root.querySelector('.cf-turnstile')) {
      return;
    }
    if (window.turnstile) {
      maybeRenderTurnstile(root);
      return;
    }
    if (attempts > 50) {
      return;
    }
    setTimeout(function () {
      scheduleTurnstileRender(root, attempts + 1);
    }, 80);
  }

  function onMainContentSwap(evt) {
    var t = evt.detail && evt.detail.target;
    if (!t || t.id !== 'main-content') {
      return;
    }
    var xhr = evt.detail.xhr;
    var title = pageTitleFromHtmxXhr(xhr);
    if (title) {
      document.title = title;
    }
    syncActiveMenu();
    syncWebsiteNav();
    scrollAfterNav();
    initToastsIn(t);
    scheduleTurnstileRender(t, 0);
  }

  document.addEventListener('htmx:afterSwap', onMainContentSwap);
  window.addEventListener('popstate', function () {
    syncActiveMenu();
    scrollAfterNav();
  });
  document.addEventListener('DOMContentLoaded', function () {
    syncActiveMenu();
    syncWebsiteNav();
  });
})();

(function () {
  'use strict';

  var SIDEBAR_COLLAPSED_KEY = 'sidebarCollapsed';

  function mqDesktop() {
    return window.matchMedia('(min-width: 992px)').matches;
  }

  function syncSidebarCollapseButton() {
    var btn = document.getElementById('sidebarCollapseToggle');
    if (!btn) {
      return;
    }
    var collapsed = document.body.classList.contains('sidebar-collapsed');
    var icon = btn.querySelector('i');
    if (icon) {
      icon.className = collapsed ? 'bi bi-chevron-double-right' : 'bi bi-chevron-double-left';
    }
    btn.setAttribute('aria-expanded', collapsed ? 'false' : 'true');
    btn.setAttribute('title', collapsed ? 'Kenar çubuğunu göster' : 'Kenar çubuğunu gizle');
  }

  function setSidebarCollapsed(collapsed) {
    if (collapsed) {
      document.body.classList.add('sidebar-collapsed');
    } else {
      document.body.classList.remove('sidebar-collapsed');
    }
    syncSidebarCollapseButton();
    try {
      localStorage.setItem(SIDEBAR_COLLAPSED_KEY, collapsed ? '1' : '0');
    } catch (e) {
      /* ignore */
    }
  }

  function initSidebarCollapse() {
    var btn = document.getElementById('sidebarCollapseToggle');
    if (!btn) {
      return;
    }
    try {
      if (localStorage.getItem(SIDEBAR_COLLAPSED_KEY) === '1' && mqDesktop()) {
        document.body.classList.add('sidebar-collapsed');
      }
    } catch (e) {
      /* ignore */
    }
    syncSidebarCollapseButton();
    btn.addEventListener('click', function () {
      if (!mqDesktop()) {
        return;
      }
      setSidebarCollapsed(!document.body.classList.contains('sidebar-collapsed'));
    });
    window.addEventListener('resize', function () {
      if (!mqDesktop()) {
        document.body.classList.remove('sidebar-collapsed');
      } else {
        try {
          if (localStorage.getItem(SIDEBAR_COLLAPSED_KEY) === '1') {
            document.body.classList.add('sidebar-collapsed');
          }
        } catch (e) {
          /* ignore */
        }
      }
      syncSidebarCollapseButton();
    });
  }

  const sidebar = document.getElementById('sidebar');
  const sidebarOverlay = document.getElementById('sidebarOverlay');
  const sidebarToggle = document.getElementById('sidebarToggle');
  const sidebarCollapseToggle = document.getElementById('sidebarCollapseToggle');

  function closeMobileSidebar() {
    if (!sidebar || !sidebarOverlay || mqDesktop()) {
      return;
    }
    sidebar.classList.remove('show');
    sidebarOverlay.classList.remove('show');
  }

  function isSidebarOpenOnMobile() {
    return !mqDesktop() && !!sidebar && sidebar.classList.contains('show');
  }

  if (sidebar && sidebarOverlay && sidebarToggle) {
    sidebarToggle.addEventListener('click', function () {
      sidebar.classList.toggle('show');
      sidebarOverlay.classList.toggle('show');
    });
    sidebarOverlay.addEventListener('click', function () {
      closeMobileSidebar();
    });

    document.addEventListener('click', function (event) {
      if (!isSidebarOpenOnMobile()) {
        return;
      }
      var target = event.target;
      if (!target) {
        return;
      }
      var clickedInsideSidebar = sidebar.contains(target);
      var clickedSidebarToggle = sidebarToggle.contains(target);
      var clickedDesktopCollapse = !!sidebarCollapseToggle && sidebarCollapseToggle.contains(target);
      if (clickedInsideSidebar || clickedSidebarToggle || clickedDesktopCollapse) {
        return;
      }
      closeMobileSidebar();
    });
  }

  document.addEventListener('DOMContentLoaded', initSidebarCollapse);

  window.toggleSubmenu = function (element) {
    if (!element) {
      return;
    }
    element.classList.toggle('open');
    const submenu = element.nextElementSibling;
    if (submenu) {
      submenu.classList.toggle('show');
    }
  };

  const menuItems = document.querySelectorAll('.menu-item:not(.menu-dropdown)');
  menuItems.forEach(function (item) {
    item.addEventListener('click', function () {
      menuItems.forEach(function (i) {
        i.classList.remove('active');
      });
      item.classList.add('active');
      closeMobileSidebar();
    });
  });
})();

(function () {
  'use strict';

  const sidebar = document.getElementById('sidebar');
  const sidebarOverlay = document.getElementById('sidebarOverlay');
  const sidebarToggle = document.getElementById('sidebarToggle');
  if (sidebar && sidebarOverlay && sidebarToggle) {
    sidebarToggle.addEventListener('click', function () {
      sidebar.classList.toggle('show');
      sidebarOverlay.classList.toggle('show');
    });
    sidebarOverlay.addEventListener('click', function () {
      sidebar.classList.remove('show');
      sidebarOverlay.classList.remove('show');
    });
  }

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
    });
  });

  const fullscreenBtn = document.getElementById('fullscreenBtn');
  if (fullscreenBtn) {
    fullscreenBtn.addEventListener('click', function () {
      if (!document.fullscreenElement) {
        document.documentElement.requestFullscreen();
      } else {
        document.exitFullscreen();
      }
    });
  }
})();

(function () {
  'use strict';

  function setEyeIcon(icon, passwordVisible) {
    if (!icon) {
      return;
    }
    icon.classList.remove('bi-eye', 'bi-eye-slash');
    icon.classList.add(passwordVisible ? 'bi-eye-slash' : 'bi-eye');
  }

  function bindToggles() {
    document.querySelectorAll('.auth-input-wrap .toggle-password').forEach(function (btn) {
      btn.addEventListener('click', function () {
        const wrap = btn.closest('.auth-input-wrap');
        if (!wrap) {
          return;
        }
        const input = wrap.querySelector('input');
        if (!input) {
          return;
        }
        const icon = btn.querySelector('i');
        const show = input.type === 'password';
        input.type = show ? 'text' : 'password';
        setEyeIcon(icon, show);
      });
    });
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', bindToggles);
  } else {
    bindToggles();
  }
})();

(function () {
  var body = document.body;
  if (!body) return;

  function setOpen(open) {
    body.classList.toggle("sidebar-open", open);
  }

  document.querySelectorAll("[data-sidebar-toggle]").forEach(function (el) {
    el.addEventListener("click", function () {
      setOpen(!body.classList.contains("sidebar-open"));
    });
  });
  document.querySelectorAll("[data-sidebar-close]").forEach(function (el) {
    el.addEventListener("click", function () {
      setOpen(false);
    });
  });

  var path = location.pathname.replace(/\/+$/, "") || "/";
  document.querySelectorAll(".sidebar-nav-link[href]").forEach(function (a) {
    var href = (a.getAttribute("href") || "").replace(/\/+$/, "") || "/";
    if (href === "/" || href === "/dashboard") {
      if (path === href) a.classList.add("sidebar-nav-link--active");
      return;
    }
    if (path === href || path.indexOf(href + "/") === 0) {
      a.classList.add("sidebar-nav-link--active");
    }
  });
})();

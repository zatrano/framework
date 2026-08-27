(function () {
  var root = document.querySelector("[data-analytics-poll]");
  if (!root) return;
  var url = root.getAttribute("data-analytics-poll") || "/api/v1/analytics/overview";
  var interval = parseInt(root.getAttribute("data-analytics-interval") || "15000", 10);
  if (interval < 3000) interval = 3000;

  function apply(data) {
    if (!data || typeof data !== "object") return;
    Object.keys(data).forEach(function (key) {
      var el = root.querySelector('[data-metric="' + key + '"]');
      if (el) el.textContent = String(data[key]);
    });
    var stamp = root.querySelector("[data-analytics-updated]");
    if (stamp) stamp.textContent = new Date().toLocaleTimeString();
  }

  function tick() {
    fetch(url, { headers: { Accept: "application/json", "X-Requested-With": "XMLHttpRequest" }, credentials: "same-origin" })
      .then(function (r) { return r.ok ? r.json() : null; })
      .then(apply)
      .catch(function () {});
  }

  tick();
  setInterval(tick, interval);
})();

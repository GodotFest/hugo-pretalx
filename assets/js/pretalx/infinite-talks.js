/**
 * pretalx infinite scroll for talks list — reveals more items when sentinel enters viewport.
 */
(function () {
  var PAGE_SIZE = 15;

  function init() {
    var list = document.getElementById("pretalx-talks-list");
    var sentinel = document.getElementById("pretalx-talks-sentinel");
    if (!list || !sentinel) return;

    var initial = parseInt(list.getAttribute("data-pretalx-initial"), 10) || PAGE_SIZE;
    var items = list.querySelectorAll(".pretalx-talks__item");
    var total = items.length;
    var revealed = Math.min(initial, total);

    // Hide items beyond initial
    for (var i = revealed; i < total; i++) {
      items[i].classList.add("pretalx-talks__item--more");
    }

    if (revealed >= total) {
      sentinel.style.display = "none";
      return;
    }

    var io = new IntersectionObserver(
      function (entries) {
        var e = entries[0];
        if (!e || !e.isIntersecting) return;
        var next = revealed + PAGE_SIZE;
        for (var j = revealed; j < next && j < total; j++) {
          items[j].classList.remove("pretalx-talks__item--more");
        }
        revealed = Math.min(next, total);
        if (revealed >= total) {
          sentinel.style.display = "none";
          io.unobserve(sentinel);
        }
      },
      { rootMargin: "400px 0px" }
    );
    io.observe(sentinel);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();

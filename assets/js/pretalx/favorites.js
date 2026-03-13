/**
 * pretalx favorites — localStorage-backed favorite talks and "Favorites only" filter.
 * Uses #pretalx-schedule-data for prefix (when present) to namespace storage: pretalx:favorites:{prefix}:v1
 */
(function () {
  var STORAGE_KEY_BASE = "pretalx:favorites:";
  var TOGGLE_KEY_BASE = "pretalx:favoritesOnly:";

  function getPrefix() {
    var el = document.getElementById("pretalx-schedule-data");
    if (!el) return "";
    try {
      var data = JSON.parse(el.textContent || "{}");
      return (data && data.prefix) ? data.prefix : "";
    } catch (_) {
      return "";
    }
  }

  function storageKey() {
    var p = getPrefix();
    return STORAGE_KEY_BASE + (p || "default") + ":v1";
  }

  function toggleStorageKey() {
    var p = getPrefix();
    return TOGGLE_KEY_BASE + (p || "default") + ":v1";
  }

  function getFavorites() {
    try {
      var raw = localStorage.getItem(storageKey());
      return raw ? JSON.parse(raw) : [];
    } catch (_) {
      return [];
    }
  }

  function setFavorites(list) {
    try {
      localStorage.setItem(storageKey(), JSON.stringify(list));
      window.dispatchEvent(new CustomEvent("pretalx:favorites:changed", { detail: { list: list } }));
    } catch (_) {}
  }

  function getFavoritesOnlyPref() {
    try {
      return localStorage.getItem(toggleStorageKey()) === "true";
    } catch (_) {
      return false;
    }
  }

  function setFavoritesOnlyPref(active) {
    try {
      localStorage.setItem(toggleStorageKey(), active ? "true" : "false");
    } catch (_) {}
  }

  function isFav(id) {
    return getFavorites().indexOf(id) !== -1;
  }

  function toggleFav(id) {
    var list = getFavorites();
    var idx = list.indexOf(id);
    if (idx >= 0) {
      list.splice(idx, 1);
    } else {
      list.push(id);
    }
    setFavorites(list);
  }

  function updateFavButtons() {
    var favs = getFavorites();
    document.querySelectorAll(".pretalx-fav-toggle[data-fav-id]").forEach(function (btn) {
      var id = btn.getAttribute("data-fav-id");
      var active = favs.indexOf(id) !== -1;
      btn.setAttribute("aria-pressed", active ? "true" : "false");
      btn.setAttribute("aria-label", active ? "Remove from favorites" : "Add to favorites");
      btn.classList.toggle("is-active", active);
    });
  }

  function bindFavButtons() {
    document.body.addEventListener("click", function (e) {
      var btn = e.target.closest(".pretalx-fav-toggle[data-fav-id]");
      if (!btn) return;
      e.preventDefault();
      e.stopPropagation();
      var id = btn.getAttribute("data-fav-id");
      toggleFav(id);
      updateFavButtons();
      applyFavoritesFilterIfActive();
    });
  }

  function hasAnyFavorites() {
    return getFavorites().length > 0;
  }

  function updateToggleVisibility() {
    var wrapper = document.getElementById("pretalx-favorites-only-toggle");
    var control = document.getElementById("pretalx-favorites-only");
    if (!wrapper || !control) return;
    wrapper.style.display = hasAnyFavorites() ? "" : "none";
    if (!hasAnyFavorites()) {
      control.setAttribute("aria-pressed", "false");
      setFavoritesOnlyPref(false);
    }
  }

  function applyFavoritesFilterIfActive() {
    var control = document.getElementById("pretalx-favorites-only");
    if (!control) return;
    var active = control.getAttribute("aria-pressed") === "true";
    var favs = getFavorites();

    // Schedule: .pretalx-schedule__entry containing .pretalx-talk-card[data-pretalx-fav-id] or special (no id)
    var schedule = document.querySelector("[data-pretalx-schedule]");
    if (schedule) {
      schedule.querySelectorAll(".pretalx-schedule__entry").forEach(function (entry) {
        var card = entry.querySelector(".pretalx-talk-card[data-pretalx-fav-id]");
        var special = entry.querySelector(".pretalx-schedule__special");
        if (special) {
          entry.classList.toggle("is-hidden", active);
          return;
        }
        if (!card) {
          entry.classList.remove("is-hidden");
          return;
        }
        var id = card.getAttribute("data-pretalx-fav-id");
        var show = !active || favs.indexOf(id) !== -1;
        entry.classList.toggle("is-hidden", !show);
      });

      // Hide empty slots when filtering
      schedule.querySelectorAll(".pretalx-schedule__slot").forEach(function (slot) {
        var visible = slot.querySelectorAll(".pretalx-schedule__entry:not(.is-hidden)").length;
        if (visible === 0 && active) {
          slot.setAttribute("hidden", "");
        } else {
          slot.removeAttribute("hidden");
        }
      });
    }

    // Talks list: .pretalx-talks__list .pretalx-talk-card[data-pretalx-fav-id]; hide wrapper .pretalx-talks__item
    var list = document.getElementById("pretalx-talks-list");
    if (list) {
      list.querySelectorAll(".pretalx-talk-card[data-pretalx-fav-id]").forEach(function (card) {
        var id = card.getAttribute("data-pretalx-fav-id");
        var show = !active || favs.indexOf(id) !== -1;
        var wrap = card.closest(".pretalx-talks__item") || card.closest("a") || card.closest("div") || card.parentElement;
        if (wrap) wrap.classList.toggle("is-hidden", !show);
      });
    }
  }

  function bindFavoritesOnlyToggle() {
    var btn = document.getElementById("pretalx-favorites-only");
    if (!btn) return;
    btn.addEventListener("click", function () {
      var current = btn.getAttribute("aria-pressed") === "true";
      var next = !current;
      btn.setAttribute("aria-pressed", next ? "true" : "false");
      setFavoritesOnlyPref(next);
      applyFavoritesFilterIfActive();
    });
  }

  function init() {
    bindFavButtons();
    bindFavoritesOnlyToggle();
    updateFavButtons();
    updateToggleVisibility();
    var control = document.getElementById("pretalx-favorites-only");
    if (control) {
      control.setAttribute("aria-pressed", getFavoritesOnlyPref() ? "true" : "false");
    }
    applyFavoritesFilterIfActive();
  }

  window.addEventListener("pretalx:favorites:changed", function () {
    updateFavButtons();
    updateToggleVisibility();
    applyFavoritesFilterIfActive();
  });

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();

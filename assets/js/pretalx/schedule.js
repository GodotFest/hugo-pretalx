/**
 * hugo-pretalx — Schedule interactivity
 *
 * Handles day tab switching and room filtering for the schedule component.
 * Works with the HTML structure produced by pretalx/schedule-timeline.html.
 *
 * No dependencies — vanilla JS, progressive enhancement.
 */
(function () {
  'use strict';

  document.addEventListener('DOMContentLoaded', function () {
    var schedules = document.querySelectorAll('[data-pretalx-schedule]');
    schedules.forEach(initSchedule);
  });

  function initSchedule(root) {
    initDayTabs(root);
    initRoomFilters(root);
  }

  /** Day tab switching */
  function initDayTabs(root) {
    var tabs = root.querySelectorAll('.pretalx-schedule__tab');
    if (tabs.length === 0) return;

    tabs.forEach(function (tab) {
      tab.addEventListener('click', function () {
        var day = tab.getAttribute('data-day');

        // Update tabs
        tabs.forEach(function (t) {
          var isActive = t.getAttribute('data-day') === day;
          t.classList.toggle('is-active', isActive);
          t.setAttribute('aria-selected', isActive ? 'true' : 'false');
        });

        // Show/hide day panels
        var panels = root.querySelectorAll('.pretalx-schedule__day');
        panels.forEach(function (panel) {
          var isActive = panel.getAttribute('data-day') === day;
          panel.classList.toggle('is-active', isActive);
          if (isActive) {
            panel.removeAttribute('hidden');
          } else {
            panel.setAttribute('hidden', '');
          }
        });
      });
    });
  }

  /** Room filter buttons */
  function initRoomFilters(root) {
    var filters = root.querySelectorAll('.pretalx-schedule__filter');
    if (filters.length === 0) return;

    filters.forEach(function (btn) {
      btn.addEventListener('click', function () {
        var room = btn.getAttribute('data-room');

        // Update filter buttons
        filters.forEach(function (f) {
          f.classList.toggle('is-active', f === btn);
        });

        // Show/hide entries
        var entries = root.querySelectorAll('.pretalx-schedule__entry');
        entries.forEach(function (entry) {
          if (room === 'all' || entry.getAttribute('data-room') === room) {
            entry.removeAttribute('hidden');
          } else {
            entry.setAttribute('hidden', '');
          }
        });

        // Hide empty time slots (all entries hidden)
        var slots = root.querySelectorAll('.pretalx-schedule__slot');
        slots.forEach(function (slot) {
          var visibleEntries = slot.querySelectorAll('.pretalx-schedule__entry:not([hidden])');
          if (visibleEntries.length === 0) {
            slot.setAttribute('hidden', '');
          } else {
            slot.removeAttribute('hidden');
          }
        });
      });
    });
  }
})();

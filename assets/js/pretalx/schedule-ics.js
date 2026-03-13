/**
 * pretalx schedule ICS export — reads #pretalx-schedule-data and offers download of calendar file.
 * Expects JSON: { baseURL, prefix, sessions: [{ start, title, duration, room, slug?, code? }], specials: [{ start, title, duration, room }] }
 */
(function () {
  function readEmbeddedData() {
    const el = document.getElementById("pretalx-schedule-data");
    if (!el) return { baseURL: "/", prefix: "", sessions: [], specials: [] };
    try {
      let parsed = JSON.parse(el.textContent || "{}");
      if (typeof parsed === "string") {
        try {
          parsed = JSON.parse(parsed);
        } catch (_) {}
      }
      return parsed && typeof parsed === "object"
        ? parsed
        : { baseURL: "/", prefix: "", sessions: [], specials: [] };
    } catch (e) {
      console.error("pretalx schedule-ics: failed to parse pretalx-schedule-data", e);
      return { baseURL: "/", prefix: "", sessions: [], specials: [] };
    }
  }

  function pad2(n) {
    return String(n).padStart(2, "0");
  }

  function parseIsoDate(isoString) {
    if (!isoString) return new Date(NaN);
    const s = String(isoString);
    const normalized = s.replace(/([+-]\d{2}):(\d{2})$/, "$1$2");
    return new Date(normalized);
  }

  function toIcsDate(date) {
    const y = date.getUTCFullYear();
    const m = pad2(date.getUTCMonth() + 1);
    const d = pad2(date.getUTCDate());
    const hh = pad2(date.getUTCHours());
    const mm = pad2(date.getUTCMinutes());
    const ss = pad2(date.getUTCSeconds());
    return `${y}${m}${d}T${hh}${mm}${ss}Z`;
  }

  function sanitize(text) {
    if (text == null) return "";
    const s = String(text);
    return s
      .replace(/\\/g, "\\\\")
      .replace(/\n/g, "\\n")
      .replace(/,/g, "\\,")
      .replace(/;/g, "\\;");
  }

  function makeUid(item, namespace) {
    const base = item.code || item.slug || `${item.title}-${item.start}`;
    return `${base}@${namespace}`;
  }

  function buildEvents({ items, baseURL, prefix, onlyFavorites, getFavorites }) {
    const favs = onlyFavorites && getFavorites ? new Set(getFavorites()) : null;
    const talksBase = prefix ? (baseURL.replace(/\/$/, "") + "/" + prefix + "/talks/") : baseURL + "talks/";
    const events = [];

    for (const it of items) {
      if (!it || !it.start || !it.title) continue;
      if (favs && it.slug && !favs.has(it.slug)) continue;
      if (favs && !it.slug && it.code && !favs.has(it.code)) continue;

      const start = parseIsoDate(it.start);
      if (!isFinite(start.getTime())) continue;
      const end = it.end ? parseIsoDate(it.end) : new Date(start.getTime() + (Number(it.duration) || 0) * 60000);
      const url = it.slug ? talksBase + it.slug + "/" : (it.code ? talksBase + it.code + "/" : "");

      const lines = [
        "BEGIN:VEVENT",
        "UID:" + sanitize(makeUid(it, "pretalx")),
        "DTSTAMP:" + toIcsDate(new Date()),
        "DTSTART:" + toIcsDate(start),
        "DTEND:" + toIcsDate(end),
        "SUMMARY:" + sanitize(it.title),
        it.room ? "LOCATION:" + sanitize(it.room) : null,
        url ? "URL:" + sanitize(url) : null,
      ].filter(Boolean);

      const descParts = [];
      if (it.speakers && Array.isArray(it.speakers) && it.speakers.length)
        descParts.push("Speakers: " + it.speakers.map(function (s) { return s.name || s; }).join(", "));
      if (it.duration) descParts.push("Duration: " + it.duration + " minutes");
      if (url) descParts.push(url);
      if (descParts.length) lines.push("DESCRIPTION:" + sanitize(descParts.join("\n")));

      lines.push("END:VEVENT");
      events.push(lines.join("\r\n"));
    }
    return events;
  }

  function generateIcs(data, onlyFavorites, getFavorites) {
    const baseURL = data.baseURL || "/";
    const prefix = data.prefix || "";
    const sessions = Array.isArray(data.sessions) ? data.sessions : [];
    const specials = Array.isArray(data.specials) ? data.specials : [];
    const items = sessions.concat(specials);

    const events = buildEvents({
      items: items,
      baseURL: baseURL,
      prefix: prefix,
      onlyFavorites: onlyFavorites,
      getFavorites: getFavorites,
    });

    const prodId = "-//pretalx//Schedule//EN";
    const header = [
      "BEGIN:VCALENDAR",
      "VERSION:2.0",
      "PRODID:" + prodId,
      "CALSCALE:GREGORIAN",
      "METHOD:PUBLISH",
    ].join("\r\n");
    const footer = "END:VCALENDAR";
    return [header, ...events, footer].join("\r\n");
  }

  function downloadIcs(filename, content) {
    const blob = new Blob([content], { type: "text/calendar;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(url);
  }

  function init() {
    const btn = document.getElementById("pretalx-download-ics");
    if (!btn) return;

    btn.addEventListener("click", function () {
      const data = readEmbeddedData();
      const filename = data.prefix ? "schedule-" + data.prefix + ".ics" : "schedule.ics";
      const ics = generateIcs(data, false, null);
      downloadIcs(filename, ics);
    });
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();

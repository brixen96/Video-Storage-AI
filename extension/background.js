// VideoStorageAI Companion Bridge - background event page.
// Holds a WebSocket to the local backend, reports tab state, answers
// commands (screenshot, snapshot, navigation, downloads, history, bookmarks).

// 127.0.0.1 NOT localhost: the API binds IPv4 loopback only, and Firefox resolves
// "localhost" to ::1 first for WebSocket attempts - which is refused silently
// (no server-side log either). IPv4 literal sidesteps the whole family question.
const DEFAULT_BRIDGE_URL = "ws://127.0.0.1:8080/api/v1/browser/ws";
const RECONNECT_BASE_MS = 2000;
const RECONNECT_MAX_MS = 30000;

let bridgeUrl = DEFAULT_BRIDGE_URL;
let ws = null;
let reconnectDelay = RECONNECT_BASE_MS;
let reconnectTimer = null;
let tabStateTimer = null;
let bookmarkTimer = null;

async function loadConfig() {
  try {
    const stored = await browser.storage.local.get("bridgeUrl");
    if (stored && stored.bridgeUrl) bridgeUrl = stored.bridgeUrl;
  } catch (e) {
    // storage unavailable - fall back to default URL
  }
}

function send(obj) {
  if (ws && ws.readyState === WebSocket.OPEN) {
    try {
      ws.send(JSON.stringify(obj));
      return true;
    } catch (e) {
      // fall through to reconnect logic
    }
  }
  return false;
}

function scheduleReconnect() {
  if (reconnectTimer) return;
  reconnectTimer = setTimeout(() => {
    reconnectTimer = null;
    connect();
  }, reconnectDelay);
  reconnectDelay = Math.min(reconnectDelay * 1.5, RECONNECT_MAX_MS);
}

function connect() {
  if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
    return;
  }
  try {
    ws = new WebSocket(bridgeUrl);
  } catch (e) {
    scheduleReconnect();
    return;
  }

  ws.onopen = () => {
    reconnectDelay = RECONNECT_BASE_MS;
    console.log("[vsai] bridge connected to " + bridgeUrl);
    send({
      type: "hello",
      data: {
        version: browser.runtime.getManifest().version,
        ua: navigator.userAgent,
      },
    });
    pushTabState();
    pushBookmarks();
  };

  ws.onmessage = (event) => {
    let env = null;
    try {
      env = JSON.parse(event.data);
    } catch (e) {
      return;
    }
    if (env && env.type === "command" && env.id) {
      handleCommand(env.id, env.action, env.args || {});
    }
  };

  ws.onerror = () => {
    console.warn("[vsai] bridge error on " + bridgeUrl);
    try {
      ws.close();
    } catch (e) {
      // already closing
    }
  };

  ws.onclose = (event) => {
    console.warn("[vsai] bridge closed (code " + (event && event.code) + ") - reconnecting");
    ws = null;
    scheduleReconnect();
  };
}

// ---- command handling (request/reply with correlation id) ----

async function handleCommand(id, action, args) {
  let data = null;
  let error = null;
  try {
    data = await runAction(action, args);
  } catch (e) {
    error = String((e && e.message) || e);
  }
  if (error !== null) {
    send({ type: "result", id, ok: false, error });
  } else {
    send({ type: "result", id, ok: true, data: data === undefined ? null : data });
  }
}

async function getActiveTab() {
  // Multi-monitor/multi-window: tabs.query({active:true}) matches EVERY normal
  // window, and the first hit may be a window on another monitor. Anchor to the
  // most recently focused window - the one Brix is actually working in.
  try {
    const win = await browser.windows.getLastFocused({
      populate: true,
      windowTypes: ["normal"],
    });
    if (win && win.tabs) {
      const tab = win.tabs.find((t) => t.active);
      if (tab) return tab;
    }
  } catch (e) {
    // fall through to the plain query
  }
  const tabs = await browser.tabs.query({ active: true, windowType: "normal" });
  if (!tabs || tabs.length === 0) {
    throw new Error("No active Firefox tab");
  }
  return tabs[0];
}

function isPrivileged(url) {
  return !url || /^(about|moz-extension|chrome|resource|view-source):/.test(url);
}

// send a message to the content script of the active tab, injecting it first
// if that page was loaded before the extension was installed.
async function sendToActiveTab(type, payload) {
  const tab = await getActiveTab();
  if (isPrivileged(tab.url)) {
    throw new Error("This page (" + (tab.url || "unknown") + ") cannot be scripted - browser-internal pages are off limits");
  }
  const msg = Object.assign({ type }, payload);
  try {
    const reply = await browser.tabs.sendMessage(tab.id, msg);
    return Object.assign({ url: tab.url, title: tab.title || "" }, reply || {});
  } catch (e) {
    await browser.scripting.executeScript({ target: { tabId: tab.id }, files: ["content.js"] });
    const reply = await browser.tabs.sendMessage(tab.id, msg);
    return Object.assign({ url: tab.url, title: tab.title || "" }, reply || {});
  }
}

async function runAction(action, args) {
  switch (action) {
    case "screenshot": {
      const tab = await getActiveTab();
      if (isPrivileged(tab.url)) {
        throw new Error("Cannot screenshot browser-internal page: " + tab.url);
      }
      const image = await browser.tabs.captureVisibleTab(tab.windowId, { format: "png" });
      return { image, url: tab.url, title: tab.title || "" };
    }

    case "snapshot":
      return await sendToActiveTab("vsai_snapshot", {});

    case "navigate": {
      const url = String(args.url || "").trim();
      if (!url) throw new Error("navigate: url required");
      const tab = await getActiveTab();
      await browser.tabs.update(tab.id, { url });
      return { navigating: url, tab_id: tab.id };
    }

    case "history_nav": {
      const tab = await getActiveTab();
      if (args.direction === "back") {
        await browser.tabs.goBack(tab.id);
        return { did: "back" };
      }
      if (args.direction === "forward") {
        await browser.tabs.goForward(tab.id);
        return { did: "forward" };
      }
      if (args.direction === "reload") {
        await browser.tabs.reload(tab.id);
        return { did: "reload" };
      }
      throw new Error("history_nav: direction must be back, forward or reload");
    }

    case "click":
      return await sendToActiveTab("vsai_click", { selector: args.selector });

    case "type":
      return await sendToActiveTab("vsai_type", {
        selector: args.selector,
        text: args.text,
        submit: !!args.submit,
        keep_value: !!args.keep_value,
      });

    case "scroll":
      return await sendToActiveTab("vsai_scroll", {
        direction: args.direction,
        amount: args.amount,
      });

    case "downloads": {
      const limit = Math.min(Math.max(parseInt(args.limit, 10) || 10, 1), 50);
      // Firefox implements downloads.search (Chrome uses downloads.query)
      const items = await browser.downloads.search({});
      items.sort((a, b) => (b.startTime || 0) - (a.startTime || 0));
      return items.slice(0, limit).map((d) => ({
        filename: d.filename || "",
        url: d.url || "",
        state: d.state || "",
        error: d.error || null,
        bytes_received: d.bytesReceived || 0,
        total_bytes: d.totalBytes === -1 ? null : d.totalBytes,
        started: d.startTime || null,
        ended: d.endTime || null,
        exists: !!d.exists,
      }));
    }

    case "history": {
      const limit = Math.min(Math.max(parseInt(args.limit, 10) || 20, 1), 50);
      const days = Math.min(Math.max(parseInt(args.days, 10) || 7, 1), 365);
      const items = await browser.history.search({
        text: String(args.text || ""),
        startTime: Date.now() - days * 86400000,
        maxResults: limit,
      });
      return items.map((h) => ({
        title: h.title || "",
        url: h.url || "",
        visit_time: h.lastVisitTime || null,
      }));
    }

    case "bookmarks_get":
      return await collectBookmarks();

    default:
      throw new Error("Unknown browser action: " + action);
  }
}

// ---- bookmarks (flatten tree, newest first) ----

async function collectBookmarks() {
  const tree = await browser.bookmarks.getTree();
  const out = [];
  function walk(nodes) {
    for (const n of nodes) {
      if (n.url) {
        out.push({ title: n.title || "", url: n.url, added: n.dateAdded || 0 });
      }
      if (n.children) walk(n.children);
    }
  }
  walk(tree);
  out.sort((a, b) => b.added - a.added);
  return out.slice(0, 3000);
}

function pushBookmarks() {
  if (bookmarkTimer) return;
  bookmarkTimer = setTimeout(async () => {
    bookmarkTimer = null;
    try {
      const list = await collectBookmarks();
      send({ type: "bookmarks", data: list });
    } catch (e) {
      // bookmarks API unavailable - cached copy on the backend stays as-is
    }
  }, 1000);
}

// ---- tab state (debounced push on every change) ----

function pushTabState() {
  if (tabStateTimer) return;
  tabStateTimer = setTimeout(async () => {
    tabStateTimer = null;
    if (!ws || ws.readyState !== WebSocket.OPEN) return;
    try {
      // Only the focused window's tab counts as "active" - other windows have
      // their own active tabs and would otherwise race for the slot.
      let focusedWindowId = null;
      try {
        const w = await browser.windows.getLastFocused({ windowTypes: ["normal"] });
        if (w) focusedWindowId = w.id;
      } catch (e) {
        // keep null - fall back to marking all active tabs
      }
      const tabs = await browser.tabs.query({});
      const infos = tabs
        .filter((t) => t.id !== undefined)
        .map((t) => ({
          tab_id: t.id,
          url: t.url || "",
          title: t.title || "",
          fav_icon: t.favIconUrl || "",
          active: !!t.active && (focusedWindowId === null || t.windowId === focusedWindowId),
        }));
      const active = infos.find((i) => i.active) || null;
      send({ type: "tab_state", data: { tabs: infos, active } });
    } catch (e) {
      // tabs query failed (rare) - next event will retry
    }
  }, 300);
}

// ---- wiring ----

browser.tabs.onActivated.addListener(pushTabState);
browser.tabs.onUpdated.addListener((id, info, tab) => {
  if (info.status === "complete" || info.url !== undefined || info.title !== undefined) {
    pushTabState();
  }
});
browser.tabs.onRemoved.addListener(pushTabState);

browser.bookmarks.onCreated.addListener(pushBookmarks);
browser.bookmarks.onRemoved.addListener(pushBookmarks);
browser.bookmarks.onChanged.addListener(pushBookmarks);
browser.bookmarks.onMoved.addListener(pushBookmarks);

// keepalive: the MV3 event page can suspend; the alarm wakes it and repairs
// the connection. Also acts as a safety net if the socket died silently.
browser.alarms.create("vsai-bridge-keepalive", { periodInMinutes: 0.5 });
browser.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name !== "vsai-bridge-keepalive") return;
  if (!ws || ws.readyState !== WebSocket.OPEN) {
    connect();
  } else {
    // cheap round trip keeps the socket obviously alive for the backend
    send({ type: "hello", data: { version: browser.runtime.getManifest().version } });
  }
});

browser.storage.onChanged.addListener((changes, area) => {
  if (area === "local" && changes.bridgeUrl) {
    bridgeUrl = (changes.bridgeUrl.newValue || DEFAULT_BRIDGE_URL).trim();
    if (ws) {
      try {
        ws.close();
      } catch (e) {
        // already closing
      }
      ws = null;
    }
    connect();
  }
});

loadConfig().then(connect);

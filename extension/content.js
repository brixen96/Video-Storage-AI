// VideoStorageAI Companion Bridge - content script.
// Builds structured page snapshots with usable CSS selectors and performs
// DOM-level actions (click, type, scroll) on behalf of the AI Companion.

(() => {
  if (window.__vsaiSnapInstalled) return;
  window.__vsaiSnapInstalled = true;

  const MAX_ELEMENTS = 600;
  const MAX_TEXT = 120;
  const MAX_BODY_TEXT = 5000;

  function uniqueSelector(el) {
    if (el.id && /^[A-Za-z][\w-]*$/.test(el.id)) {
      const probe = document.querySelectorAll("#" + CSS.escape(el.id));
      if (probe.length === 1) return "#" + CSS.escape(el.id);
    }
    const parts = [];
    let node = el;
    while (node && node.nodeType === 1 && node !== document.body && node !== document.documentElement) {
      let seg = node.tagName.toLowerCase();
      if (node.id && /^[A-Za-z][\w-]*$/.test(node.id)) {
        parts.unshift("#" + CSS.escape(node.id));
        break;
      }
      const parent = node.parentElement;
      if (parent) {
        const same = Array.from(parent.children).filter((c) => c.tagName === node.tagName);
        if (same.length > 1) seg += ":nth-of-type(" + (same.indexOf(node) + 1) + ")";
      }
      parts.unshift(seg);
      node = parent;
    }
    if (parts.length === 0) return "body";
    return parts.join(" > ");
  }

  function isVisible(el) {
    if (!el.getClientRects) return false;
    if (el.tagName === "INPUT" && el.type === "hidden") return false;
    const rects = el.getClientRects();
    if (rects.length === 0) return false;
    const r = rects[0];
    return r.width > 0 || r.height > 0 || el.tagName === "INPUT";
  }

  function shortText(el) {
    const raw = (el.innerText || el.value || el.getAttribute("aria-label") || el.title || el.alt || "")
      .replace(/\s+/g, " ")
      .trim();
    return raw.slice(0, MAX_TEXT);
  }

  function describe(el) {
    const entry = {
      tag: el.tagName.toLowerCase(),
      selector: uniqueSelector(el),
      text: shortText(el),
    };
    if (el.getAttribute("role")) entry.role = el.getAttribute("role");
    if (el.tagName === "A" && el.href) entry.href = el.href;
    if (el.tagName === "INPUT") {
      entry.type = el.type || "text";
      if (el.name) entry.name = el.name;
      if (el.placeholder) entry.placeholder = el.placeholder.slice(0, MAX_TEXT);
      if (el.type === "checkbox" || el.type === "radio") entry.checked = !!el.checked;
    }
    if (el.tagName === "TEXTAREA" && el.placeholder) entry.placeholder = el.placeholder.slice(0, MAX_TEXT);
    if (el.disabled) entry.disabled = true;
    return entry;
  }

  function snapshot() {
    const selector =
      "h1, h2, h3, h4, h5, h6, a[href], button, input, textarea, select, summary, " +
      "[role=button], [role=link], [role=tab], [role=menuitem], [role=checkbox], [role=radio], " +
      "[role=textbox], [role=searchbox], [role=navigation], [role=main], [onclick], [tabindex]";
    const found = document.querySelectorAll(selector);
    const elements = [];
    const seen = new Set();
    for (const el of found) {
      if (elements.length >= MAX_ELEMENTS) break;
      if (el.closest("script, style, noscript, template")) continue;
      if (!isVisible(el)) continue;
      const key = el.tagName + "|" + (el.id || "") + "|" + (el.className || "");
      if (seen.has(key)) continue;
      seen.add(key);
      elements.push(describe(el));
    }

    const metaDesc = document.querySelector('meta[name="description"]');
    let bodyText = "";
    try {
      bodyText = (document.body && document.body.innerText ? document.body.innerText : "")
        .replace(/\n{3,}/g, "\n\n")
        .trim()
        .slice(0, MAX_BODY_TEXT);
    } catch (e) {
      bodyText = "";
    }

    return {
      url: location.href,
      title: document.title || "",
      description: metaDesc ? metaDesc.content || "" : "",
      lang: document.documentElement.lang || "",
      element_count: elements.length,
      elements,
      body_text: bodyText,
    };
  }

  function resolve(selector) {
    if (!selector) throw new Error("selector is required");
    const el = document.querySelector(selector);
    if (!el) {
      throw new Error('Element not found for selector "' + selector + '" - the page changed, run browser_read again for fresh selectors');
    }
    return el;
  }

  function doClick(selector) {
    const el = resolve(selector);
    try {
      el.scrollIntoView({ block: "center", inline: "center" });
    } catch (e) {
      // not scrollable - harmless
    }
    if (typeof el.focus === "function") {
      try {
        el.focus();
      } catch (e) {
        // focusless element
      }
    }
    const opts = { bubbles: true, cancelable: true, view: window, button: 0 };
    el.dispatchEvent(new MouseEvent("pointerdown", opts));
    el.dispatchEvent(new MouseEvent("mousedown", opts));
    el.dispatchEvent(new MouseEvent("mouseup", opts));
    el.dispatchEvent(new MouseEvent("click", opts));
    return { clicked: true, tag: el.tagName.toLowerCase(), text: shortText(el) };
  }

  function doType({ selector, text, submit, keep_value }) {
    const el = resolve(selector);
    if (el.tagName !== "INPUT" && el.tagName !== "TEXTAREA" && el.tagName !== "SELECT") {
      throw new Error("browser_type only works on input/textarea/select elements (got " + el.tagName.toLowerCase() + ")");
    }
    if (typeof el.focus === "function") el.focus();

    let proto = HTMLInputElement.prototype;
    if (el.tagName === "TEXTAREA") proto = HTMLTextAreaElement.prototype;
    if (el.tagName === "SELECT") proto = HTMLSelectElement.prototype;
    const desc = Object.getOwnPropertyDescriptor(proto, "value");
    const typed = String(text == null ? "" : text);
    const next = keep_value && el.value ? el.value + typed : typed;
    if (desc && desc.set) desc.set.call(el, next);
    else el.value = next;

    el.dispatchEvent(new Event("input", { bubbles: true }));
    el.dispatchEvent(new Event("change", { bubbles: true }));

    if (submit) {
      const form = el.form;
      if (form && typeof form.requestSubmit === "function") {
        form.requestSubmit();
      } else {
        throw new Error("Typed the text but this field has no form to submit - ask Brix what he wants next");
      }
    }
    return { typed: true, length: next.length, submitted: !!submit };
  }

  function doScroll({ direction, amount }) {
    const a = Math.min(Math.max(parseInt(amount, 10) || 700, 1), 20000);
    if (direction === "top") {
      window.scrollTo({ top: 0, behavior: "instant" });
    } else if (direction === "bottom") {
      window.scrollTo({ top: document.documentElement.scrollHeight, behavior: "instant" });
    } else if (direction === "up") {
      window.scrollBy({ top: -a, behavior: "instant" });
    } else if (direction === "down") {
      window.scrollBy({ top: a, behavior: "instant" });
    } else {
      throw new Error("direction must be down, up, top or bottom");
    }
    return { y: Math.round(window.scrollY), max_y: document.documentElement.scrollHeight };
  }

  browser.runtime.onMessage.addListener((msg) => {
    if (!msg || typeof msg.type !== "string") return undefined;
    try {
      switch (msg.type) {
        case "vsai_snapshot":
          return Promise.resolve(snapshot());
        case "vsai_click":
          return Promise.resolve(doClick(msg.selector));
        case "vsai_type":
          return Promise.resolve(doType(msg));
        case "vsai_scroll":
          return Promise.resolve(doScroll(msg));
        default:
          return undefined;
      }
    } catch (e) {
      return Promise.reject(new Error(String((e && e.message) || e)));
    }
  });
})();

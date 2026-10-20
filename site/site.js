(function () {
  "use strict";

  /* Mobile menu */
  var toggle = document.getElementById("menu-toggle");
  var menu = document.getElementById("mobile-menu");

  function setMenu(open) {
    if (!toggle || !menu) return;
    toggle.setAttribute("aria-expanded", open ? "true" : "false");
    toggle.setAttribute("aria-label", toggle.getAttribute(open ? "data-label-close" : "data-label-open") || "");
    menu.hidden = !open;
  }

  if (toggle && menu) {
    toggle.addEventListener("click", function () {
      setMenu(toggle.getAttribute("aria-expanded") !== "true");
    });

    document.addEventListener("keydown", function (event) {
      if (event.key === "Escape" && toggle.getAttribute("aria-expanded") === "true") {
        setMenu(false);
        toggle.focus();
      }
    });

    document.addEventListener("click", function (event) {
      if (toggle.getAttribute("aria-expanded") !== "true") return;
      if (!menu.contains(event.target) && !toggle.contains(event.target)) setMenu(false);
    });

    menu.querySelectorAll("a").forEach(function (link) {
      link.addEventListener("click", function () { setMenu(false); });
    });

    var wide = window.matchMedia("(min-width: 961px)");
    if (wide.addEventListener) {
      wide.addEventListener("change", function (event) {
        if (event.matches) setMenu(false);
      });
    }
  }

  /* Installation tabs */
  var tabs = Array.prototype.slice.call(document.querySelectorAll('[role="tab"]'));
  if (tabs.length) {
    function selectTab(tab, moveFocus) {
      tabs.forEach(function (item) {
        var selected = item === tab;
        item.setAttribute("aria-selected", selected ? "true" : "false");
        item.tabIndex = selected ? 0 : -1;
        var panel = document.getElementById(item.getAttribute("aria-controls"));
        if (panel) panel.hidden = !selected;
      });
      if (moveFocus) tab.focus();
    }

    tabs.forEach(function (tab, index) {
      tab.addEventListener("click", function () { selectTab(tab, false); });
      tab.addEventListener("keydown", function (event) {
        var next = null;
        if (event.key === "ArrowRight") next = (index + 1) % tabs.length;
        else if (event.key === "ArrowLeft") next = (index - 1 + tabs.length) % tabs.length;
        else if (event.key === "Home") next = 0;
        else if (event.key === "End") next = tabs.length - 1;
        if (next === null) return;
        event.preventDefault();
        selectTab(tabs[next], true);
      });
    });
  }

  /* Copy commands */
  var copyButton = document.getElementById("copy-commands");
  var copyLabel = document.getElementById("copy-label");
  var copyStatus = document.getElementById("copy-status");
  var resetTimer = null;

  function activePanel() {
    var selected = tabs.filter(function (tab) {
      return tab.getAttribute("aria-selected") === "true";
    })[0];
    return selected ? document.getElementById(selected.getAttribute("aria-controls")) : null;
  }

  function commandText() {
    var panel = activePanel();
    if (!panel) return "";
    return Array.prototype.slice.call(panel.querySelectorAll("code")).map(function (code) {
      return code.textContent.trim();
    }).filter(Boolean).join("\n");
  }

  function legacyCopy(text) {
    var area = document.createElement("textarea");
    area.value = text;
    area.setAttribute("readonly", "");
    area.style.position = "fixed";
    area.style.top = "-1000px";
    area.style.opacity = "0";
    document.body.appendChild(area);
    area.select();
    var ok = false;
    try { ok = document.execCommand("copy"); } catch (error) { ok = false; }
    document.body.removeChild(area);
    return ok;
  }

  function report(state) {
    copyButton.setAttribute("data-state", state === "empty" ? "failed" : state);
    copyLabel.textContent = copyButton.getAttribute(
      state === "copied" ? "data-label-copied" : state === "failed" || state === "empty" ? "data-label-failed" : "data-label-copy"
    ) || "";
    if (copyStatus) copyStatus.textContent = copyButton.getAttribute(
      state === "copied" ? "data-status-copied" : state === "empty" ? "data-status-empty" : "data-status-failed"
    ) || "";
    window.clearTimeout(resetTimer);
    resetTimer = window.setTimeout(function () {
      copyButton.removeAttribute("data-state");
      copyLabel.textContent = copyButton.getAttribute("data-label-copy") || "";
    }, state === "failed" ? 5000 : 2200);
  }

  if (copyButton && copyLabel) {
    copyButton.addEventListener("click", function () {
      var text = commandText();
      if (!text) {
        report("empty");
        return;
      }
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(function () {
          report("copied");
        }).catch(function () {
          if (legacyCopy(text)) report("copied");
          else report("failed");
        });
      } else if (legacyCopy(text)) {
        report("copied");
      } else {
        report("failed");
      }
    });
  }
})();
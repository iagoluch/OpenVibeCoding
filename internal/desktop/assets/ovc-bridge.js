// OVC desktop bridge alias.
//
// Wails v3 beta.30's runtime is a pure ES module: it never assigns
// window.wails, and its Call interface exists only as a module export, so
// the page's desktopBridge() would find nothing. This module — served by the
// shell itself (same origin, so the daemon CSP's script-src 'self' allows
// both) and loaded right after the wails runtime — imports that export and
// installs it on window.wails, so the page contract
// (window.wails.Call.ByName) works unchanged.
//
// Wails v3 also derives binding names from the full Go package path, so the
// bound method registers as "<package>.UI.PickFolder". The shipped page pins
// the short contract name "desktop.UI.PickFolder" and the Go side owns the
// mismatch, so right after the install the alias rewrites exactly that one
// name to the real FQN.
//
// The alias cannot create capability: the Call interface only reaches
// methods the Go process already bound, the install is a merge (anything
// pre-existing on window.wails or its Call is kept, never clobbered), every
// other name passes through untouched, and the module is inert when the
// runtime module cannot be imported (a plain browser never loads it; a
// failed import leaves window.wails unset and the picker hidden).
import { Call as runtimeCall } from "/wails/runtime.js";

(function () {
  "use strict";
  var CONTRACT = "desktop.UI.PickFolder";
  var FQN = "__BINDING_FQN__";
  var wails = window.wails || {};
  // Merge the runtime's Call interface into whatever Call already exists:
  // pre-existing members win, missing ones are installed.
  var call = {};
  for (var had in wails.Call) call[had] = wails.Call[had];
  for (var name in runtimeCall) if (!(name in call)) call[name] = runtimeCall[name];
  if (typeof call.ByName === "function") {
    var real = call.ByName.bind(call);
    call.ByName = function (name) {
      var rest = Array.prototype.slice.call(arguments, 1);
      return real.apply(null, [name === CONTRACT ? FQN : name].concat(rest));
    };
  }
  var wrapped = {};
  for (var key in wails) wrapped[key] = wails[key];
  wrapped.Call = call;
  window.wails = wrapped;
})();

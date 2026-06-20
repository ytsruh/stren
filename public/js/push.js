// /js/push.js — wires the profile page "Notifications" card to the
// browser's Push API. Behaviour is driven by the data attributes on
// #push-banner:
//
//   data-state     "disabled" | "enabled" | "unsupported"
//   data-vapid-key URL-safe base64 of the server's VAPID public key
//
// The script is defensive by design: if any precondition fails (no
// service worker, no Notification permission, no PushManager support)
// it degrades to the "unsupported" state and shows a Re-check button
// rather than failing silently.
//
// A single <input role="switch"> drives enable/disable. The change
// handler routes to doSubscribe or doUnsubscribe, both of which
// distinguish the various error paths (NotAllowedError, AbortError,
// network, malformed subscription, etc.) and roll back partial
// success (e.g. browser-side subscription followed by a failed server
// POST) so the UI never lies about the user's state.
//
// IMPORTANT: the change listener is attached synchronously, BEFORE
// any awaits. A previous version attached it after awaiting
// serviceWorker.ready, which could hang for many seconds after a
// fresh deploy or stale SW and left the user clicking a dead switch.

(function () {
  'use strict';

  function $(id) {
    return document.getElementById(id);
  }

  function urlB64ToUint8Array(base64String) {
    var padding = '='.repeat((4 - (base64String.length % 4)) % 4);
    var base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
    var raw = atob(base64);
    var output = new Uint8Array(raw.length);
    for (var i = 0; i < raw.length; i++) {
      output[i] = raw.charCodeAt(i);
    }
    return output;
  }

  function escapeHtml(s) {
    return String(s).replace(/[&<>"']/g, function (c) {
      return ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c];
    });
  }

  function toast(message, kind) {
    var toaster = $('toaster');
    if (!toaster) return;
    var el = document.createElement('div');
    el.className = 'toast';
    el.setAttribute('role', 'status');
    el.setAttribute('aria-atomic', 'true');
    el.setAttribute('aria-hidden', 'false');
    el.setAttribute('data-category', kind || 'info');
    el.innerHTML = '<div class="toast-content"><section><h2>' + escapeHtml(message.title || '') + '</h2>' +
      (message.body ? '<p>' + escapeHtml(message.body) + '</p>' : '') +
      '</section></div>';
    toaster.appendChild(el);
    setTimeout(function () { if (el.parentNode) el.parentNode.removeChild(el); }, 5000);
  }

  function setSpinner(show) {
    var spinner = $('push-spinner');
    if (spinner) spinner.hidden = !show;
  }

  function setSwitchDisabled(disabled) {
    var toggle = $('push-toggle');
    if (toggle) toggle.disabled = disabled;
  }

  function setSwitchState(state) {
    var banner = $('push-banner');
    var content = $('push-content');
    var unsupported = $('push-unsupported');
    var toggle = $('push-toggle');
    var status = $('push-status');
    if (banner) banner.setAttribute('data-state', state);
    if (content) content.hidden = state === 'unsupported';
    if (unsupported) unsupported.hidden = state !== 'unsupported';
    if (toggle && state !== 'unsupported') {
      toggle.checked = state === 'enabled';
    }
    if (status) {
      var label = state === 'enabled' ? 'Enabled'
                : state === 'unsupported' ? 'Unavailable'
                : 'Disabled';
      status.textContent = 'Status: ' + label;
    }
  }

  function setUnsupportedReason(message) {
    var el = $('push-unsupported-reason');
    if (el) el.textContent = message;
  }

  function detectSupport() {
    if (!('serviceWorker' in navigator)) {
      return { ok: false, reason: 'This browser does not support service workers.' };
    }
    if (!('PushManager' in window)) {
      return { ok: false, reason: 'This browser does not support the Push API.' };
    }
    if (typeof Notification === 'undefined') {
      return { ok: false, reason: 'This browser does not support notifications.' };
    }
    if (Notification.permission === 'denied') {
      return { ok: false, reason: 'Notifications are blocked. Update your browser settings to allow them.' };
    }
    return { ok: true };
  }

  // getRegistration races serviceWorker.ready against a timeout so a
  // failed or slow SW registration can't hang the change handler
  // forever.
  function getRegistration(timeoutMs) {
    return new Promise(function (resolve, reject) {
      var timer = setTimeout(function () {
        reject(new Error('Service worker not ready. Try reloading the page.'));
      }, timeoutMs || 5000);
      navigator.serviceWorker.ready.then(
        function (reg) { clearTimeout(timer); resolve(reg); },
        function (err) { clearTimeout(timer); reject(err); }
      );
    });
  }

  async function postJSON(url, method, body) {
    var resp = await fetch(url, {
      method: method,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body || {})
    });
    if (!resp.ok) {
      var text;
      try { text = await resp.text(); } catch (_) { text = ''; }
      throw new Error(text || ('HTTP ' + resp.status));
    }
    return resp;
  }

  async function doSubscribe(vapidKey) {
    if (typeof Notification !== 'undefined' && Notification.permission === 'denied') {
      throw new Error('Notifications are blocked. Update your browser settings to allow them.');
    }

    var permission;
    try {
      permission = await Notification.requestPermission();
    } catch (e) {
      throw new Error('Could not request notification permission.');
    }
    if (permission !== 'granted') {
      throw new Error('Permission not granted.');
    }

    var reg = await getRegistration();

    var sub;
    try {
      sub = await reg.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlB64ToUint8Array(vapidKey)
      });
    } catch (e) {
      if (e && e.name === 'NotAllowedError') {
        throw new Error('Permission denied.');
      }
      if (e && e.name === 'AbortError') {
        throw new Error('Subscription cancelled.');
      }
      if (e && e.name === 'InvalidStateError') {
        throw new Error('Service worker not active. Try again.');
      }
      throw new Error('Could not subscribe: ' + ((e && e.message) || 'unknown error'));
    }

    var json;
    try { json = sub.toJSON(); } catch (e) { json = null; }
    if (!json || !json.endpoint || !json.keys || !json.keys.p256dh || !json.keys.auth) {
      try { await sub.unsubscribe(); } catch (_) {}
      throw new Error('Subscription malformed. Please try again.');
    }

    // If the server POST fails, the browser still has a subscription
    // but the server will never send anything. Roll it back so the
    // UI matches reality and the user can retry.
    try {
      await postJSON('/api/push/subscribe', 'POST', {
        endpoint: json.endpoint,
        p256dh: json.keys.p256dh,
        auth: json.keys.auth
      });
    } catch (e) {
      try { await sub.unsubscribe(); } catch (_) {}
      throw new Error('Could not save subscription to server. Check your connection and try again.');
    }

    return true;
  }

  async function doUnsubscribe() {
    var reg;
    try { reg = await getRegistration(); }
    catch (e) { throw e; }

    var sub;
    try {
      sub = await reg.pushManager.getSubscription();
    } catch (e) {
      throw new Error('Could not read subscription state.');
    }

    var endpoint = sub ? sub.endpoint : null;

    // Best-effort browser unsubscribe. If it throws, log and
    // continue so we still try to clean up the server side.
    if (sub) {
      try { await sub.unsubscribe(); }
      catch (e) { console.warn('push: browser unsubscribe failed', e); }
    }

    if (!endpoint) {
      return true;
    }

    try {
      await postJSON('/api/push/unsubscribe', 'DELETE', { endpoint: endpoint });
    } catch (e) {
      throw new Error('Browser unsubscribed, but server cleanup failed. Please try again later.');
    }

    return true;
  }

  // userInteracting is set to true while the change handler is in
  // flight, so the background reconciliation step knows to leave
  // the toggle alone.
  var userInteracting = false;

  async function init() {
    var banner = $('push-banner');
    if (!banner) return;

    var vapidKey = banner.getAttribute('data-vapid-key') || '';
    var toggle = $('push-toggle');
    var recheck = $('push-recheck');

    // === Attach listeners synchronously ===
    // These must be wired BEFORE any awaits, otherwise a slow or
    // hung serviceWorker.ready leaves the user clicking a dead
    // switch.

    if (recheck) {
      recheck.addEventListener('click', function () {
        var s = detectSupport();
        if (s.ok) {
          setSwitchState('disabled');
        } else {
          setUnsupportedReason(s.reason);
        }
      });
    }

    if (toggle) {
      toggle.addEventListener('change', async function () {
        if (userInteracting) return;
        var wantsOn = toggle.checked;
        userInteracting = true;
        setSpinner(true);
        setSwitchDisabled(true);
        try {
          if (wantsOn) {
            await doSubscribe(vapidKey);
            setSwitchState('enabled');
            toast({ title: 'Notifications enabled', body: "You'll receive admin messages." }, 'success');
          } else {
            await doUnsubscribe();
            setSwitchState('disabled');
            toast({ title: 'Notifications disabled' }, 'success');
          }
        } catch (e) {
          console.error('push: toggle failed', e);
          setSwitchState(wantsOn ? 'disabled' : 'enabled');
          toast({ title: 'Notifications', body: (e && e.message) || 'Operation failed.' }, 'error');
        } finally {
          setSpinner(false);
          setSwitchDisabled(false);
          userInteracting = false;
        }
      });
    }

    // === Reconciliation runs in the background ===
    // Once it resolves it updates the visual state to match the
    // browser's actual subscription. It bails out if the user is
    // mid-toggle so we don't clobber their interaction.
    (async function reconcile() {
      var support = detectSupport();
      if (!support.ok) {
        setSwitchState('unsupported');
        setUnsupportedReason(support.reason);
        return;
      }
      if (!vapidKey) {
        setSwitchState('unsupported');
        setUnsupportedReason('Server is missing its VAPID key.');
        return;
      }
      try {
        var reg = await navigator.serviceWorker.ready;
        if (userInteracting) return;
        var existing = await reg.pushManager.getSubscription();
        if (userInteracting) return;
        setSwitchState(existing ? 'enabled' : 'disabled');
      } catch (e) {
        console.warn('push: reconciliation failed', e);
        if (userInteracting) return;
        var initial = banner.getAttribute('data-state') || 'disabled';
        setSwitchState(initial);
      }
    })();
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();

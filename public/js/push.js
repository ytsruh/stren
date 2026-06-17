// /js/push.js — wires the profile page "Notifications" banner to the
// browser's Push API. Behaviour is driven by the data attributes on
// #push-banner:
//
//   data-state     "disabled" | "enabled" | "unsupported"
//   data-vapid-key URL-safe base64 of the server's VAPID public key
//
// The script is defensive by design: if any precondition fails (no
// service worker, no Notification permission, no PushManager support)
// it degrades to the "unsupported" state and shows a Re-check button
// rather than failing silently. This keeps the banner visible (and
// honest) on browsers or configurations that cannot subscribe.

(function () {
  'use strict';

  function $(id) {
    return document.getElementById(id);
  }

  function urlB64ToUint8Array(base64String) {
    // The VAPID public key is base64url-no-pad. Convert to a Uint8Array
    // for applicationServerKey.
    var padding = '='.repeat((4 - (base64String.length % 4)) % 4);
    var base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
    var raw = atob(base64);
    var output = new Uint8Array(raw.length);
    for (var i = 0; i < raw.length; i++) {
      output[i] = raw.charCodeAt(i);
    }
    return output;
  }

  function setState(banner, state) {
    banner.setAttribute('data-state', state);
    var content = banner.querySelector('.push-content');
    var unsupported = $('push-unsupported');
    if (content) content.hidden = state === 'unsupported';
    if (unsupported) unsupported.hidden = state !== 'unsupported';

    var status = $('push-status');
    if (status) {
      status.textContent = state === 'enabled' ? 'Enabled' : (state === 'unsupported' ? 'Unavailable' : 'Disabled');
    }
  }

  function showSpinner(show) {
    var spinner = $('push-spinner');
    if (spinner) spinner.hidden = !show;
  }

  function setButtonsDisabled(disabled) {
    var enable = $('push-enable');
    var disable = $('push-disable');
    if (enable) enable.disabled = disabled;
    if (disable) disable.disabled = disabled;
  }

  function showReason(message) {
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

  async function postJSON(url, method, body) {
    var resp = await fetch(url, {
      method: method,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body || {})
    });
    if (!resp.ok) {
      var text = await resp.text();
      throw new Error(text || ('HTTP ' + resp.status));
    }
    return resp;
  }

  function toast(message, kind) {
    // Use the same toaster that the rest of the app uses. The toaster
    // listens for elements appended to #toaster; we add a small
    // dismissable element that mimics the Toast template.
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

  function escapeHtml(s) {
    return String(s).replace(/[&<>"']/g, function (c) {
      return ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c];
    });
  }

  async function init() {
    var banner = $('push-banner');
    if (!banner) return;

    var vapidKey = banner.getAttribute('data-vapid-key') || '';
    var support = detectSupport();
    if (!support.ok) {
      setState(banner, 'unsupported');
      showReason(support.reason);
      return;
    }

    if (!vapidKey) {
      setState(banner, 'unsupported');
      showReason('Server is missing its VAPID key.');
      return;
    }

    var enable = $('push-enable');
    var disable = $('push-disable');
    var recheck = $('push-recheck');

    if (recheck) {
      recheck.addEventListener('click', function () {
        var s = detectSupport();
        if (s.ok) {
          setState(banner, 'disabled');
        } else {
          showReason(s.reason);
        }
      });
    }

    // Reconcile with the browser. The server's first-paint guess may
    // be wrong (e.g. the user unsubscribed on another device), so
    // always re-check.
    try {
      var reg = await navigator.serviceWorker.ready;
      var existing = await reg.pushManager.getSubscription();
      if (existing) {
        setState(banner, 'enabled');
      } else {
        setState(banner, 'disabled');
      }
    } catch (e) {
      // Service worker may not be ready yet. Fall back to the
      // server's first-paint guess.
      console.warn('push: reconciliation failed', e);
    }

    if (enable) {
      enable.addEventListener('click', async function () {
        showSpinner(true);
        setButtonsDisabled(true);
        try {
          var permission = await Notification.requestPermission();
          if (permission !== 'granted') {
            toast({ title: 'Notifications', body: 'Permission not granted.' }, 'error');
            return;
          }
          var r = await navigator.serviceWorker.ready;
          var sub = await r.pushManager.subscribe({
            userVisibleOnly: true,
            applicationServerKey: urlB64ToUint8Array(vapidKey)
          });
          var json = sub.toJSON();
          await postJSON('/api/push/subscribe', 'POST', {
            endpoint: json.endpoint,
            p256dh: json.keys.p256dh,
            auth: json.keys.auth
          });
          setState(banner, 'enabled');
          toast({ title: 'Notifications enabled', body: "You'll receive admin messages." }, 'success');
        } catch (e) {
          console.error('push: subscribe failed', e);
          toast({ title: 'Notifications', body: (e && e.message) || 'Subscribe failed.' }, 'error');
        } finally {
          showSpinner(false);
          setButtonsDisabled(false);
        }
      });
    }

    if (disable) {
      disable.addEventListener('click', async function () {
        showSpinner(true);
        setButtonsDisabled(true);
        try {
          var r2 = await navigator.serviceWorker.ready;
          var sub2 = await r2.pushManager.getSubscription();
          if (sub2) {
            var endpoint = sub2.endpoint;
            await sub2.unsubscribe();
            await postJSON('/api/push/unsubscribe', 'DELETE', { endpoint: endpoint });
          }
          setState(banner, 'disabled');
          toast({ title: 'Notifications disabled' }, 'success');
        } catch (e) {
          console.error('push: unsubscribe failed', e);
          toast({ title: 'Notifications', body: (e && e.message) || 'Unsubscribe failed.' }, 'error');
        } finally {
          showSpinner(false);
          setButtonsDisabled(false);
        }
      });
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();

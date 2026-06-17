// Stren service worker.
//
// Responsibilities:
//   1. Activate immediately on install/activate so users opening the
//      PWA for the first time get push support without a reload.
//   2. Pass network requests through (no offline caching for now).
//   3. Receive push events from the push service and display a
//      notification. The payload is JSON of the form
//      {title, body, url?}.
//   4. On click, open the URL (when provided) or focus an existing
//      app window.
self.addEventListener('install', function (event) {
  self.skipWaiting();
});

self.addEventListener('activate', function (event) {
  event.waitUntil(self.clients.claim());
});

self.addEventListener('fetch', function (event) {
  // Pass through all requests to the network.
  // Offline support can be added here later if needed.
});

// push event — fired by the push service when the server delivers a
// message. The body is whatever the server sent; for this app it is
// always JSON {title, body, url?}.
self.addEventListener('push', function (event) {
  if (!event.data) {
    return;
  }

  var raw = event.data.text();
  var payload;
  try {
    payload = JSON.parse(raw);
  } catch (e) {
    // Fall back to a minimal text-only notification so the message
    // is still visible if the server sends something we don't
    // understand.
    payload = { title: 'Stren', body: raw };
  }

  var title = (payload && payload.title) || 'Stren';
  var options = {
    body: (payload && payload.body) || '',
    icon: '/icons/web-app-manifest-192x192.png',
    badge: '/icons/web-app-manifest-192x192.png',
    data: { url: (payload && payload.url) || '/' }
  };

  event.waitUntil(self.registration.showNotification(title, options));
});

// notificationclick — fired when the user taps the notification.
// Opens the URL in an existing tab if possible, otherwise in a new
// tab. Falls back to the dashboard if no URL is provided.
self.addEventListener('notificationclick', function (event) {
  event.notification.close();

  var targetUrl = (event.notification.data && event.notification.data.url) || '/';
  // Only follow same-origin URLs to avoid leaking the click via
  // window.open. Anything else is ignored and the notification is
  // just dismissed.
  if (typeof targetUrl !== 'string' || targetUrl[0] !== '/') {
    return;
  }

  event.waitUntil(
    self.clients
      .matchAll({ type: 'window', includeUncontrolled: true })
      .then(function (clientList) {
        for (var i = 0; i < clientList.length; i++) {
          var client = clientList[i];
          if (client.url && 'focus' in client) {
            client.focus();
            if ('navigate' in client) {
              return client.navigate(targetUrl);
            }
            return;
          }
        }
        if (self.clients.openWindow) {
          return self.clients.openWindow(targetUrl);
        }
      })
  );
});

// Minimal service worker for PWA installability.
// No caching strategy — offline support is intentionally not implemented.
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

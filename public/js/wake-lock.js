(function() {
	'use strict';

	var TAG = '[WakeLock]';
	var sentinel = null;
	var intended = false;
	var visibilityHandler = null;

	function log(message) {
		console.log(TAG, message);
	}

	function isSupported() {
		return typeof navigator !== 'undefined' && 'wakeLock' in navigator;
	}

	function isActive() {
		return sentinel !== null;
	}

	function clearVisibilityHandler() {
		if (visibilityHandler !== null) {
			document.removeEventListener('visibilitychange', visibilityHandler);
			visibilityHandler = null;
		}
	}

	function installVisibilityHandler() {
		if (visibilityHandler !== null) return;
		visibilityHandler = function() {
			if (document.visibilityState === 'visible' && intended && sentinel === null) {
				requestSentinel();
			}
		};
		document.addEventListener('visibilitychange', visibilityHandler);
	}

	function requestSentinel() {
		if (!isSupported() || sentinel !== null) return;
		navigator.wakeLock.request('screen').then(function(s) {
			sentinel = s;
			s.addEventListener('release', function() {
				if (sentinel === s) {
					sentinel = null;
				}
			});
			log('acquired');
		}).catch(function(err) {
			sentinel = null;
			log('request failed: ' + (err && err.name ? err.name : err));
		});
	}

	function acquire() {
		if (!isSupported()) {
			log('not supported in this browser');
			return;
		}
		if (intended) return;
		intended = true;
		installVisibilityHandler();
		if (document.visibilityState === 'visible') {
			requestSentinel();
		}
	}

	function release() {
		intended = false;
		clearVisibilityHandler();
		if (sentinel !== null) {
			try {
				sentinel.release().then(function() {
					log('released');
				}).catch(function() {
					log('release failed');
				});
			} catch (err) {
				log('release error: ' + err);
			}
			sentinel = null;
		}
	}

	window.WakeLock = {
		acquire: acquire,
		release: release,
		isActive: isActive
	};
})();

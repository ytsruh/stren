(function() {
  'use strict';

  var timer = null;
  var remainingSeconds = 0;

  function formatTime(seconds) {
    var mins = Math.floor(seconds / 60);
    var secs = seconds % 60;
    return String(mins).padStart(2, '0') + ':' + String(secs).padStart(2, '0');
  }

  function updateDisplay() {
    var display = document.getElementById('countdown-display');
    if (display) {
      display.textContent = formatTime(remainingSeconds);
    }
  }

  function showControls() {
    var startBtn = document.getElementById('start-btn');
    var stopBtn = document.getElementById('stop-btn');
    var resetBtn = document.getElementById('reset-btn');
    var display = document.getElementById('timer-display');
    var countdownDisplay = document.getElementById('countdown-display');
    var completeDiv = document.getElementById('timer-complete');
    if (startBtn) startBtn.classList.remove('hidden');
    if (stopBtn) stopBtn.classList.remove('hidden');
    if (resetBtn) resetBtn.classList.remove('hidden');
    if (display) {
      display.classList.remove('hidden');
      display.classList.add('flex');
    }
    if (countdownDisplay) countdownDisplay.classList.remove('hidden');
    if (completeDiv) {
      completeDiv.classList.add('hidden');
      completeDiv.classList.remove('flex');
    }
  }

  function hideControls() {
    var startBtn = document.getElementById('start-btn');
    var stopBtn = document.getElementById('stop-btn');
    var resetBtn = document.getElementById('reset-btn');
    var display = document.getElementById('timer-display');
    var countdownDisplay = document.getElementById('countdown-display');
    if (startBtn) startBtn.classList.add('hidden');
    if (stopBtn) stopBtn.classList.add('hidden');
    if (resetBtn) resetBtn.classList.add('hidden');
    if (display) {
      display.classList.add('hidden');
      display.classList.remove('flex');
    }
    if (countdownDisplay) countdownDisplay.classList.add('hidden');
  }

  function onTimerComplete() {
    hideControls();
    var display = document.getElementById('countdown-display');
    if (display) {
      display.textContent = 'Done!';
    }
    var completeDiv = document.getElementById('timer-complete');
    if (completeDiv) {
      completeDiv.classList.remove('hidden');
      completeDiv.classList.add('flex');
    }
    if ('vibrate' in navigator) {
      navigator.vibrate([200, 100, 200]);
    }
    if (window.WakeLock) {
      window.WakeLock.release();
    }
  }

  function tick() {
    remainingSeconds--;
    updateDisplay();
    if (remainingSeconds <= 0) {
      clearInterval(timer);
      timer = null;
      onTimerComplete();
    }
  }

  function startTimer() {
    if (timer) return;
    if (remainingSeconds <= 0) return;
    if (window.WakeLock) {
      window.WakeLock.acquire();
    }
    timer = setInterval(tick, 1000);
    showControls();
  }

  function stopTimer() {
    if (timer) {
      clearInterval(timer);
      timer = null;
    }
    if (window.WakeLock) {
      window.WakeLock.release();
    }
  }

  function resetTimer() {
    stopTimer();
    remainingSeconds = 0;
    updateDisplay();
    hideControls();
    var display = document.getElementById('countdown-display');
    if (display) {
      display.textContent = '--:--';
    }
    var completeDiv = document.getElementById('timer-complete');
    if (completeDiv) {
      completeDiv.classList.add('hidden');
      completeDiv.classList.remove('flex');
    }
    var presetBtns = document.querySelectorAll('#timer-presets button');
    presetBtns.forEach(function(btn) {
      btn.classList.remove('hidden');
    });
    document.getElementById('custom-duration').value = '';
    var customSection = document.getElementById('custom-duration-section');
    if (customSection) {
      customSection.classList.remove('hidden');
    }
  }

  window.startTimer = startTimer;
  window.stopTimer = stopTimer;
  window.resetTimer = resetTimer;

  window.startPresetTimer = function(seconds) {
    remainingSeconds = seconds;
    updateDisplay();
    showControls();
    startTimer();
    var presetBtns = document.querySelectorAll('#timer-presets button');
    presetBtns.forEach(function(btn) {
      btn.classList.add('hidden');
    });
    var customSection = document.getElementById('custom-duration-section');
    if (customSection) {
      customSection.classList.add('hidden');
    }
  };

  window.startCustomTimer = function() {
    var input = document.getElementById('custom-duration');
    var value = parseInt(input.value, 10);
    if (isNaN(value) || value < 1 || value > 300) {
      htmx.ajax('POST', '/timer/error', {target: '#toaster', swap: 'beforeend'});
      input.value = '';
      return;
    }
    remainingSeconds = value;
    updateDisplay();
    showControls();
    startTimer();
    var presetBtns = document.querySelectorAll('#timer-presets button');
    presetBtns.forEach(function(btn) {
      btn.classList.add('hidden');
    });
    var customSection = document.getElementById('custom-duration-section');
    if (customSection) {
      customSection.classList.add('hidden');
    }
  };
})();
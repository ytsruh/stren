(function() {
  'use strict';

  var timer = null;
  var totalRounds = 0;
  var currentRound = 1;
  var remainingSeconds = 60;

  function formatTime(seconds) {
    var mins = Math.floor(seconds / 60);
    var secs = seconds % 60;
    return String(mins).padStart(2, '0') + ':' + String(secs).padStart(2, '0');
  }

  function updateDisplay() {
    var roundDisplay = document.getElementById('round-display');
    var countdownDisplay = document.getElementById('countdown-display');
    if (roundDisplay) {
      roundDisplay.textContent = 'Round ' + currentRound + '/' + totalRounds;
    }
    if (countdownDisplay) {
      countdownDisplay.textContent = formatTime(remainingSeconds);
    }
  }

  function vibrate() {
    if ('vibrate' in navigator) {
      navigator.vibrate([200, 100, 200]);
    }
  }

  function showControls() {
    var startBtn = document.getElementById('start-btn');
    var stopBtn = document.getElementById('stop-btn');
    var resetBtn = document.getElementById('reset-btn');
    var display = document.getElementById('emom-display');
    var countdownDisplay = document.getElementById('countdown-display');
    var completeDiv = document.getElementById('emom-complete');
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
    var display = document.getElementById('emom-display');
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

  function onRoundComplete() {
    htmx.ajax('POST', '/timer/emom/round', {target: '#toaster', swap: 'beforeend', values: {round: currentRound}});
    vibrate();
    if (currentRound >= totalRounds) {
      onEMOMComplete();
      return;
    }
    currentRound++;
    remainingSeconds = 60;
    updateDisplay();
  }

  function onEMOMComplete() {
    clearInterval(timer);
    timer = null;
    hideControls();
    var roundDisplay = document.getElementById('round-display');
    if (roundDisplay) {
      roundDisplay.textContent = 'Complete!';
    }
    var countdownDisplay = document.getElementById('countdown-display');
    if (countdownDisplay) {
      countdownDisplay.textContent = 'Done!';
    }
    var completeDiv = document.getElementById('emom-complete');
    if (completeDiv) {
      completeDiv.classList.remove('hidden');
      completeDiv.classList.add('flex');
    }
    vibrate();
    if (window.WakeLock) {
      window.WakeLock.release();
    }
  }

  function tick() {
    remainingSeconds--;
    updateDisplay();
    if (remainingSeconds <= 0) {
      onRoundComplete();
    }
  }

  function startEMOM() {
    if (timer) return;
    if (totalRounds <= 0) return;
    if (window.WakeLock) {
      window.WakeLock.acquire();
    }
    timer = setInterval(tick, 1000);
    showControls();
  }

  function stopEMOM() {
    if (timer) {
      clearInterval(timer);
      timer = null;
    }
    if (window.WakeLock) {
      window.WakeLock.release();
    }
  }

  function resetEMOM() {
    stopEMOM();
    totalRounds = 0;
    currentRound = 1;
    remainingSeconds = 60;
    updateDisplay();
    hideControls();
    var roundDisplay = document.getElementById('round-display');
    if (roundDisplay) {
      roundDisplay.textContent = 'Round 1/--';
    }
    var countdownDisplay = document.getElementById('countdown-display');
    if (countdownDisplay) {
      countdownDisplay.textContent = '0:60';
    }
    var completeDiv = document.getElementById('emom-complete');
    if (completeDiv) {
      completeDiv.classList.add('hidden');
      completeDiv.classList.remove('flex');
    }
    var presetBtns = document.querySelectorAll('#emom-presets button');
    presetBtns.forEach(function(btn) {
      btn.classList.remove('hidden');
    });
    document.getElementById('custom-rounds').value = '';
    var customSection = document.getElementById('custom-rounds-section');
    if (customSection) {
      customSection.classList.remove('hidden');
    }
  }

  window.startEMOM = startEMOM;
  window.stopEMOM = stopEMOM;
  window.resetEMOM = resetEMOM;

  window.startPresetEMOM = function(rounds) {
    totalRounds = rounds;
    currentRound = 1;
    remainingSeconds = 60;
    updateDisplay();
    showControls();
    startEMOM();
    var presetBtns = document.querySelectorAll('#emom-presets button');
    presetBtns.forEach(function(btn) {
      btn.classList.add('hidden');
    });
    var customSection = document.getElementById('custom-rounds-section');
    if (customSection) {
      customSection.classList.add('hidden');
    }
  };

  window.startCustomEMOM = function() {
    var input = document.getElementById('custom-rounds');
    var value = parseInt(input.value, 10);
    if (isNaN(value) || value < 1 || value > 15) {
      htmx.ajax('POST', '/timer/emom/error', {target: '#toaster', swap: 'beforeend'});
      input.value = '';
      return;
    }
    totalRounds = value;
    currentRound = 1;
    remainingSeconds = 60;
    updateDisplay();
    showControls();
    startEMOM();
    var presetBtns = document.querySelectorAll('#emom-presets button');
    presetBtns.forEach(function(btn) {
      btn.classList.add('hidden');
    });
    var customSection = document.getElementById('custom-rounds-section');
    if (customSection) {
      customSection.classList.add('hidden');
    }
  };
})();
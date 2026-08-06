# Stren iOS Makefile
# Drives XcodeGen + xcodebuild. All real work happens in
# Xcode/CommandLineTools — this file is just a thin wrapper so
# the day-to-day commands are short and memorable.

SCHEME = Stren
# Use the generic simulator destination so xcodebuild doesn't
# try to resolve `OS:latest` to the latest installed runtime
# (which can be a beta iOS that doesn't have the iPhone model
# the user picked). The build is OS-agnostic; the `run` target
# picks a real simulator UUID via `xcrun simctl` after the
# build finishes.
DESTINATION ?= generic/platform=iOS Simulator
XCODEPROJ = Stren.xcodeproj
BUNDLE_ID = com.ytsruh.stren

# A regex that matches a simulator UUID. Used by `boot` and
# `run` to extract a usable device id from the (somewhat
# noisy) `xcrun simctl list devices` output.
SIM_UUID_RE = [A-F0-9]{8}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{12}

# Absolute path to the Simulator.app inside the active
# developer directory. Resolved once at parse time from
# `xcode-select -p`, so it works for Xcode.app, Xcode-beta.app,
# and any other variant without depending on Launch Services
# having registered the app under its short name. Used by
# `run` to bring the Simulator window to the front after
# `simctl boot` starts the device headless.
SIMULATOR_APP = $(shell xcode-select -p)/Applications/Simulator.app

.PHONY: help gen build boot run test clean

help:
	@echo "Stren iOS — available targets:"
	@echo ""
	@echo "  make gen    Regenerate $(XCODEPROJ) from project.yml"
	@echo "  make build  Compile for the booted simulator"
	@echo "  make boot   Boot the first available iPhone simulator (no-op if one is already running)"
	@echo "  make run    Build, install, and launch on the booted simulator (auto-boots one if needed)"
	@echo "  make test   Run the unit-test target on the simulator"
	@echo "  make clean  Wipe build artifacts"
	@echo ""
	@echo "Override the destination with DESTINATION=...  e.g."
	@echo "  make run DESTINATION='platform=iOS Simulator,name=iPhone 15 Pro'"

# XcodeGen — re-run after editing project.yml or moving/adding files.
# File globs in project.yml pick up new Swift files automatically;
# you only need this for structural changes (new target, plist keys,
# capabilities).
gen:
	@command -v xcodegen >/dev/null 2>&1 || { echo "xcodegen not installed. Run: brew install xcodegen"; exit 1; }
	xcodegen generate
	@echo "✓ $(XCODEPROJ) regenerated"

build:
	xcodebuild \
		-project $(XCODEPROJ) \
		-scheme $(SCHEME) \
		-configuration Debug \
		-destination '$(DESTINATION)' \
		build

# `boot` opens the first available iPhone simulator. It's a
# no-op if one is already booted so the user can run it
# freely. The simulator's UUID is written to stdout so any
# caller can chain off it.
#
# IMPORTANT: like `run`, this recipe body MUST be one
# continuous shell block. Every `#` comment without a
# trailing `\` would break the continuation chain and split
# the script across multiple subshells. All explanatory
# comments live above the recipe.
boot:
	@BOOTED=$$(xcrun simctl list devices booted 2>/dev/null | grep -E "iPhone|iPad" | head -1 | grep -oE "$(SIM_UUID_RE)"); \
	if [ -n "$$BOOTED" ]; then \
		echo "$$BOOTED"; \
		exit 0; \
	fi; \
	DEVICE=$$(xcrun simctl list devices available 2>/dev/null | grep -E "iPhone" | grep -v "unavailable" | head -1 | grep -oE "$(SIM_UUID_RE)"); \
	if [ -z "$$DEVICE" ]; then \
		echo "No iPhone simulator available. Open Xcode → Settings → Platforms to download one." >&2; \
		exit 1; \
	fi; \
	echo "Booting simulator $$DEVICE..." >&2; \
	xcrun simctl boot "$$DEVICE" >/dev/null 2>&1 || true; \
	for i in $$(seq 1 30); do \
		if xcrun simctl list devices 2>/dev/null | grep "$$DEVICE" | grep -q "Booted"; then \
			break; \
		fi; \
		sleep 1; \
	done; \
	echo "$$DEVICE"

# `run` is the one-keystroke dev loop. It builds, resolves
# (or boots) a simulator, then installs and launches. If a
# simulator is already booted, that device is used — boot a
# different one first to override.
#
# The boot logic is inlined here rather than delegated to
# the `boot` target via `$(MAKE) -s boot`, because capturing
# stdout from a sub-make through `$$()` is fragile (sub-make
# output buffering + `$$()` pipe handling + the `-s` flag
# can lose the captured value in ways that are hard to
# reproduce). Inlining keeps the entire boot → install →
# launch sequence in one shell, one variable scope, and one
# place to add error reporting. The `boot` target remains
# as a standalone "boot a simulator and print its UUID"
# command for scripts that want to chain off it.
#
# IMPORTANT: the recipe body MUST be one continuous shell
# block. Every line of the recipe is a single `sh -c`
# invocation, so a `#` comment that doesn't end with `\` will
# break the continuation chain and split the script across
# multiple shells (causing "unexpected end of file" errors
# because `if/fi`, `for/done`, etc. get truncated). All
# explanatory comments live above the recipe, not inside it.
run: build
	@BOOTED=$$(xcrun simctl list devices booted 2>/dev/null | grep -E "iPhone" | head -1 | grep -oE "$(SIM_UUID_RE)"); \
	if [ -n "$$BOOTED" ]; then \
		DEVICE="$$BOOTED"; \
		echo "Using already-booted simulator $$DEVICE" >&2; \
	else \
		DEVICE=$$(xcrun simctl list devices available 2>/dev/null | grep -E "iPhone" | grep -v "unavailable" | head -1 | grep -oE "$(SIM_UUID_RE)"); \
		if [ -z "$$DEVICE" ]; then \
			echo "No iPhone simulator available. Open Xcode → Settings → Platforms to download one." >&2; \
			exit 1; \
		fi; \
		echo "Booting simulator $$DEVICE..." >&2; \
		xcrun simctl boot "$$DEVICE" >/dev/null 2>&1 || true; \
		open "$(SIMULATOR_APP)" 2>/dev/null || true; \
		for i in $$(seq 1 30); do \
			if xcrun simctl list devices 2>/dev/null | grep "$$DEVICE" | grep -q "Booted"; then \
				break; \
			fi; \
			sleep 1; \
		done; \
	fi; \
	APP_PATH=$$(xcodebuild -project $(XCODEPROJ) -scheme $(SCHEME) -configuration Debug -destination '$(DESTINATION)' -showBuildSettings 2>/dev/null | awk -F' = ' '/^[[:space:]]+TARGET_BUILD_DIR/ {d=$$2} /^[[:space:]]+FULL_PRODUCT_NAME/ {n=$$2} END {print d"/"n}'); \
	if [ -z "$$DEVICE" ]; then \
		echo "Internal error: device UUID is empty after boot. Please report this." >&2; \
		exit 1; \
	fi; \
	if [ -z "$$APP_PATH" ] || [ ! -d "$$APP_PATH" ]; then \
		echo "Could not locate built .app. Try: make build" >&2; \
		exit 1; \
	fi; \
	echo "Installing $$APP_PATH on $$DEVICE"; \
	xcrun simctl install "$$DEVICE" "$$APP_PATH" && \
	PID=$$(xcrun simctl launch "$$DEVICE" $(BUNDLE_ID) | awk '{print $$NF}') && \
	echo "" && \
	echo "✓ Stren is running on $$DEVICE (pid $$PID)" && \
	echo "  The Simulator window should be on screen. If not, run: open \"$(SIMULATOR_APP)\""

test:
	xcodebuild \
		-project $(XCODEPROJ) \
		-scheme $(SCHEME) \
		-configuration Debug \
		-destination '$(DESTINATION)' \
		test

clean:
	rm -rf build
	xcodebuild \
		-project $(XCODEPROJ) \
		-scheme $(SCHEME) \
		clean 2>/dev/null || true

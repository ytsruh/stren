import SwiftUI

/// The "Today" tab. The layout mirrors the web dashboard:
/// action cards at the top, a "Last 7 Days" stats row, the
/// "Most Popular Exercises (7d)" donut, and the date-grouped
/// list of sets. Tapping a set row pushes the exercise's
/// history view, matching the web's eye-icon link.
struct DashboardView: View {
    @EnvironmentObject private var env: AppEnvironment
    @EnvironmentObject private var authStore: AuthStore

    @State private var entries: [ExerciseEntryDTO] = []
    @State private var exerciseLookup: [String: ExerciseDTO] = [:]
    @State private var isLoading: Bool = false
    @State private var errorMessage: String?
    @State private var showingNewSet: Bool = false
    @State private var showingTimer: Bool = false

    var body: some View {
        NavigationStack {
            content
                .navigationTitle("Dashboard")
                .navigationDestination(for: ExerciseDTO.self) { exercise in
                    ExerciseHistoryView(exercise: exercise)
                }
                .sheet(isPresented: $showingNewSet, onDismiss: {
                    Task { await load() }
                }) {
                    NewSetView()
                        .environmentObject(env)
                }
                .sheet(isPresented: $showingTimer) {
                    TimerView()
                }
        }
        .task { await load() }
        .refreshable { await load() }
    }

    // MARK: - Content states

    @ViewBuilder
    private var content: some View {
        if isLoading && entries.isEmpty {
            ProgressView()
                .frame(maxWidth: .infinity, maxHeight: .infinity)
        } else if let errorMessage, entries.isEmpty {
            errorState(errorMessage)
        } else if entries.isEmpty {
            loadedScrollView(empty: true)
        } else {
            loadedScrollView(empty: false)
        }
    }

    /// Renders the dashboard's normal layout (action cards,
    /// donut, list). When `empty` is true the list is
    /// replaced with the empty-state view so the layout still
    /// shows the action cards (Add Set is the empty-state's
    /// primary call to action anyway).
    @ViewBuilder
    private func loadedScrollView(empty: Bool) -> some View {
        ScrollView {
            VStack(alignment: .leading, spacing: DSSpacing.md) {
                actionCards
                donutSection
                if empty {
                    emptyState
                } else {
                    sectionHeader("Recent Sets")
                    DashboardSetList(
                        groups: groups,
                        weightUnit: weightUnit,
                        exerciseLookup: exerciseLookup,
                        onDelete: { entry in
                            Task { await deleteEntry(entry) }
                        }
                    )
                }
            }
            .padding(DSSpacing.md)
        }
        .background(DSColors.background.ignoresSafeArea())
    }

    private var actionCards: some View {
        HStack(spacing: DSSpacing.xs) {
            ActionCard(label: "Add Set", systemImage: Icons.addSet) {
                showingNewSet = true
            }
            ActionCard(label: "Timer", systemImage: Icons.timer) {
                showingTimer = true
            }
        }
    }

    @ViewBuilder
    private var donutSection: some View {
        let buckets = popularExerciseBuckets(entries)
        if !buckets.isEmpty {
            VStack(alignment: .leading, spacing: DSSpacing.xs) {
                sectionHeader("7 Day History")
                DashboardDonutChart(buckets: buckets)
                    .padding(DSSpacing.md)
                    .frame(maxWidth: .infinity)
                    .background(
                        RoundedRectangle(cornerRadius: DSSpacing.cornerRadius, style: .continuous)
                            .fill(DSColors.surface)
                    )
                    .overlay(
                        RoundedRectangle(cornerRadius: DSSpacing.cornerRadius, style: .continuous)
                            .stroke(DSColors.separator, lineWidth: 0.5)
                    )
            }
        }
    }

    private func sectionHeader(_ text: String) -> some View {
        Text(text)
            .font(.headline)
            .foregroundStyle(DSColors.text)
            .padding(.horizontal, DSSpacing.xs)
    }

    private var emptyState: some View {
        VStack(spacing: DSSpacing.md) {
            Image(systemName: Icons.dumbbellLarge)
                .font(.system(size: 48))
                .foregroundStyle(DSColors.textSecondary)
            Text("No sets yet")
                .font(.title3.weight(.semibold))
            Text("Tap Add Set to log your first workout.")
                .font(.body)
                .foregroundStyle(DSColors.textSecondary)
                .multilineTextAlignment(.center)
        }
        .padding(DSSpacing.lg)
        .frame(maxWidth: .infinity)
    }

    private func errorState(_ message: String) -> some View {
        VStack(spacing: DSSpacing.md) {
            Image(systemName: Icons.warning)
                .font(.largeTitle)
                .foregroundStyle(DSColors.destructive)
            Text(message)
                .multilineTextAlignment(.center)
                .foregroundStyle(DSColors.textSecondary)
            Button("Try again") { Task { await load() } }
                .buttonStyle(.dsSecondary)
        }
        .padding(DSSpacing.lg)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    // MARK: - Derived data

    private var weightUnit: String {
        authStore.currentUser?.weightUnit ?? "kg"
    }

    private var groups: [DashboardDateGroup] {
        Self.group(entries: entries)
    }

    // MARK: - Networking

    /// Fetches the 7-day entries and the full exercise
    /// catalogue in parallel. The catalogue is needed so
    /// every list row can navigate to its exercise's
    /// history view (the row's `exerciseID` alone is not
    /// enough to push — we need the full `ExerciseDTO`).
    /// Both fetches share the same 401 handling, so an
    /// expired session still signs the user out exactly
    /// once.
    private func load() async {
        errorMessage = nil
        isLoading = true
        defer { isLoading = false }
        do {
            async let entriesTask = env.api.listExerciseEntries(days: 7)
            async let exercisesTask = env.api.listExercises()
            let (fetchedEntries, fetchedExercises) = try await (entriesTask, exercisesTask)
            entries = fetchedEntries
            exerciseLookup = Dictionary(
                uniqueKeysWithValues: fetchedExercises.map { ($0.id, $0) }
            )
        } catch let error as APIError {
            if case .unauthorized = error { return }
            errorMessage = error.errorDescription
        } catch {
            errorMessage = "Could not load your sets."
        }
    }

    private func deleteEntry(_ entry: ExerciseEntryDTO) async {
        // Optimistic remove; rollback on error.
        entries.removeAll { $0.id == entry.id }
        do {
            try await env.api.deleteExerciseEntry(id: entry.id)
        } catch {
            await load()
        }
    }

    /// Groups an unsorted set of entries into day buckets in
    /// the local timezone, ordered newest first.
    private static func group(entries: [ExerciseEntryDTO]) -> [DashboardDateGroup] {
        let calendar = Calendar.current
        let grouped = Dictionary(grouping: entries) { entry in
            calendar.startOfDay(for: entry.createdAt)
        }
        return grouped
            .map {
                DashboardDateGroup(
                    date: $0.key,
                    entries: $0.value.sorted { $0.createdAt > $1.createdAt }
                )
            }
            .sorted { $0.date > $1.date }
    }
}

#Preview {
    DashboardView()
        .environmentObject(AppEnvironment.live(baseURL: URL(string: "http://localhost:8080/api/v1")!))
        .environmentObject(AuthStore(api: APIClient(
            baseURL: URL(string: "http://localhost:8080/api/v1")!,
            tokenProvider: { nil }
        )))
}

import SwiftUI

/// The "Today" tab. Shows the user's sets from the last 7
/// days, grouped by date, newest first. The toolbar "+"
/// button presents `NewSetView` as a sheet; after a
/// successful save the list reloads so the new set is
/// visible without a manual pull-to-refresh.
struct DashboardView: View {
    @EnvironmentObject private var env: AppEnvironment
    @EnvironmentObject private var authStore: AuthStore

    @State private var groups: [DateGroup] = []
    @State private var isLoading: Bool = false
    @State private var errorMessage: String?
    @State private var showingNewSet: Bool = false

    var body: some View {
        NavigationStack {
            content
                .navigationTitle("Today")
                .toolbar {
                    ToolbarItem(placement: .topBarTrailing) {
                        Button {
                            showingNewSet = true
                        } label: {
                            Image(systemName: "plus")
                                .accessibilityLabel("Add set")
                        }
                    }
                }
                .sheet(isPresented: $showingNewSet, onDismiss: {
                    Task { await load() }
                }) {
                    NewSetView()
                        .environmentObject(env)
                }
        }
        .task { await load() }
        .refreshable { await load() }
    }

    @ViewBuilder
    private var content: some View {
        if isLoading && groups.isEmpty {
            ProgressView()
                .frame(maxWidth: .infinity, maxHeight: .infinity)
        } else if let errorMessage, groups.isEmpty {
            VStack(spacing: DSSpacing.md) {
                Image(systemName: "exclamationmark.triangle")
                    .font(.largeTitle)
                    .foregroundStyle(DSColors.destructive)
                Text(errorMessage)
                    .multilineTextAlignment(.center)
                    .foregroundStyle(DSColors.textSecondary)
                Button("Try again") { Task { await load() } }
                    .buttonStyle(.dsSecondary)
            }
            .padding(DSSpacing.lg)
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        } else if groups.isEmpty {
            emptyState
        } else {
            List {
                ForEach(groups) { group in
                    Section(header: header(for: group.date)) {
                        ForEach(group.entries) { entry in
                            SetRow(entry: entry, weightUnit: authStore.currentUser?.weightUnit ?? "kg")
                        }
                        .onDelete { indexSet in
                            Task { await deleteEntries(at: indexSet, in: group) }
                        }
                    }
                }
            }
            .listStyle(.insetGrouped)
        }
    }

    private func header(for date: Date) -> some View {
        Text(date, format: .dateTime.weekday(.wide).day().month())
            .font(.subheadline.weight(.semibold))
            .foregroundStyle(DSColors.textSecondary)
    }

    private var emptyState: some View {
        VStack(spacing: DSSpacing.md) {
            Image(systemName: "dumbbell")
                .font(.system(size: 48))
                .foregroundStyle(DSColors.textSecondary)
            Text("No sets yet")
                .font(.title3.weight(.semibold))
            Text("Tap the + button to log your first set.")
                .font(.body)
                .foregroundStyle(DSColors.textSecondary)
                .multilineTextAlignment(.center)
        }
        .padding(DSSpacing.lg)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    // MARK: - Data

    private func load() async {
        errorMessage = nil
        isLoading = true
        defer { isLoading = false }
        do {
            let entries = try await env.api.listExerciseEntries(days: 7)
            groups = Self.group(entries: entries)
        } catch let error as APIError {
            if case .unauthorized = error { return }
            errorMessage = error.errorDescription
        } catch {
            errorMessage = "Could not load your sets."
        }
    }

    private func deleteEntries(at indexSet: IndexSet, in group: DateGroup) async {
        let toDelete = indexSet.map { group.entries[$0] }
        // Optimistic remove; rollback on error.
        for entry in toDelete {
            groups = groups.compactMap { g in
                let filtered = g.entries.filter { $0.id != entry.id }
                return filtered.isEmpty ? nil : DateGroup(date: g.date, entries: filtered)
            }
        }
        for entry in toDelete {
            do {
                try await env.api.deleteExerciseEntry(id: entry.id)
            } catch {
                // Roll back by reloading.
                await load()
                return
            }
        }
    }

    /// Groups an unsorted set of entries into day buckets
    /// in the local timezone, ordered newest first.
    private static func group(entries: [ExerciseEntryDTO]) -> [DateGroup] {
        let calendar = Calendar.current
        let grouped = Dictionary(grouping: entries) { entry in
            calendar.startOfDay(for: entry.createdAt)
        }
        return grouped
            .map { DateGroup(date: $0.key, entries: $0.value.sorted { $0.createdAt > $1.createdAt }) }
            .sorted { $0.date > $1.date }
    }
}

struct DateGroup: Identifiable, Equatable {
    var id: TimeInterval { date.timeIntervalSince1970 }
    let date: Date
    let entries: [ExerciseEntryDTO]
}

/// A single row in the dashboard list. Shows the exercise
/// name, the set's reps x weight, and the time of day. The
/// weight unit is taken from the user's stored preference.
struct SetRow: View {
    let entry: ExerciseEntryDTO
    let weightUnit: String

    var body: some View {
        HStack(alignment: .firstTextBaseline) {
            VStack(alignment: .leading, spacing: DSSpacing.xxs) {
                Text(entry.exerciseName)
                    .font(.body.weight(.semibold))
                    .foregroundStyle(DSColors.text)
                if !entry.notes.isEmpty {
                    Text(entry.notes)
                        .font(.footnote)
                        .foregroundStyle(DSColors.textSecondary)
                        .lineLimit(2)
                }
            }
            Spacer()
            VStack(alignment: .trailing, spacing: DSSpacing.xxs) {
                Text("\(entry.reps) × \(formattedWeight)")
                    .font(.body.weight(.semibold).monospacedDigit())
                    .foregroundStyle(DSColors.text)
                Text(entry.createdAt, style: .time)
                    .font(.caption)
                    .foregroundStyle(DSColors.textSecondary)
            }
        }
        .padding(.vertical, DSSpacing.xxs)
    }

    private var formattedWeight: String {
        String(format: "%.1f %@", entry.weight, weightUnit)
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

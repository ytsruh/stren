import Charts
import SwiftUI

/// The per-exercise history screen. Pushed onto the
/// `ExerciseListView` navigation stack when the user taps
/// an exercise. Shows a Swift Charts line chart of the
/// user's progress (heaviest set per day) at the top, then
/// a paginated table of every set in reverse chronological
/// order, and lifetime stats (max weight, last set) above
/// the table.
struct ExerciseHistoryView: View {
    @EnvironmentObject private var env: AppEnvironment
    @EnvironmentObject private var authStore: AuthStore

    let exercise: ExerciseDTO

    @State private var page: HistoryPageDTO?
    @State private var chartPoints: [ChartPoint] = []
    @State private var isLoading: Bool = true
    @State private var errorMessage: String?
    @State private var currentPage: Int = 1

    private let pageSize: Int = 25

    var body: some View {
        content
            .navigationTitle(exercise.name)
            .navigationBarTitleDisplayMode(.inline)
            .task(id: currentPage) { await load() }
    }

    @ViewBuilder
    private var content: some View {
        if isLoading && page == nil {
            ProgressView()
                .frame(maxWidth: .infinity, maxHeight: .infinity)
        } else if let errorMessage, page == nil {
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
            .padding()
        } else if let page, page.entries.isEmpty {
            emptyState
        } else {
            List {
                chartSection
                statsSection
                Section("Sets") {
                    ForEach(page?.entries ?? []) { entry in
                        SetRow(entry: entry, weightUnit: weightUnit)
                    }
                }
                paginationSection
            }
            .listStyle(.insetGrouped)
        }
    }

    @ViewBuilder
    private var chartSection: some View {
        if !chartPoints.isEmpty {
            Section("Progress") {
                Chart(chartPoints) { point in
                    LineMark(
                        x: .value("Date", point.date),
                        y: .value("Max weight", point.weight)
                    )
                    .interpolationMethod(.monotone)
                    PointMark(
                        x: .value("Date", point.date),
                        y: .value("Max weight", point.weight)
                    )
                }
                .frame(height: 200)
                .chartYAxis {
                    AxisMarks(position: .leading)
                }
            }
        }
    }

    @ViewBuilder
    private var statsSection: some View {
        if let stats = page?.stats {
            Section("Stats") {
                HStack {
                    Text("Max weight")
                    Spacer()
                    Text(String(format: "%.1f %@", stats.maxWeight, weightUnit))
                        .font(.body.weight(.semibold).monospacedDigit())
                        .foregroundStyle(DSColors.text)
                }
                if let lastSet = stats.lastSet {
                    HStack {
                        Text("Last set")
                        Spacer()
                        Text("\(lastSet.reps) × \(String(format: "%.1f", lastSet.weight)) \(weightUnit)")
                            .font(.body.weight(.semibold).monospacedDigit())
                            .foregroundStyle(DSColors.text)
                    }
                    HStack {
                        Text("Last on")
                        Spacer()
                        Text(lastSet.createdAt, format: .dateTime.day().month().year())
                            .foregroundStyle(DSColors.textSecondary)
                    }
                }
            }
        }
    }

    @ViewBuilder
    private var paginationSection: some View {
        HStack {
            Button("Previous") { currentPage = max(1, currentPage - 1) }
                .disabled(!(page?.hasPrev ?? false))
            Spacer()
            Text("Page \(currentPage)")
                .font(.footnote)
                .foregroundStyle(DSColors.textSecondary)
            Spacer()
            Button("Next") { currentPage += 1 }
                .disabled(!(page?.hasNext ?? false))
        }
    }

    private var emptyState: some View {
        VStack(spacing: DSSpacing.md) {
            Image(systemName: "chart.line.uptrend.xyaxis")
                .font(.system(size: 48))
                .foregroundStyle(DSColors.textSecondary)
            Text("No sets yet")
                .font(.title3.weight(.semibold))
            Text("Log a set from the Today tab to start tracking \(exercise.name).")
                .font(.body)
                .foregroundStyle(DSColors.textSecondary)
                .multilineTextAlignment(.center)
        }
        .padding()
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private var weightUnit: String {
        authStore.currentUser?.weightUnit ?? "kg"
    }

    // MARK: - Data

    private func load() async {
        if page == nil { isLoading = true }
        defer { isLoading = false }
        do {
            let result = try await env.api.getExerciseHistory(id: exercise.id, page: currentPage)
            page = result
            // Refresh the chart in the background; pagination
            // should feel snappy.
            Task { await loadChart() }
        } catch let error as APIError {
            if case .unauthorized = error { return }
            errorMessage = error.errorDescription
        } catch {
            errorMessage = "Could not load history."
        }
    }

    private func loadChart() async {
        do {
            let entries = try await env.api.getExerciseChartData(id: exercise.id)
            chartPoints = Self.chartPoints(from: entries)
        } catch {
            // Non-fatal: history is still useful without the chart.
            chartPoints = []
        }
    }

    /// Aggregates the full entry list into one point per
    /// calendar day (the heaviest set of the day). Matches
    /// the dashboard's "heaviest per day" series so the two
    /// views agree.
    private static func chartPoints(from entries: [ExerciseEntryDTO]) -> [ChartPoint] {
        let calendar = Calendar.current
        let grouped = Dictionary(grouping: entries) { entry in
            calendar.startOfDay(for: entry.createdAt)
        }
        return grouped
            .map { day, sets in
                ChartPoint(
                    date: day,
                    weight: sets.map(\.weight).max() ?? 0
                )
            }
            .sorted { $0.date < $1.date }
    }
}

private struct ChartPoint: Identifiable {
    let date: Date
    let weight: Double
    var id: TimeInterval { date.timeIntervalSince1970 }
}

#Preview {
    NavigationStack {
        ExerciseHistoryView(exercise: ExerciseDTO(
            id: "ex-1", name: "Squat", description: "", videoURL: "", imgURL: "", type: "strength"
        ))
        .environmentObject(AppEnvironment.live(baseURL: URL(string: "http://localhost:8080/api/v1")!))
        .environmentObject(AuthStore(api: APIClient(
            baseURL: URL(string: "http://localhost:8080/api/v1")!,
            tokenProvider: { nil }
        )))
    }
}

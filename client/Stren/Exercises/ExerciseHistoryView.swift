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
    @State private var selectedDate: Date?

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
                        HistorySetRow(entry: entry, weightUnit: weightUnit)
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
                    // Highlight marks drawn in chart
                    // coordinate space (RuleMark / PointMark
                    // must live inside `Chart { ... }` — the
                    // .chartOverlay content is a plain View and
                    // can only host the floating card).
                    if let selectedDate,
                       let nearest = nearestPoint(to: selectedDate),
                       point.id == nearest.id {
                        RuleMark(x: .value("Selected", nearest.date))
                            .foregroundStyle(DSColors.textSecondary.opacity(0.35))
                            .lineStyle(StrokeStyle(lineWidth: 1, dash: [3, 3]))
                            .zIndex(-1)
                        PointMark(
                            x: .value("Date", nearest.date),
                            y: .value("Max weight", nearest.weight)
                        )
                        .foregroundStyle(DSColors.accent)
                        .symbolSize(120)
                    }
                }
                .frame(height: 200)
                .chartYScale(domain: chartYDomain)
                .chartYAxis {
                    AxisMarks(position: .leading, values: .automatic(desiredCount: 4)) { value in
                        AxisGridLine()
                        AxisTick()
                        AxisValueLabel {
                            if let weight = value.as(Double.self) {
                                Text("\(Int(weight.rounded())) \(weightUnit)")
                                    .font(.caption2)
                            }
                        }
                    }
                }
                .chartXSelection(value: $selectedDate)
                .chartOverlay { proxy in
                    if let selectedDate,
                       let nearest = nearestPoint(to: selectedDate),
                       let xPos = proxy.position(forX: nearest.date),
                       let yPos = proxy.position(forY: nearest.weight) {
                        // Floating card overlay. The proxy
                        // returns positions in the chart's
                        // content area; we clamp the x so the
                        // card never runs off the chart edges.
                        GeometryReader { geometry in
                            let cardHalfWidth: CGFloat = 50
                            let minX = cardHalfWidth + 4
                            let maxX = geometry.size.width - cardHalfWidth - 4
                            let clampedX = min(max(xPos, minX), maxX)
                            let clampedY = max(yPos - 28, 20)

                            chartTooltip(for: nearest)
                                .position(x: clampedX, y: clampedY)
                        }
                    }
                }
            }
        }
    }

    /// Y-axis domain for the progress chart. Rounds the data's
    /// min/max out to a "nice" step and then pads by 10% on
    /// each side so the line doesn't kiss the top and bottom
    /// edges. The nice-step rounding keeps the axis from
    /// snapping to the next round number below the data (e.g.
    /// anchoring to 10 when the data actually starts at 11).
    private var chartYDomain: ClosedRange<Double> {
        let weights = chartPoints.map(\.weight)
        let rawLo = weights.min() ?? 0
        let rawHi = weights.max() ?? 0
        let step = Self.niceStep(for: rawHi - rawLo)
        let lo = (rawLo / step).rounded(.down) * step
        let hi = (rawHi / step).rounded(.up) * step
        let pad = max((hi - lo) * 0.1, step)
        return (lo - pad)...(hi + pad)
    }

    /// Picks a "nice" tick step (1, 2.5, 5, 10, …) sized to
    /// the data's range so axis labels read as clean numbers.
    private static func niceStep(for range: Double) -> Double {
        switch range {
        case ..<5:    return 1
        case ..<20:   return 2.5
        case ..<50:   return 5
        case ..<200:  return 10
        case ..<500:  return 25
        default:      return 50
        }
    }

    /// Returns the chart point whose date is closest to the
    /// given target, or nil if the chart has no data. Used by
    /// the tooltip to snap the user's touch to a real
    /// observation.
    private func nearestPoint(to date: Date) -> ChartPoint? {
        chartPoints.min { lhs, rhs in
            abs(lhs.date.timeIntervalSince(date)) < abs(rhs.date.timeIntervalSince(date))
        }
    }

    /// Small floating card showing the date and weight for the
    /// highlighted chart point. Positioned by the caller via
    /// `.position(x:y:)` so the card sits above the dot.
    @ViewBuilder
    private func chartTooltip(for point: ChartPoint) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(point.date.formatted(.dateTime.day().month(.abbreviated).year()))
                .font(.caption2)
                .foregroundStyle(DSColors.textSecondary)
            Text("\(Int(point.weight.rounded())) \(weightUnit)")
                .font(.caption.weight(.semibold).monospacedDigit())
                .foregroundStyle(DSColors.text)
        }
        .padding(.horizontal, 8)
        .padding(.vertical, 6)
        .background(
            RoundedRectangle(cornerRadius: 8)
                .fill(DSColors.surface)
                .shadow(color: .black.opacity(0.15), radius: 4, y: 1)
        )
        .overlay(
            RoundedRectangle(cornerRadius: 8)
                .stroke(DSColors.separator, lineWidth: 0.5)
        )
        .fixedSize()
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
                        Text("Last")
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
        HStack(spacing: DSSpacing.sm) {
            Button { currentPage = max(1, currentPage - 1) } label: {
                Image(systemName: "chevron.left")
                    .font(.footnote.weight(.semibold))
            }
            .buttonStyle(.plain)
            .foregroundStyle(DSColors.textSecondary)
            .disabled(!(page?.hasPrev ?? false))

            Text("\(currentPage)")
                .font(.footnote.weight(.semibold).monospacedDigit())
                .foregroundStyle(DSColors.textSecondary)
                .frame(minWidth: 20)

            Button { currentPage += 1 } label: {
                Image(systemName: "chevron.right")
                    .font(.footnote.weight(.semibold))
            }
            .buttonStyle(.plain)
            .foregroundStyle(DSColors.textSecondary)
            .disabled(!(page?.hasNext ?? false))
        }
        .frame(maxWidth: .infinity)
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
        let isFirstLoad = page == nil
        if isFirstLoad { isLoading = true }
        defer { isLoading = false }
        do {
            let result = try await env.api.getExerciseHistory(id: exercise.id, page: currentPage)
            page = result
            // The chart is a lifetime view of all sets, so it
            // only needs to be fetched once — not on every page
            // change. Caching the points makes pagination feel
            // instant.
            if isFirstLoad {
                Task { await loadChart() }
            }
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

/// A single row in the exercise-history list. Intentionally
/// narrower than the dashboard's `SetRow`: the exercise name
/// is implicit (we're already inside that exercise's screen),
/// notes are dropped, and the time of day is hidden in favour
/// of the date so rows scan cleanly when paging through
/// history.
private struct HistorySetRow: View {
    let entry: ExerciseEntryDTO
    let weightUnit: String

    var body: some View {
        HStack(alignment: .firstTextBaseline) {
            Text(entry.createdAt, format: .dateTime.day().month(.abbreviated).year())
                .font(.subheadline)
                .foregroundStyle(DSColors.textSecondary)
            Spacer()
            Text("\(entry.reps) × \(String(format: "%.1f", entry.weight)) \(weightUnit)")
                .font(.body.weight(.semibold).monospacedDigit())
                .foregroundStyle(DSColors.text)
        }
        .padding(.vertical, DSSpacing.xxs)
    }
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

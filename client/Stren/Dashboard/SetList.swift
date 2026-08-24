import SwiftUI

/// The date-grouped list of sets below the donut. Each
/// row is a `NavigationLink` to the exercise's history
/// view, matching the web's eye-icon link from the
/// dashboard table. Tapping the row (not the swipe action)
/// is what navigates; swipe-to-delete lives on the row
/// itself.
///
/// The `exerciseLookup` is a map from `exerciseID` to the
/// full `ExerciseDTO` the row needs to push. The dashboard
/// fetches the exercise list once at load time so every
/// row in the list can resolve its destination without an
/// extra fetch per row.
struct DashboardSetList: View {
    let groups: [DashboardDateGroup]
    let weightUnit: String
    let distanceUnit: String
    let exerciseLookup: [String: ExerciseDTO]
    let onDelete: (ExerciseEntryDTO) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: DSSpacing.md) {
            ForEach(groups) { group in
                VStack(spacing: 0) {
                    ForEach(Array(group.entries.enumerated()), id: \.element.id) { index, entry in
                        rowLink(for: entry)
                        if index < group.entries.count - 1 {
                            Divider()
                                .background(DSColors.separator)
                                .padding(.leading, DSSpacing.md)
                        }
                    }
                }
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

    @ViewBuilder
    private func rowLink(for entry: ExerciseEntryDTO) -> some View {
        if let exercise = exerciseLookup[entry.exerciseID] {
            NavigationLink(value: exercise) {
                SetRow(entry: entry, weightUnit: weightUnit, distanceUnit: distanceUnit)
            }
            .buttonStyle(.plain)
            .swipeActions(edge: .trailing, allowsFullSwipe: true) {
                Button(role: .destructive) {
                    onDelete(entry)
                } label: {
                    Label("Delete", systemImage: "trash")
                }
            }
        } else {
            // The exercise was deleted server-side after the
            // set was logged. Show the row read-only so the
            // user can still see the set without crashing on
            // a missing destination.
            SetRow(entry: entry, weightUnit: weightUnit, distanceUnit: distanceUnit)
                .opacity(0.6)
        }
    }
}

/// One row in the date-grouped list. Mirrors the existing
/// dashboard `SetRow` and is kept here so the list is a
/// self-contained component. Only shows the exercise name,
/// notes, and a one-line metric summary — reps × weight for
/// strength entries, duration · distance for cardio — with
/// time of day and date intentionally omitted to keep the
/// list scannable.
struct SetRow: View {
    let entry: ExerciseEntryDTO
    let weightUnit: String
    let distanceUnit: String

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
            Text(summaryText)
                .font(.body.weight(.semibold).monospacedDigit())
                .foregroundStyle(DSColors.text)
        }
        .padding(.vertical, DSSpacing.sm)
        .padding(.horizontal, DSSpacing.md)
        .contentShape(Rectangle())
    }

    /// Type-aware one-line summary: "5 × 100.0 kg" for strength,
    /// "25:00 · 5.20 km" for cardio (mirrors the web's Details column).
    private var summaryText: String {
        if entry.isCardio {
            return "\(Self.formattedDuration(entry.durationSeconds)) · \(formattedDistance)"
        }
        return "\(entry.reps) × \(formattedWeight)"
    }

    private var formattedWeight: String {
        String(format: "%.1f %@", entry.weight, weightUnit)
    }

    /// Distance converted from stored metres to the user's unit.
    private var formattedDistance: String {
        let km = entry.distanceMeters / 1000.0
        if distanceUnit == "mi" {
            return String(format: "%.2f mi", km / 1.609344)
        }
        return String(format: "%.2f km", km)
    }

    /// M:SS, switching to H:MM:SS past an hour ("25:00", "1:05:30").
    static func formattedDuration(_ seconds: Int) -> String {
        let s = max(0, seconds)
        let h = s / 3600, m = (s % 3600) / 60, sec = s % 60
        if h > 0 {
            return String(format: "%d:%02d:%02d", h, m, sec)
        }
        return String(format: "%02d:%02d", m, sec)
    }
}

/// A single day bucket in the dashboard list. Renamed from
/// the previous `DateGroup` so the dashboard file is the
/// only place that needs to know about the grouping logic.
struct DashboardDateGroup: Identifiable, Equatable {
    let date: Date
    let entries: [ExerciseEntryDTO]
    var id: TimeInterval { date.timeIntervalSince1970 }
}

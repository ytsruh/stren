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
                SetRow(entry: entry, weightUnit: weightUnit)
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
            SetRow(entry: entry, weightUnit: weightUnit)
                .opacity(0.6)
        }
    }
}

/// One row in the date-grouped list. Mirrors the existing
/// dashboard `SetRow` and is kept here so the list is a
/// self-contained component. Only shows the exercise name,
/// notes, and the reps × weight — the time of day and the
/// date are intentionally omitted to keep the list
/// scannable.
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
            Text("\(entry.reps) × \(formattedWeight)")
                .font(.body.weight(.semibold).monospacedDigit())
                .foregroundStyle(DSColors.text)
        }
        .padding(.vertical, DSSpacing.sm)
        .padding(.horizontal, DSSpacing.md)
        .contentShape(Rectangle())
    }

    private var formattedWeight: String {
        String(format: "%.1f %@", entry.weight, weightUnit)
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

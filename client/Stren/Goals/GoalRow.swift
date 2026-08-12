import SwiftUI

/// Single goal row in `GoalsListView`. Mirrors the web's
/// `GoalCard` (`internal/views/goals/list.templ`) so the
/// card has the same anatomy: a status dot, the title, the
/// date chips, and the action buttons. Long-press / swipe
/// affordances match iOS conventions rather than the web's
/// htmx behavior.
///
/// `onMarkComplete` / `onReopen` / `onEdit` / `onDelete`
/// are injected so the row stays a pure view — all network
/// and state mutations live in `GoalStore`.
///
/// The row animates its own opacity + strikethrough when
/// `goal.isCompleted` flips (see the `.animation` modifier
/// at the bottom of `body`), so the parent doesn't have to
/// drive anything — it just toggles the value.
struct GoalRow: View {
    let goal: GoalDTO
    let onMarkComplete: () -> Void
    let onReopen: () -> Void
    let onEdit: () -> Void
    let onDelete: () -> Void

    var body: some View {
        HStack(alignment: .top, spacing: DSSpacing.md) {
            statusDot
            VStack(alignment: .leading, spacing: DSSpacing.xs) {
                title
                if !goal.description.isEmpty {
                    Text(goal.description)
                        .font(.subheadline)
                        .foregroundStyle(DSColors.textSecondary)
                        .lineLimit(3)
                }
                dateChips
                actions
            }
            Spacer(minLength: 0)
        }
        .padding(DSSpacing.md)
        .background(
            RoundedRectangle(cornerRadius: DSSpacing.cornerRadius, style: .continuous)
                .fill(DSColors.surface)
        )
        .overlay(
            RoundedRectangle(cornerRadius: DSSpacing.cornerRadius, style: .continuous)
                .stroke(DSColors.separator, lineWidth: 0.5)
        )
        .opacity(goal.isCompleted ? 0.65 : 1)
        .animation(.easeInOut(duration: 0.25), value: goal.isCompleted)
    }

    // MARK: - Subviews

    /// Filled accent dot for active goals; muted dot for
    /// completed. Mirrors the web's `bg-primary` /
    /// `bg-muted-foreground/30` status dot.
    private var statusDot: some View {
        Circle()
            .fill(goal.isCompleted ? DSColors.textSecondary.opacity(0.35) : DSColors.accent)
            .frame(width: 10, height: 10)
            .padding(.top, DSSpacing.xs)
    }

    private var title: some View {
        Text(goal.title)
            .font(.headline)
            .foregroundStyle(DSColors.text)
            .strikethrough(goal.isCompleted, color: DSColors.textSecondary)
    }

    /// Renders the optional date chips ("Started", "Target",
    /// "Ended", "Done") in the same order the web's
    /// `GoalDateInline` helper does. Hidden on the smallest
    /// screens to keep the row legible.
    @ViewBuilder
    private var dateChips: some View {
        let chips = GoalDateChips.from(goal: goal)
        if !chips.isEmpty {
            FlowLayout(spacing: DSSpacing.xs) {
                ForEach(chips) { chip in
                    GoalDateChipView(chip: chip)
                }
            }
        }
    }

    private var actions: some View {
        HStack(spacing: DSSpacing.xs) {
            if goal.isCompleted {
                Button(action: onReopen) {
                    Label("Reopen", systemImage: "arrow.uturn.backward")
                        .labelStyle(.titleAndIcon)
                }
                .buttonStyle(.dsSecondary)
            } else {
                Button(action: onMarkComplete) {
                    Label("Mark Complete", systemImage: "checkmark")
                        .labelStyle(.titleAndIcon)
                }
                .buttonStyle(.dsSecondary)
            }
            Button(action: onEdit) {
                Label("Edit", systemImage: "pencil")
                    .labelStyle(.titleAndIcon)
            }
            .buttonStyle(.dsSecondary)
        }
        .font(.footnote.weight(.semibold))
        .padding(.top, DSSpacing.xxs)
    }
}

// MARK: - Date chip model

/// Lightweight model for a single date chip on the goal
/// row. Kept inside `GoalRow.swift` because nothing else
/// uses it — it's a presentational detail of this view.
private struct GoalDateChip: Identifiable {
    enum Kind {
        case started, target(daysLeft: Int?), ended, done
    }
    let id = UUID()
    let label: String
    let kind: Kind

    var background: Color {
        switch kind {
        case .target(let daysLeft):
            // Highlight "today" / "N days" with the accent
            // surface; otherwise a neutral chip.
            return daysLeft != nil ? DSColors.accentSubtle : DSColors.surfaceElevated
        default:
            return DSColors.surfaceElevated
        }
    }

    var foreground: Color {
        switch kind {
        case .target(let daysLeft):
            return daysLeft != nil ? DSColors.onAccentSubtle : DSColors.textSecondary
        default:
            return DSColors.textSecondary
        }
    }
}

/// Computes the chips for a goal. Mirrors the web's
/// `GoalDateInline` helper at
/// `internal/views/goals/list.templ`. The "today" / "N days"
/// target badge mirrors `models.Goal.DaysUntilTarget` (the
/// server-side equivalent is in `internal/models/goal.go`).
private struct GoalDateChips {
    let chips: [GoalDateChip]

    static func from(goal: GoalDTO) -> [GoalDateChip] {
        var out: [GoalDateChip] = []
        if let start = goal.startDate {
            out.append(GoalDateChip(
                label: "Started \(Self.long(start))",
                kind: .started
            ))
        }
        if let target = goal.targetDate {
            let daysLeft = Self.daysUntilTarget(from: Date(), to: target)
            let badge = daysLeft.map { d -> String in
                if d == 0 { return "Today" }
                return "\(d)d"
            }
            let label = badge.map { "Target \(Self.long(target)) · \($0)" }
                ?? "Target \(Self.long(target))"
            out.append(GoalDateChip(
                label: label,
                kind: .target(daysLeft: badge != nil ? daysLeft : nil)
            ))
        }
        if let end = goal.endDate {
            out.append(GoalDateChip(
                label: "Ended \(Self.long(end))",
                kind: .ended
            ))
        }
        if let done = goal.completedAt {
            out.append(GoalDateChip(
                label: "Done \(Self.long(done))",
                kind: .done
            ))
        }
        return out
    }

    static func long(_ date: Date) -> String {
        date.formatted(.dateTime.day().month(.abbreviated))
    }

    /// Mirrors `models.Goal.DaysUntilTarget` — truncated to
    /// whole calendar days so a target later today reads as 0
    /// (Today) rather than 1.
    static func daysUntilTarget(from now: Date, to target: Date) -> Int? {
        let calendar = Calendar.current
        let start = calendar.startOfDay(for: now)
        let end = calendar.startOfDay(for: target)
        let days = calendar.dateComponents([.day], from: start, to: end).day ?? 0
        return days >= 0 ? days : nil
    }
}

/// Visual chip for the date row. Background / foreground
/// come from `GoalDateChip` so the "Target · Today" chip
/// can use the accent surface and everything else falls
/// back to the neutral elevated surface.
private struct GoalDateChipView: View {
    let chip: GoalDateChip

    var body: some View {
        Text(chip.label)
            .font(.caption.weight(.medium))
            .foregroundStyle(chip.foreground)
            .padding(.horizontal, DSSpacing.xs)
            .padding(.vertical, DSSpacing.xxs)
            .background(
                RoundedRectangle(cornerRadius: 6, style: .continuous)
                    .fill(chip.background)
            )
    }
}

/// Minimal flow layout — wraps chips onto multiple lines
/// when they overflow. SwiftUI's `Grid` doesn't wrap, so a
/// simple HStack + width-bounded layout works for the two
/// or three chips a goal ever has.
private struct FlowLayout: Layout {
    var spacing: CGFloat = DSSpacing.xs

    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        let maxWidth = proposal.width ?? .infinity
        let rows = layoutRows(in: maxWidth, subviews: subviews)
        let height = rows.reduce(CGFloat(0)) { partial, row in
            partial + row.height
        } + spacing * CGFloat(max(0, rows.count - 1))
        return CGSize(width: maxWidth.isFinite ? maxWidth : rows.map(\.width).max() ?? 0, height: height)
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        let rows = layoutRows(in: bounds.width, subviews: subviews)
        var y = bounds.minY
        for row in rows {
            var x = bounds.minX
            for index in row.indices {
                let size = subviews[index].sizeThatFits(.unspecified)
                subviews[index].place(at: CGPoint(x: x, y: y), proposal: ProposedViewSize(size))
                x += size.width + spacing
            }
            y += row.height + spacing
        }
    }

    private func layoutRows(in maxWidth: CGFloat, subviews: Subviews) -> [Row] {
        var rows: [Row] = [Row()]
        var currentWidth: CGFloat = 0
        for index in subviews.indices {
            let size = subviews[index].sizeThatFits(.unspecified)
            if currentWidth + size.width > maxWidth, !rows[rows.count - 1].indices.isEmpty {
                rows.append(Row())
                currentWidth = 0
            }
            rows[rows.count - 1].indices.append(index)
            rows[rows.count - 1].width += size.width + spacing
            rows[rows.count - 1].height = max(rows[rows.count - 1].height, size.height)
            currentWidth += size.width + spacing
        }
        return rows
    }

    private struct Row {
        var indices: [Int] = []
        var width: CGFloat = 0
        var height: CGFloat = 0
    }
}

#Preview {
    let active = GoalDTO(
        id: "g1",
        title: "Bench 100kg for 5 reps",
        description: "Hit a new PB on bench.",
        startDate: Date().addingTimeInterval(-7 * 24 * 3600),
        targetDate: Date().addingTimeInterval(14 * 24 * 3600),
        endDate: nil,
        completedAt: nil,
        createdAt: Date(),
        updatedAt: Date()
    )
    let completed = GoalDTO(
        id: "g2",
        title: "Run a 5k",
        description: "Non-stop.",
        startDate: Date().addingTimeInterval(-30 * 24 * 3600),
        targetDate: Date().addingTimeInterval(-1 * 24 * 3600),
        endDate: nil,
        completedAt: Date().addingTimeInterval(-2 * 24 * 3600),
        createdAt: Date(),
        updatedAt: Date()
    )
    return VStack(spacing: DSSpacing.md) {
        GoalRow(goal: active, onMarkComplete: {}, onReopen: {}, onEdit: {}, onDelete: {})
        GoalRow(goal: completed, onMarkComplete: {}, onReopen: {}, onEdit: {}, onDelete: {})
    }
    .padding()
    .background(DSColors.background)
}
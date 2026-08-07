import SwiftUI

/// SF Symbol name constants. Centralised so view code reads
/// `Icons.exercises` instead of the magic string `"dumbbell"`,
/// and so the iOS icon set stays in lockstep with the web's
/// Lucide icon set (the web's `dumbbell`, `plus`, etc. are
/// matched 1:1 to the symbols below).
public enum Icons {

    // MARK: - Tab bar

    public static let today     = "list.bullet.rectangle"
    public static let exercises = "dumbbell"
    public static let profile   = "person.crop.circle"

    // MARK: - Dashboard

    /// Toolbar "add set" button (and the "Add Set" action card).
    public static let addSet = "plus"
    /// "Timer" action card.
    public static let timer = "timer"
    /// Empty-state for the dashboard's "no sets yet" view.
    public static let dumbbellLarge = "dumbbell"
    /// Empty-state for the per-exercise history view.
    public static let chartEmpty = "chart.line.uptrend.xyaxis"

    // MARK: - Forms

    /// "Add another row" affordance in the new-set form.
    public static let plusCircle = "plus.circle"

    // MARK: - Status

    /// Generic error / load-failure state.
    public static let warning = "exclamationmark.triangle"
    /// Success confirmation (toast icon, etc.).
    public static let success = "checkmark.circle.fill"
    /// Failure in destructive confirmations.
    public static let failure = "xmark.octagon.fill"

    // MARK: - Appearance toggle

    /// "Switch to light mode" icon (shown in dark mode).
    public static let sun  = "sun.max"
    /// "Switch to dark mode" icon (shown in light mode).
    public static let moon = "moon.fill"
}

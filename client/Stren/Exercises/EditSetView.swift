import SwiftUI

/// Edit sheet for a single set. Mirrors the web's
/// `EditExerciseEntryForm` in
/// `internal/views/exercise/exercise_entry_form.templ:290-415`:
/// single set, exercise is read-only (you can't change
/// which exercise a set belongs to), and the user can
/// adjust reps, weight, rest, notes, and the date.
///
/// Presented by the per-exercise history view's swipe-edit
/// action. On Save the form PUTs the update via
/// `APIClient.updateExerciseEntry` and calls `onUpdated`
/// with the new server-confirmed `ExerciseEntryDTO` so
/// the parent can splice it into the visible page without
/// re-fetching the whole list.
struct EditSetView: View {
    @EnvironmentObject private var env: AppEnvironment
    @EnvironmentObject private var authStore: AuthStore
    @Environment(\.dismiss) private var dismiss

    /// The entry as it currently lives on the page. Used
    /// to pre-populate the form fields and as the identity
    /// for the PUT request.
    let exerciseEntry: ExerciseEntryDTO

    /// Called after a successful save with the
    /// server-confirmed entry (which may differ slightly
    /// from the local copy — e.g. server-normalised date).
    let onUpdated: (ExerciseEntryDTO) -> Void

    @State private var repsText: String = ""
    @State private var weightText: String = ""
    @State private var restText: String = ""
    @State private var notes: String = ""
    @State private var createdAt: Date = Date()
    @State private var isSaving: Bool = false
    @State private var errorMessage: String?

    private var weightUnit: String {
        authStore.currentUser?.weightUnit ?? "kg"
    }

    private var parsedReps: Int? {
        let trimmed = repsText.trimmingCharacters(in: .whitespaces)
        return trimmed.isEmpty ? nil : Int(trimmed)
    }

    private var parsedWeight: Double? {
        let trimmed = weightText.trimmingCharacters(in: .whitespaces)
        return trimmed.isEmpty ? nil : Double(trimmed)
    }

    private var parsedRest: Int {
        let trimmed = restText.trimmingCharacters(in: .whitespaces)
        return Int(trimmed) ?? 0
    }

    /// `true` when reps and weight are both parseable and
    /// reps is at least 1. Weight can be 0 for bodyweight
    /// exercises (the server accepts 0). Rest time defaults
    /// to 0 when blank.
    private var canSave: Bool {
        guard !isSaving else { return false }
        guard let reps = parsedReps, reps > 0 else { return false }
        guard let weight = parsedWeight, weight >= 0 else { return false }
        return true
    }

    var body: some View {
        NavigationStack {
            Form {
                exerciseSection
                setsSection
                notesSection
                timestampSection
                if let errorMessage {
                    Section {
                        Text(errorMessage)
                            .font(.footnote)
                            .foregroundStyle(DSColors.destructive)
                    }
                }
            }
            .navigationTitle("Edit Set")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("Cancel") { dismiss() }
                        .disabled(isSaving)
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button {
                        Task { await save() }
                    } label: {
                        if isSaving {
                            ProgressView()
                        } else {
                            Text("Save").bold()
                        }
                    }
                    .disabled(!canSave)
                }
            }
        }
        .task { populateFromEntry() }
    }

    // MARK: - Sections

    /// Exercise is read-only on the edit form. The web's
    /// `EditExerciseEntryForm` also disables this field —
    /// a set can't be re-assigned to a different exercise
    /// because the server's update endpoint validates the
    /// `exercise_id` against the entry's existing
    /// relationship.
    private var exerciseSection: some View {
        Section("Exercise") {
            HStack {
                Text(exerciseEntry.exerciseName)
                    .foregroundStyle(DSColors.text)
                Spacer()
                Image(systemName: "lock.fill")
                    .font(.caption2)
                    .foregroundStyle(DSColors.textSecondary)
            }
        }
    }

    /// Reps / weight / rest in a single row so the form
    /// fits on one screen on the smallest iPhone. Mirrors
    /// the web's three-column grid on `sm:` and stacks
    /// vertically below it.
    private var setsSection: some View {
        Section("Set") {
            HStack(spacing: DSSpacing.xs) {
                VStack(alignment: .leading, spacing: 2) {
                    Text("Reps")
                        .font(.caption)
                        .foregroundStyle(DSColors.textSecondary)
                    TextField("0", text: $repsText)
                        .keyboardType(.numberPad)
                        .textFieldStyle(.ds)
                }
                VStack(alignment: .leading, spacing: 2) {
                    Text("Weight (\(weightUnit))")
                        .font(.caption)
                        .foregroundStyle(DSColors.textSecondary)
                    TextField("0.0", text: $weightText)
                        .keyboardType(.decimalPad)
                        .textFieldStyle(.ds)
                }
                VStack(alignment: .leading, spacing: 2) {
                    Text("Rest (s)")
                        .font(.caption)
                        .foregroundStyle(DSColors.textSecondary)
                    TextField("0", text: $restText)
                        .keyboardType(.numberPad)
                        .textFieldStyle(.ds)
                }
            }
        }
    }

    private var notesSection: some View {
        Section("Notes") {
            TextField("Optional", text: $notes, axis: .vertical)
                .lineLimit(1...4)
        }
    }

    private var timestampSection: some View {
        Section("When") {
            DatePicker(
                "Date & Time",
                selection: $createdAt,
                displayedComponents: [.date, .hourAndMinute]
            )
        }
    }

    // MARK: - Data

    /// Hydrates the editable state from the entry passed
    /// in. Runs once when the sheet appears.
    private func populateFromEntry() {
        repsText = String(exerciseEntry.reps)
        weightText = String(format: "%.1f", exerciseEntry.weight)
        restText = exerciseEntry.restTime > 0 ? String(exerciseEntry.restTime) : ""
        notes = exerciseEntry.notes
        createdAt = exerciseEntry.createdAt
    }

    private func save() async {
        errorMessage = nil
        guard let reps = parsedReps, reps > 0,
              let weight = parsedWeight, weight >= 0 else {
            errorMessage = "Reps must be at least 1 and weight must be a non-negative number."
            return
        }
        isSaving = true
        defer { isSaving = false }
        do {
            let updated = try await env.api.updateExerciseEntry(
                id: exerciseEntry.id,
                request: UpdateExerciseEntryRequest(
                    exerciseID: exerciseEntry.exerciseID,
                    notes: notes,
                    reps: reps,
                    weight: weight,
                    restTime: parsedRest,
                    createdAt: createdAt
                )
            )
            onUpdated(updated)
            dismiss()
        } catch let error as APIError {
            errorMessage = error.errorDescription
        } catch {
            errorMessage = "Could not save your changes."
        }
    }
}

#Preview {
    EditSetView(
        exerciseEntry: ExerciseEntryDTO(
            id: "entry-1",
            exerciseID: "ex-1",
            exerciseName: "Squat",
            reps: 5,
            weight: 80,
            notes: "Felt strong",
            restTime: 120,
            createdAt: Date()
        ),
        onUpdated: { _ in }
    )
    .environmentObject(AppEnvironment.live(baseURL: URL(string: "http://localhost:8080/api/v1")!))
    .environmentObject(AuthStore(api: APIClient(
        baseURL: URL(string: "http://localhost:8080/api/v1")!,
        tokenProvider: { nil }
    )))
}

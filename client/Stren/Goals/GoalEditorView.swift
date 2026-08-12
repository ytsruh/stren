import SwiftUI

/// Create / edit sheet for a goal. Mirrors the web's
/// `/goals/new` and `/goals/:id/edit` forms
/// (`internal/views/goals/form.templ`): title, optional
/// description, and three optional date pickers (start,
/// target, end). Save POSTs (create) or PUTs (edit) the
/// corresponding `/api/v1/goals/*` endpoint via the shared
/// `GoalStore` injected by the parent.
struct GoalEditorView: View {
    enum Mode {
        case create
        case edit(GoalDTO)
    }

    @EnvironmentObject private var env: AppEnvironment
    @Environment(\.dismiss) private var dismiss

    let mode: Mode
    /// Shared with `GoalsListView` so the list view picks
    /// up the new / edited row in place (optimistic
    /// insertion + server confirmation).
    @ObservedObject var store: GoalStore

    @State private var title: String = ""
    @State private var description: String = ""

    /// Each date has its own "enabled" flag so the user can
    /// independently toggle a date without losing the others.
    /// The picker is hidden when the flag is off, and the
    /// underlying `Date` is sent as `nil` so the server
    /// clears that column.
    @State private var hasStartDate: Bool = false
    @State private var hasTargetDate: Bool = false
    @State private var hasEndDate: Bool = false

    @State private var startDate: Date = .now
    @State private var targetDate: Date = .now
    @State private var endDate: Date = .now

    @State private var isSaving: Bool = false
    @State private var errorMessage: String?

    private var isEditing: Bool {
        if case .edit = mode { return true }
        return false
    }

    private var editingGoalID: String? {
        if case .edit(let goal) = mode { return goal.id }
        return nil
    }

    private var trimmedTitle: String {
        title.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private var canSave: Bool {
        guard !isSaving else { return false }
        let count = trimmedTitle.count
        return count >= 1 && count <= 200
    }

    var body: some View {
        NavigationStack {
            Form {
                titleSection
                descriptionSection
                dateSection(
                    label: "Start date",
                    isOn: $hasStartDate,
                    date: $startDate,
                    footer: "Optional. When you plan to start working on this goal."
                )
                dateSection(
                    label: "Target date",
                    isOn: $hasTargetDate,
                    date: $targetDate,
                    footer: "Optional. When you aim to complete this goal. Used for the \"Today / N days\" chip."
                )
                dateSection(
                    label: "End date",
                    isOn: $hasEndDate,
                    date: $endDate,
                    footer: "Optional. The last day you want to give yourself to finish."
                )
                if let errorMessage {
                    Section {
                        Text(errorMessage)
                            .font(.footnote)
                            .foregroundStyle(DSColors.destructive)
                    }
                }
            }
            .navigationTitle(isEditing ? "Edit goal" : "New goal")
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
            .onAppear { seedIfNeeded() }
        }
    }

    // MARK: - Sections

    private var titleSection: some View {
        Section {
            TextField("Title", text: $title, axis: .vertical)
                .lineLimit(1...3)
                .textInputAutocapitalization(.sentences)
        } header: {
            Text("Title")
        } footer: {
            Text("Up to 200 characters.")
        }
    }

    private var descriptionSection: some View {
        Section {
            TextField("Notes (optional)", text: $description, axis: .vertical)
                .lineLimit(1...6)
        } header: {
            Text("Description")
        } footer: {
            Text("Up to 2000 characters.")
        }
    }

    /// Reusable date-section builder. The toggle is bound
    /// separately so the user can clear a date without
    /// touching the picker.
    @ViewBuilder
    private func dateSection(
        label: String,
        isOn: Binding<Bool>,
        date: Binding<Date>,
        footer: String
    ) -> some View {
        Section {
            Toggle(isOn: isOn) {
                Text(label)
            }
            if isOn.wrappedValue {
                DatePicker(
                    label,
                    selection: date,
                    displayedComponents: .date
                )
                .labelsHidden()
                .datePickerStyle(.graphical)
            }
        } footer: {
            Text(footer)
        }
    }

    // MARK: - Save

    /// Posts (create) or PUTs (edit) via the same
    /// `GoalStore` API the list view uses. The store's own
    /// mutation methods run on the main actor, so we don't
    /// need any explicit `MainActor.run` block here.
    private func save() async {
        errorMessage = nil
        let trimmed = trimmedTitle
        guard trimmed.count >= 1, trimmed.count <= 200 else {
            errorMessage = "Title must be between 1 and 200 characters."
            return
        }
        isSaving = true
        defer { isSaving = false }

        let request = CreateGoalRequest(
            title: trimmed,
            description: description.trimmingCharacters(in: .whitespacesAndNewlines),
            startDate: hasStartDate ? startDate : nil,
            targetDate: hasTargetDate ? targetDate : nil,
            endDate: hasEndDate ? endDate : nil
        )
        do {
            switch mode {
            case .create:
                let created = await store.create(request)
                if created == nil {
                    // The store already populated
                    // errorMessage; surface it.
                    errorMessage = store.errorMessage ?? "Could not save the goal."
                    return
                }
            case .edit(let goal):
                let update = UpdateGoalRequest(
                    title: request.title,
                    description: request.description,
                    startDate: request.startDate,
                    targetDate: request.targetDate,
                    endDate: request.endDate
                )
                await store.update(id: goal.id, request: update)
                if let msg = store.errorMessage {
                    errorMessage = msg
                    return
                }
            }
            dismiss()
        } catch let error as APIError {
            errorMessage = error.errorDescription
        } catch {
            errorMessage = "Could not save the goal."
        }
    }

    // MARK: - Seeding

    /// Populates the form from the existing goal on first
    /// appear (edit mode only). Skipped when the title is
    /// already populated so a SwiftUI re-render mid-edit
    /// doesn't wipe the user's typing.
    private func seedIfNeeded() {
        guard case .edit(let goal) = mode else { return }
        guard title.isEmpty else { return }
        title = goal.title
        description = goal.description
        if let start = goal.startDate {
            hasStartDate = true
            startDate = start
        }
        if let target = goal.targetDate {
            hasTargetDate = true
            targetDate = target
        }
        if let end = goal.endDate {
            hasEndDate = true
            endDate = end
        }
    }
}

#Preview("Create") {
    GoalEditorView(
        mode: .create,
        store: GoalStore(api: APIClient(
            baseURL: URL(string: "http://localhost:8080/api/v1")!,
            tokenProvider: { nil }
        ))
    )
    .environmentObject(AppEnvironment.live(baseURL: URL(string: "http://localhost:8080/api/v1")!))
    .environmentObject(AuthStore(api: APIClient(
        baseURL: URL(string: "http://localhost:8080/api/v1")!,
        tokenProvider: { nil }
    )))
}

#Preview("Edit") {
    GoalEditorView(
        mode: .edit(GoalDTO(
            id: "g1",
            title: "Bench 100kg",
            description: "Working sets",
            startDate: Date(),
            targetDate: Date().addingTimeInterval(14 * 24 * 3600),
            endDate: nil,
            completedAt: nil,
            createdAt: Date(),
            updatedAt: Date()
        )),
        store: GoalStore(api: APIClient(
            baseURL: URL(string: "http://localhost:8080/api/v1")!,
            tokenProvider: { nil }
        ))
    )
    .environmentObject(AppEnvironment.live(baseURL: URL(string: "http://localhost:8080/api/v1")!))
    .environmentObject(AuthStore(api: APIClient(
        baseURL: URL(string: "http://localhost:8080/api/v1")!,
        tokenProvider: { nil }
    )))
}
import SwiftUI
import UIKit

/// The "Goals" tab. Mirrors the web's `/goals` page
/// (`internal/views/goals/list.templ`): an "Active goals"
/// section with the new/edit/complete controls, and a
/// collapsible "Completed goals" section underneath.
///
/// All networking and state lives in `GoalStore`; this view
/// is purely presentational. The store is injected by the
/// parent (`MainTabView` constructs it once from
/// `AppEnvironment.api` so the store can outlive view
/// rebuilds and tests can supply a custom store).
struct GoalsListView: View {
    @EnvironmentObject private var env: AppEnvironment
    @ObservedObject var store: GoalStore

    @State private var showingNewGoal: Bool = false
    @State private var editingGoal: GoalDTO?
    @State private var deletingGoal: GoalDTO?

    var body: some View {
        NavigationStack {
            content
                .navigationTitle("Goals")
                .toolbar { toolbarContent }
                .sheet(isPresented: $showingNewGoal) {
                    GoalEditorView(mode: .create, store: store)
                        .environmentObject(env)
                }
                .sheet(item: $editingGoal) { goal in
                    GoalEditorView(mode: .edit(goal), store: store)
                        .environmentObject(env)
                }
                .confirmationDialog(
                    "Delete this goal?",
                    isPresented: Binding(
                        get: { deletingGoal != nil },
                        set: { if !$0 { deletingGoal = nil } }
                    ),
                    titleVisibility: .visible,
                    presenting: deletingGoal
                ) { goal in
                    Button("Delete goal", role: .destructive) {
                        Task { await store.delete(id: goal.id) }
                        deletingGoal = nil
                    }
                    Button("Cancel", role: .cancel) {
                        deletingGoal = nil
                    }
                } message: { _ in
                    Text("This permanently removes the goal and cannot be undone.")
                }
        }
        .task { await store.load() }
        .refreshable { await store.load() }
    }

    // MARK: - Toolbar

    @ToolbarContentBuilder
    private var toolbarContent: some ToolbarContent {
        ToolbarItem(placement: .topBarTrailing) {
            Button {
                showingNewGoal = true
            } label: {
                Image(systemName: Icons.plusCircle)
            }
            .accessibilityLabel("Add goal")
        }
    }

    // MARK: - Content states

    @ViewBuilder
    private var content: some View {
        if store.isLoading && store.goals.isEmpty {
            ProgressView()
                .frame(maxWidth: .infinity, maxHeight: .infinity)
        } else if let error = store.errorMessage, store.goals.isEmpty {
            errorState(error)
        } else if store.goals.isEmpty {
            emptyState
        } else {
            loadedList
        }
    }

    /// Active section + collapsible completed section. The
    /// `Section` ordering matches the web: active first,
    /// then completed in a `<details>`-style accordion.
    private var loadedList: some View {
        List {
            activeSection
            if !store.completedGoals.isEmpty {
                completedSection
            }
        }
        .listStyle(.insetGrouped)
        .scrollContentBackground(.hidden)
        .background(DSColors.background.ignoresSafeArea())
    }

    @ViewBuilder
    private var activeSection: some View {
        Section {
            if store.activeGoals.isEmpty {
                emptyActiveHint
            } else {
                ForEach(store.activeGoals) { goal in
                    GoalRow(
                        goal: goal,
                        onMarkComplete: { completeGoal(goal) },
                        onReopen: { reopenGoal(goal) },
                        onEdit: { editingGoal = goal },
                        onDelete: { deletingGoal = goal }
                    )
                    .listRowInsets(EdgeInsets(top: DSSpacing.xs, leading: DSSpacing.md, bottom: DSSpacing.xs, trailing: DSSpacing.md))
                    .listRowBackground(Color.clear)
                    .listRowSeparator(.hidden)
                }
            }
        } header: {
            sectionHeader(
                title: "Active",
                count: store.activeGoals.count
            )
        }
    }

    @ViewBuilder
    private var completedSection: some View {
        Section {
            ForEach(store.completedGoals) { goal in
                GoalRow(
                    goal: goal,
                    onMarkComplete: { completeGoal(goal) },
                    onReopen: { reopenGoal(goal) },
                    onEdit: { editingGoal = goal },
                    onDelete: { deletingGoal = goal }
                )
                .listRowInsets(EdgeInsets(top: DSSpacing.xs, leading: DSSpacing.md, bottom: DSSpacing.xs, trailing: DSSpacing.md))
                .listRowBackground(Color.clear)
                .listRowSeparator(.hidden)
            }
        } header: {
            sectionHeader(
                title: "Completed",
                count: store.completedGoals.count
            )
        }
    }

    private func sectionHeader(title: String, count: Int) -> some View {
        HStack {
            Text(title)
                .font(.headline)
                .foregroundStyle(DSColors.text)
            Text("(\(count))")
                .font(.subheadline)
                .foregroundStyle(DSColors.textSecondary)
        }
        .textCase(nil)
    }

    private var emptyActiveHint: some View {
        Text("No active goals — tap + to add one.")
            .font(.subheadline)
            .foregroundStyle(DSColors.textSecondary)
            .padding(.vertical, DSSpacing.xs)
    }

    private var emptyState: some View {
        VStack(spacing: DSSpacing.md) {
            Image(systemName: Icons.goals)
                .font(.system(size: 48))
                .foregroundStyle(DSColors.textSecondary)
            Text("No goals yet")
                .font(.title3.weight(.semibold))
                .foregroundStyle(DSColors.text)
            Text("Set a goal to keep your training on track.")
                .font(.body)
                .foregroundStyle(DSColors.textSecondary)
                .multilineTextAlignment(.center)
            Button {
                showingNewGoal = true
            } label: {
                Label("Add your first goal", systemImage: Icons.plusCircle)
            }
            .buttonStyle(.dsPrimary)
        }
        .padding(DSSpacing.lg)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private func errorState(_ message: String) -> some View {
        VStack(spacing: DSSpacing.md) {
            Image(systemName: Icons.warning)
                .font(.largeTitle)
                .foregroundStyle(DSColors.destructive)
            Text(message)
                .multilineTextAlignment(.center)
                .foregroundStyle(DSColors.textSecondary)
            Button("Try again") {
                Task { await store.load() }
            }
            .buttonStyle(.dsSecondary)
        }
        .padding(DSSpacing.lg)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    // MARK: - Mutation handlers

    /// Marks a goal complete. The optimistic update happens
    /// inside `GoalStore.markComplete(id:)`; we additionally
    /// fire the success haptic here (alongside the row's own
    /// opacity/strikethrough animation) so the user gets
    /// tactile confirmation that the goal is done.
    private func completeGoal(_ goal: GoalDTO) {
        guard !goal.isCompleted else { return }
        let generator = UINotificationFeedbackGenerator()
        generator.prepare()
        generator.notificationOccurred(.success)
        Task { await store.markComplete(id: goal.id) }
    }

    /// Reopens a completed goal. Uses a lighter haptic (a
    /// "selection" tap) rather than the success chime so
    /// reopening feels less celebratory — matches the web's
    /// "Goal reopened" toast that has no confetti.
    private func reopenGoal(_ goal: GoalDTO) {
        guard goal.isCompleted else { return }
        let generator = UISelectionFeedbackGenerator()
        generator.prepare()
        generator.selectionChanged()
        Task { await store.reopen(id: goal.id) }
    }
}

#Preview {
    GoalsListView(store: GoalStore(api: APIClient(
        baseURL: URL(string: "http://localhost:8080/api/v1")!,
        tokenProvider: { nil }
    )))
    .environmentObject(AppEnvironment.live(baseURL: URL(string: "http://localhost:8080/api/v1")!))
    .environmentObject(AuthStore(api: APIClient(
        baseURL: URL(string: "http://localhost:8080/api/v1")!,
        tokenProvider: { nil }
    )))
}
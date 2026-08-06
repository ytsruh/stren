import SwiftUI

/// The "Exercises" tab. Lists every exercise in the
/// catalogue with a search field at the top. Tap an
/// exercise to navigate to `ExerciseHistoryView` for that
/// exercise.
struct ExerciseListView: View {
    @EnvironmentObject private var env: AppEnvironment

    @State private var exercises: [ExerciseDTO] = []
    @State private var isLoading: Bool = true
    @State private var errorMessage: String?
    @State private var search: String = ""

    var body: some View {
        NavigationStack {
            content
                .navigationTitle("Exercises")
                .searchable(text: $search, placement: .navigationBarDrawer(displayMode: .always), prompt: "Search exercises")
        }
        .task { await load() }
        .refreshable { await load() }
    }

    @ViewBuilder
    private var content: some View {
        if isLoading && exercises.isEmpty {
            ProgressView()
                .frame(maxWidth: .infinity, maxHeight: .infinity)
        } else if let errorMessage, exercises.isEmpty {
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
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        } else {
            List {
                ForEach(filtered) { exercise in
                    NavigationLink(value: exercise) {
                        ExerciseRow(exercise: exercise)
                    }
                }
            }
            .listStyle(.insetGrouped)
            .navigationDestination(for: ExerciseDTO.self) { exercise in
                ExerciseHistoryView(exercise: exercise)
            }
        }
    }

    private var filtered: [ExerciseDTO] {
        let q = search.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        if q.isEmpty { return exercises }
        return exercises.filter { $0.name.lowercased().contains(q) }
    }

    private func load() async {
        errorMessage = nil
        isLoading = true
        defer { isLoading = false }
        do {
            exercises = try await env.api.listExercises()
        } catch let error as APIError {
            if case .unauthorized = error { return }
            errorMessage = error.errorDescription
        } catch {
            errorMessage = "Could not load exercises."
        }
    }
}

/// One row in the exercise list. Shows the exercise name;
/// the description is intentionally omitted here because
/// every row is the same kind of "tap for history" link
/// and a long description under every name just adds noise.
struct ExerciseRow: View {
    let exercise: ExerciseDTO

    var body: some View {
        Text(exercise.name)
            .font(.body.weight(.semibold))
            .padding(.vertical, DSSpacing.xxs)
    }
}

#Preview {
    ExerciseListView()
        .environmentObject(AppEnvironment.live(baseURL: URL(string: "http://localhost:8080/api/v1")!))
        .environmentObject(AuthStore(api: APIClient(
            baseURL: URL(string: "http://localhost:8080/api/v1")!,
            tokenProvider: { nil }
        )))
}

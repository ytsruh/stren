import SwiftUI

/// The single entry point. Builds the live `AppEnvironment`
/// at launch and switches the root view between the auth
/// flow and the main tab view based on `authStore.currentUser`.
///
/// The `restoreSession()` call runs on first appearance so
/// the app can rehydrate the Keychain token and the cached
/// user before the user sees anything other than a brief
/// splash. While that's running, `RootView` shows a spinner
/// (see `isRestoring` on `AuthStore`).
@main
struct StrenApp: App {
    @StateObject private var env = AppEnvironment.live()

    var body: some Scene {
        WindowGroup {
            RootView()
                .environmentObject(env)
                .environmentObject(env.authStore)
                .applyThemeMode()
                .task {
                    await env.authStore.restoreSession()
                }
        }
    }
}

/// The actual root view. Switches between the auth flow
/// (when no user is signed in) and the main tab view.
struct RootView: View {
    @EnvironmentObject private var env: AppEnvironment
    @EnvironmentObject private var authStore: AuthStore

    var body: some View {
        Group {
            if authStore.isRestoring {
                SplashView()
            } else if authStore.currentUser != nil {
                MainTabView()
            } else {
                LoginView()
            }
        }
        .animation(.easeInOut(duration: 0.2), value: authStore.isRestoring)
        .animation(.easeInOut(duration: 0.2), value: authStore.currentUser?.id)
    }
}

/// Lightweight splash shown while `restoreSession` is
/// running. Avoids the flash-of-login-screen on cold start
/// for users who already have a valid token.
struct SplashView: View {
    var body: some View {
        ZStack {
            DSColors.background.ignoresSafeArea()
            ProgressView()
                .controlSize(.large)
        }
    }
}

/// The main app shell — three tabs: dashboard, exercises,
/// and profile. The "new set" flow is a sheet from the
/// dashboard's toolbar so it's always one tap away.
struct MainTabView: View {
    var body: some View {
        TabView {
            DashboardView()
                .tabItem { Label("Today", systemImage: "list.bullet.rectangle") }

            ExerciseListView()
                .tabItem { Label("Exercises", systemImage: "dumbbell") }

            ProfileView()
                .tabItem { Label("Profile", systemImage: "person.crop.circle") }
        }
    }
}

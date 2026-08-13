import Foundation

// MARK: - DTOs that match the Go server's /api/v1 JSON shapes.
//
// Every struct here is a direct mirror of a DTO in
// `internal/routes/api_dto.go`. The `CodingKeys` overrides
// map Swift's camelCase to the server's snake_case so the
// network layer doesn't have to do any field renaming.
//
// Dates are decoded with the same RFC 3339 (with fractional
// seconds) format `time.Time` produces on the server, via
// `APIClient.jsonDecoder`.

// MARK: Auth

public struct LoginRequest: Encodable {
    public let email: String
    public let password: String

    public init(email: String, password: String) {
        self.email = email
        self.password = password
    }
}

public struct RegisterRequest: Encodable {
    public let name: String
    public let email: String
    public let password: String

    public init(name: String, email: String, password: String) {
        self.name = name
        self.email = email
        self.password = password
    }
}

public struct AuthResponse: Decodable {
    public let token: String
    public let user: UserDTO
}

// MARK: User

public struct UserDTO: Codable, Equatable, Identifiable {
    public let id: String
    public let name: String
    public let email: String
    public let isAdmin: Bool
    public let weightUnit: String
    public let targetWeight: Double?

    enum CodingKeys: String, CodingKey {
        case id
        case name
        case email
        case isAdmin = "is_admin"
        case weightUnit = "weight_unit"
        case targetWeight = "target_weight"
    }

    public init(
        id: String,
        name: String,
        email: String,
        isAdmin: Bool,
        weightUnit: String,
        targetWeight: Double?
    ) {
        self.id = id
        self.name = name
        self.email = email
        self.isAdmin = isAdmin
        self.weightUnit = weightUnit
        self.targetWeight = targetWeight
    }
}

// MARK: Profile

/// JSON body for `PUT /api/v1/me`. Mirrors the user-editable
/// subset of the server's `UpdateMeRequest` (name, target
/// weight, weight unit). Reminder preferences and notification
/// settings are deliberately omitted — the iOS app surfaces no
/// UI for them yet, so the iOS-only DTOs stay slim. When those
/// surfaces land on iOS, add the fields here and to `UserDTO`.
///
/// `targetWeight` is an optional pointer so an explicit `nil`
/// clears the goal (matching the HTML form's empty-input
/// behavior). The iOS edit form binds the field to a `String`
/// and converts to a `Double?` so the user can leave it blank.
public struct UpdateMeRequest: Encodable, Equatable {
    public let name: String
    public let targetWeight: Double?
    public let weightUnit: String

    enum CodingKeys: String, CodingKey {
        case name
        case targetWeight = "target_weight"
        case weightUnit = "weight_unit"
    }

    public init(
        name: String,
        targetWeight: Double?,
        weightUnit: String
    ) {
        self.name = name
        self.targetWeight = targetWeight
        self.weightUnit = weightUnit
    }
}

// MARK: Exercises

/// Mirrors the server's `ExerciseDTO` in
/// `internal/routes/api_dto.go`. The server returns an
/// `image_url` field that is the fully-qualified public URL
/// (resolved via `utils.PublicURLFor(img_url)`) and is what
/// the iOS side should use for `AsyncImage`. `imgURL` is
/// kept around for backwards-compatibility (older server
/// builds will not populate `imageURL`, in which case the
/// views should fall back to the placeholder icon).
public struct ExerciseDTO: Codable, Equatable, Identifiable, Hashable {
    public let id: String
    public let name: String
    public let description: String
    public let videoURL: String
    public let imgURL: String
    public let imageURL: String
    public let type: String

    enum CodingKeys: String, CodingKey {
        case id
        case name
        case description
        case videoURL = "video_url"
        case imgURL = "img_url"
        case imageURL = "image_url"
        case type
    }

    public init(
        id: String,
        name: String,
        description: String,
        videoURL: String,
        imgURL: String,
        imageURL: String,
        type: String
    ) {
        self.id = id
        self.name = name
        self.description = description
        self.videoURL = videoURL
        self.imgURL = imgURL
        self.imageURL = imageURL
        self.type = type
    }

    /// `true` when the exercise has a renderable image. The
    /// server returns empty strings for missing media, so
    /// reading emptiness on either field is equivalent;
    /// `imageURL` is the field the views actually use.
    public var hasImage: Bool { !imageURL.isEmpty }

    /// `true` when the exercise has a video link to open.
    public var hasVideo: Bool { !videoURL.isEmpty }

    /// Pretty display name for the `type` string. Matches the
    /// web's `capitalize` badge styling (`strength` →
    /// `Strength`, `cardio` → `Cardio`, `other` → `Other`).
    /// Unknown values fall back to a capitalized version of
    /// the raw string so we never show a lowercase badge.
    public var typeDisplayName: String {
        switch type.lowercased() {
        case "strength": return "Strength"
        case "cardio":   return "Cardio"
        case "other":    return "Other"
        default:         return type.capitalized
        }
    }
}

private extension String {
    /// Locale-aware first-letter-uppercase. Avoids pulling in
    /// `Foundation.NSString.capitalizedString` differences and
    /// matches the web's `capitalize` CSS for ASCII.
    var capitalized: String {
        guard let first = first else { return self }
        return first.uppercased() + dropFirst()
    }
}

// MARK: Exercise entries (sets)

public struct ExerciseEntryDTO: Codable, Equatable, Identifiable, Hashable {
    public let id: String
    public let exerciseID: String
    public let exerciseName: String
    public let reps: Int
    public let weight: Double
    public let notes: String
    public let restTime: Int
    public let createdAt: Date

    enum CodingKeys: String, CodingKey {
        case id
        case exerciseID = "exercise_id"
        case exerciseName = "exercise_name"
        case reps
        case weight
        case notes
        case restTime = "rest_time"
        case createdAt = "created_at"
    }
}

public struct CreateSetInput: Encodable, Equatable, Hashable {
    public let reps: Int
    public let weight: Double
    public let restTime: Int

    enum CodingKeys: String, CodingKey {
        case reps
        case weight
        case restTime = "rest_time"
    }

    public init(reps: Int, weight: Double, restTime: Int) {
        self.reps = reps
        self.weight = weight
        self.restTime = restTime
    }
}

public struct CreateExerciseEntriesRequest: Encodable {
    public let exerciseID: String
    public let notes: String
    public let createdAt: Date?
    public let sets: [CreateSetInput]

    enum CodingKeys: String, CodingKey {
        case exerciseID = "exercise_id"
        case notes
        case createdAt = "created_at"
        case sets
    }

    public init(exerciseID: String, notes: String, createdAt: Date?, sets: [CreateSetInput]) {
        self.exerciseID = exerciseID
        self.notes = notes
        self.createdAt = createdAt
        self.sets = sets
    }
}

public struct UpdateExerciseEntryRequest: Encodable {
    public let exerciseID: String
    public let notes: String
    public let reps: Int
    public let weight: Double
    public let restTime: Int
    public let createdAt: Date?

    enum CodingKeys: String, CodingKey {
        case exerciseID = "exercise_id"
        case notes
        case reps
        case weight
        case restTime = "rest_time"
        case createdAt = "created_at"
    }

    public init(
        exerciseID: String,
        notes: String,
        reps: Int,
        weight: Double,
        restTime: Int,
        createdAt: Date?
    ) {
        self.exerciseID = exerciseID
        self.notes = notes
        self.reps = reps
        self.weight = weight
        self.restTime = restTime
        self.createdAt = createdAt
    }
}

// MARK: History & chart

public struct HistoryStatsDTO: Codable, Equatable {
    public let maxWeight: Double
    public let lastSet: ExerciseEntryDTO?

    enum CodingKeys: String, CodingKey {
        case maxWeight = "max_weight"
        case lastSet = "last_set"
    }
}

public struct HistoryPageDTO: Codable, Equatable {
    public let entries: [ExerciseEntryDTO]
    public let stats: HistoryStatsDTO
    public let page: Int
    public let hasPrev: Bool
    public let hasNext: Bool

    enum CodingKeys: String, CodingKey {
        case entries
        case stats
        case page
        case hasPrev = "has_prev"
        case hasNext = "has_next"
    }
}

// MARK: Goals

/// JSON shape for a single goal returned by
/// `/api/v1/goals/*`. Mirrors the server's `GoalDTO` in
/// `internal/routes/api_dto.go`. All four date fields are
/// optional `Date`s (decoded from the server's RFC 3339
/// timestamps) so a missing date is `nil` rather than
/// `1970-01-01`, which would break the UI's conditional
/// rendering of the date chips.
///
/// `completedAt` is the source of truth for whether the
/// goal is done — use `isCompleted` rather than checking
/// `completedAt != nil` directly at call sites.
public struct GoalDTO: Codable, Equatable, Identifiable, Hashable {
    public let id: String
    public let title: String
    public let description: String
    public let startDate: Date?
    public let targetDate: Date?
    public let endDate: Date?
    public let completedAt: Date?
    public let createdAt: Date
    public let updatedAt: Date

    enum CodingKeys: String, CodingKey {
        case id
        case title
        case description
        case startDate = "start_date"
        case targetDate = "target_date"
        case endDate = "end_date"
        case completedAt = "completed_at"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }

    public init(
        id: String,
        title: String,
        description: String,
        startDate: Date?,
        targetDate: Date?,
        endDate: Date?,
        completedAt: Date?,
        createdAt: Date,
        updatedAt: Date
    ) {
        self.id = id
        self.title = title
        self.description = description
        self.startDate = startDate
        self.targetDate = targetDate
        self.endDate = endDate
        self.completedAt = completedAt
        self.createdAt = createdAt
        self.updatedAt = updatedAt
    }

    /// Convenience flag — equivalent to `completedAt != nil`
    /// but reads better at call sites.
    public var isCompleted: Bool { completedAt != nil }
}

/// Response body for `GET /api/v1/goals`. Wrapping the slice
/// in a named struct (rather than returning `[GoalDTO]`
/// directly) lets the server add fields like pagination
/// metadata without breaking the iOS contract.
public struct GoalsResponse: Decodable, Equatable {
    public let goals: [GoalDTO]
}

/// JSON body for `POST /api/v1/goals`. Mirrors the server's
/// `CreateGoalRequest`. Title is required; description and
/// all three dates are optional. The server enforces the
/// same length limits as the HTML form (title 1–200,
/// description ≤2000).
public struct CreateGoalRequest: Encodable, Equatable {
    public let title: String
    public let description: String
    public let startDate: Date?
    public let targetDate: Date?
    public let endDate: Date?

    enum CodingKeys: String, CodingKey {
        case title
        case description
        case startDate = "start_date"
        case targetDate = "target_date"
        case endDate = "end_date"
    }

    public init(
        title: String,
        description: String,
        startDate: Date?,
        targetDate: Date?,
        endDate: Date?
    ) {
        self.title = title
        self.description = description
        self.startDate = startDate
        self.targetDate = targetDate
        self.endDate = endDate
    }
}

/// JSON body for `PUT /api/v1/goals/:id`. Mirrors
/// `CreateGoalRequest` — same fields, same validation.
/// `completedAt` is intentionally NOT editable here; status
/// changes go through `markGoalComplete(id:)` /
/// `reopenGoal(id:)` so the server owns the completion
/// timestamp.
public struct UpdateGoalRequest: Encodable, Equatable {
    public let title: String
    public let description: String
    public let startDate: Date?
    public let targetDate: Date?
    public let endDate: Date?

    enum CodingKeys: String, CodingKey {
        case title
        case description
        case startDate = "start_date"
        case targetDate = "target_date"
        case endDate = "end_date"
    }

    public init(
        title: String,
        description: String,
        startDate: Date?,
        targetDate: Date?,
        endDate: Date?
    ) {
        self.title = title
        self.description = description
        self.startDate = startDate
        self.targetDate = targetDate
        self.endDate = endDate
    }
}

// MARK: Feedback

/// JSON body for `POST /api/v1/feedback`. Mirrors the
/// web app's `/feedback` form: a short `title` (5–100 chars)
/// and a longer `message` (10–1000 chars). Both are trimmed
/// and validated by `FeedbackController.Submit` on the
/// server; the iOS form mirrors those rules locally so the
/// Save button is disabled until the body is valid. The
/// `user_id` is taken from the JWT — this DTO deliberately
/// has no field for it.
public struct SubmitFeedbackRequest: Encodable, Equatable {
    public let title: String
    public let message: String

    public init(title: String, message: String) {
        self.title = title
        self.message = message
    }
}

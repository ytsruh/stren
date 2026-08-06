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

// MARK: Exercises

public struct ExerciseDTO: Codable, Equatable, Identifiable, Hashable {
    public let id: String
    public let name: String
    public let description: String
    public let videoURL: String
    public let imgURL: String
    public let type: String

    enum CodingKeys: String, CodingKey {
        case id
        case name
        case description
        case videoURL = "video_url"
        case imgURL = "img_url"
        case type
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

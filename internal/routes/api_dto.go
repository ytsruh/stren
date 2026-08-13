// Package routes: api_dto.go defines the JSON shapes the /api/v1
// namespace sends and receives. They are intentionally separate
// from the domain models so the on-the-wire contract is decoupled
// from the database schema, and so we never accidentally leak a
// field the client does not need (e.g. password hashes). The
// From… helpers below are the only way a model becomes a DTO.
package routes

import (
	"time"

	"stren/internal/models"
)

// APIError is the body returned for every non-2xx response from
// the /api/v1 namespace. The auth middleware and the per-handler
// error paths both produce this shape so the iOS client only has
// to parse a single error format.
type APIError struct {
	Error string `json:"error"`
}

// LoginRequest is the JSON body for POST /api/v1/auth/login.
type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// RegisterRequest is the JSON body for POST /api/v1/auth/register.
type RegisterRequest struct {
	Name     string `json:"name"     validate:"required,min=1,max=100"`
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

// AuthResponse is the JSON body returned by login and register.
// Token is a signed JWT the client stores in the Keychain; User
// is the safe public-facing view of the user record.
type AuthResponse struct {
	Token string  `json:"token"`
	User  UserDTO `json:"user"`
}

// UserDTO is the safe public-facing view of a user. It deliberately
// omits PasswordHash and the internal reminder scheduling fields
// (which are only useful to the server's tick job).
type UserDTO struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Email        string   `json:"email"`
	IsAdmin      bool     `json:"is_admin"`
	WeightUnit   string   `json:"weight_unit"`
	TargetWeight *float64 `json:"target_weight,omitempty"`
}

// UserFromModel converts a models.User into the safe UserDTO.
// nil is mapped to a zero-value DTO so callers don't have to
// branch before encoding the response.
func UserFromModel(u *models.User) UserDTO {
	if u == nil {
		return UserDTO{WeightUnit: "kg"}
	}
	return UserDTO{
		ID:           u.ID,
		Name:         u.Name,
		Email:        u.Email,
		IsAdmin:      u.IsAdmin,
		WeightUnit:   u.WeightUnitDisplay(),
		TargetWeight: u.TargetWeight,
	}
}

// UpdateMeRequest is the JSON body for PUT /api/v1/me. Mirrors
// the user-editable subset of the HTML profile form (name,
// target weight, weight unit) so the iOS app can update the
// same fields the web app exposes. Reminder preferences and
// notification settings are deliberately omitted: they live
// on the web app only for now, and the iOS client surfaces no
// UI for them. Add fields here (and to UserDTO) when the iOS
// app grows that surface.
//
// TargetWeight is a pointer so an omitted JSON field (or an
// explicit null) clears the user's goal, matching the form's
// empty-input semantics.
type UpdateMeRequest struct {
	Name         string   `json:"name"          validate:"required,min=2,max=100"`
	TargetWeight *float64 `json:"target_weight" validate:"omitempty,gte=0,lte=1000"`
	WeightUnit   string   `json:"weight_unit"   validate:"omitempty,oneof=kg lbs"`
}

// ExerciseDTO is the JSON shape for an exercise in any list or
// lookup response.
type ExerciseDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	VideoURL    string `json:"video_url"`
	ImgURL      string `json:"img_url"`
	Type        string `json:"type"`
}

// ExerciseFromModel converts a models.Exercise into its DTO.
func ExerciseFromModel(e models.Exercise) ExerciseDTO {
	return ExerciseDTO{
		ID:          e.ID,
		Name:        e.Name,
		Description: e.Description,
		VideoURL:    e.VideoURL,
		ImgURL:      e.ImgURL,
		Type:        string(e.Type),
	}
}

// ExercisesFromModels converts a slice of exercises into the
// DTO slice. The empty case returns an empty (non-nil) slice
// so the JSON encoder writes `[]` rather than `null` — easier
// for the Swift `Codable` decoder to consume.
func ExercisesFromModels(es []models.Exercise) []ExerciseDTO {
	out := make([]ExerciseDTO, 0, len(es))
	for _, e := range es {
		out = append(out, ExerciseFromModel(e))
	}
	return out
}

// ExerciseEntryDTO is the JSON shape for a single set.
type ExerciseEntryDTO struct {
	ID           string    `json:"id"`
	ExerciseID   string    `json:"exercise_id"`
	ExerciseName string    `json:"exercise_name"`
	Reps         int       `json:"reps"`
	Weight       float64   `json:"weight"`
	Notes        string    `json:"notes"`
	RestTime     int       `json:"rest_time"`
	CreatedAt    time.Time `json:"created_at"`
}

// ExerciseEntryFromModel converts a models.ExerciseEntry into its DTO.
func ExerciseEntryFromModel(e models.ExerciseEntry) ExerciseEntryDTO {
	return ExerciseEntryDTO{
		ID:           e.ID,
		ExerciseID:   e.ExerciseID,
		ExerciseName: e.ExerciseName,
		Reps:         e.Reps,
		Weight:       e.Weight,
		Notes:        e.Notes,
		RestTime:     e.RestTime,
		CreatedAt:    e.CreatedAt,
	}
}

// ExerciseEntriesFromModels converts a slice of exercise entries
// into the DTO slice. The empty case returns an empty (non-nil)
// slice so the JSON encoder writes `[]` rather than `null`.
func ExerciseEntriesFromModels(es []models.ExerciseEntry) []ExerciseEntryDTO {
	out := make([]ExerciseEntryDTO, 0, len(es))
	for _, e := range es {
		out = append(out, ExerciseEntryFromModel(e))
	}
	return out
}

// CreateSetInput is one set within a CreateExerciseEntriesRequest.
// The numeric limits match the HTML form (reps 1–1000, weight
// 0–5000, rest 0–3600) so a client cannot bypass the form's
// per-set validation by going through the JSON API.
type CreateSetInput struct {
	Reps     int     `json:"reps"       validate:"gte=1,lte=1000"`
	Weight   float64 `json:"weight"     validate:"gte=0,lte=5000"`
	RestTime int     `json:"rest_time"  validate:"gte=0,lte=3600"`
}

// CreateExerciseEntriesRequest is the body for
// POST /api/v1/exercise-entries. Sets is the multi-set payload
// (one entry created per set, all sharing the same exercise,
// notes and timestamp — same semantics as the web form).
// CreatedAt is optional and defaults to time.Now() on the
// server when omitted, so a client that just wants "log it
// now" can send an empty body field.
type CreateExerciseEntriesRequest struct {
	ExerciseID string           `json:"exercise_id" validate:"required"`
	Notes      string           `json:"notes"       validate:"max=500"`
	CreatedAt  *time.Time       `json:"created_at,omitempty"`
	Sets       []CreateSetInput `json:"sets"        validate:"required,min=1,dive"`
}

// UpdateExerciseEntryRequest is the body for
// PUT /api/v1/exercise-entries/:id. The single-set shape
// matches the existing HTML PUT handler, which edits one
// set at a time.
type UpdateExerciseEntryRequest struct {
	ExerciseID string     `json:"exercise_id" validate:"required"`
	Notes      string     `json:"notes"       validate:"max=500"`
	Reps       int        `json:"reps"        validate:"gte=1,lte=1000"`
	Weight     float64    `json:"weight"      validate:"gte=0,lte=5000"`
	RestTime   int        `json:"rest_time"   validate:"gte=0,lte=3600"`
	CreatedAt  *time.Time `json:"created_at,omitempty"`
}

// HistoryStatsDTO is the lifetime-stats header shown above
// the history list (personal best, last set).
type HistoryStatsDTO struct {
	MaxWeight float64           `json:"max_weight"`
	LastSet   *ExerciseEntryDTO `json:"last_set,omitempty"`
}

// HistoryStatsFromModel converts a models.HistoryStats into
// the DTO. The zero-value LastSet (when the user has no
// exercise entries for the exercise) is mapped to a nil
// pointer so the field is omitted from the JSON.
func HistoryStatsFromModel(s models.HistoryStats) HistoryStatsDTO {
	out := HistoryStatsDTO{MaxWeight: s.MaxWeight}
	if s.LastSet.ID != "" {
		dto := ExerciseEntryFromModel(s.LastSet)
		out.LastSet = &dto
	}
	return out
}

// HistoryPageDTO is the body of a paginated history
// response. The HasPrev/HasNext flags let the iOS pager
// render the same "Previous / Next" buttons as the web
// history view without a separate HEAD request.
type HistoryPageDTO struct {
	Entries []ExerciseEntryDTO `json:"entries"`
	Stats   HistoryStatsDTO    `json:"stats"`
	Page    int                `json:"page"`
	HasPrev bool               `json:"has_prev"`
	HasNext bool               `json:"has_next"`
}

// HistoryPageFromModel converts a models.ExerciseHistoryPage
// into the DTO. The empty-cases return empty (non-nil) slices
// so the JSON encoder writes `[]` rather than `null`.
func HistoryPageFromModel(p *models.ExerciseHistoryPage) HistoryPageDTO {
	return HistoryPageDTO{
		Entries: ExerciseEntriesFromModels(p.ExerciseEntries),
		Stats:   HistoryStatsFromModel(p.Stats),
		Page:    p.Page,
		HasPrev: p.HasPrev,
		HasNext: p.HasNext,
	}
}

// --- Goals ---

// GoalDTO is the JSON shape for a single goal in the
// /api/v1/goals namespace. Mirrors models.Goal but exposes
// only the fields the iOS client needs. StartDate, TargetDate,
// EndDate and CompletedAt are pointers so a missing date is
// rendered as JSON null (not zero-time, which would imply
// "1970-01-01" and break the UI's conditional rendering).
type GoalDTO struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	TargetDate  *time.Time `json:"target_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CreateGoalRequest is the body for POST /api/v1/goals. The
// validation limits match the HTML form (title 1-200,
// description 0-2000) so the JSON and HTML surfaces reject
// the same inputs.
type CreateGoalRequest struct {
	Title       string     `json:"title"       validate:"required,min=1,max=200"`
	Description string     `json:"description" validate:"max=2000"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	TargetDate  *time.Time `json:"target_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
}

// UpdateGoalRequest is the body for PUT /api/v1/goals/:id.
// CompletedAt is intentionally NOT editable here — clients
// must use POST /goals/:id/complete or /reopen so the server
// owns the completion timestamp (matches the HTML surface).
type UpdateGoalRequest struct {
	Title       string     `json:"title"       validate:"required,min=1,max=200"`
	Description string     `json:"description" validate:"max=2000"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	TargetDate  *time.Time `json:"target_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
}

// GoalFromModel converts a models.Goal into its DTO. Pass by
// value so callers don't have to dereference.
func GoalFromModel(g models.Goal) GoalDTO {
	return GoalDTO{
		ID:          g.ID,
		Title:       g.Title,
		Description: g.Description,
		StartDate:   g.StartDate,
		TargetDate:  g.TargetDate,
		EndDate:     g.EndDate,
		CompletedAt: g.CompletedAt,
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   g.UpdatedAt,
	}
}

// GoalsFromModels converts a slice of goals into the DTO
// slice. Returns a non-nil empty slice so the JSON encoder
// writes `[]` rather than `null` when the user has no goals.
func GoalsFromModels(gs []models.Goal) []GoalDTO {
	out := make([]GoalDTO, 0, len(gs))
	for _, g := range gs {
		out = append(out, GoalFromModel(g))
	}
	return out
}

// --- Feedback ---

// SubmitFeedbackRequest is the body for POST /api/v1/feedback.
// Mirrors the two fields the web app's /feedback form collects
// (`title`, `message`); validation is delegated to
// FeedbackController.Submit so the JSON and HTML surfaces
// enforce the exact same length rules (title 5–100,
// message 10–1000, both trimmed). The user_id is taken from
// the JWT — clients cannot submit feedback on someone else's
// behalf.
type SubmitFeedbackRequest struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

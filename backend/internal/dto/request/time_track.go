package request

import "time"

// TimeTrackStartRequest starts a timer.
type TimeTrackStartRequest struct {
	Description *string `json:"description"`
}

// TimeTrackStopRequest stops a timer.
type TimeTrackStopRequest struct {
	Description *string `json:"description"`
}

// TimeTrackCreateRequest manually logs a completed time entry (duration in seconds).
type TimeTrackCreateRequest struct {
	Duration    int64      `json:"duration" binding:"required,min=1"`
	Description *string    `json:"description"`
	EndedAt     *time.Time `json:"ended_at"`
}

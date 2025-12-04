package main

// MessageState holds the current message and how many times it's been set.
type MessageState struct {
	// Message is the current message (5-10 characters).
	Message string `json:"message"`
	// SetCount is the number of times the message has been set.
	SetCount int32 `json:"set_count"`
}

// SetMessageParams contains the new message value.
type SetMessageParams struct {
	Message string `json:"message" validate:"required,min=5,max=10"`
}

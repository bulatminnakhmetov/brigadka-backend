package errors

// Messaging service error constants
const (
	ErrorUserNotInChat               = "user not in chat"
	ErrorInvalidReactionCode         = "invalid reaction code"
	ErrorNotAuthorizedToReact        = "user not authorized to react to this message"
	ErrorCannotCreateChatWithSelf    = "cannot create direct chat with yourself"
	ErrorChatAlreadyExistsWithThisID = "chat already exists with this ID"
	ErrorMessageAlreadyExists        = "message with this ID already exists"
	ErrorReactionAlreadyExists       = "reaction already exists with this ID"
)

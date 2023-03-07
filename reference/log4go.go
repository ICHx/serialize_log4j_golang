/****** LogRecord ******/

// A LogRecord contains all of the pertinent information for each message
type LogRecord struct {
	Level    Level     // The log level
	Created  time.Time // The time at which the log message was created (nanoseconds)
	Source   string    // The message source
	Message  string    // The log message
	Category string    // The log group
}

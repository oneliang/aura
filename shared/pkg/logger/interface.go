package logger

// Log is the public interface for external implementations.
// External applications can implement this interface to inject custom logging
// (e.g., fmt.Print, logrus, zap) without depending on aura's internal zerolog.
type Log interface {
	Debug(msg string, keyValues ...any)
	Info(msg string, keyValues ...any)
	Warn(msg string, keyValues ...any)
	Error(msg string, keyValues ...any)
}
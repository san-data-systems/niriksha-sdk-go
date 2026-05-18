package nirikshaai

import "context"

// WithFlush wraps fn, calling Flush after it returns regardless of error.
// Use in AWS Lambda handlers, Cloud Functions, or any short-lived process where
// the OTel SDK may not have time to export pending data before the process exits.
//
// Example:
//
//	err := nirikshaai.WithFlush(ctx, func(ctx context.Context) error {
//	    // your handler logic
//	    return nil
//	})
//
// If both fn and Flush return errors, fn's error is returned and the Flush
// error is silently discarded so that the original handler failure is not
// masked.
func WithFlush(ctx context.Context, fn func(context.Context) error) error {
	fnErr := fn(ctx)
	flushErr := Flush(ctx)
	if fnErr != nil {
		return fnErr
	}
	return flushErr
}

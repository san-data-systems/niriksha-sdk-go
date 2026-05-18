package nirikshaai

import (
	"context"

	"go.opentelemetry.io/otel/baggage"
)

// SetBaggage adds a key-value pair to the OTel Baggage carried in ctx.
// Returns a new context containing the updated Baggage; the original context
// is unchanged. Silently returns the original context if the key or value is
// malformed per the W3C Baggage specification.
func SetBaggage(ctx context.Context, key, value string) context.Context {
	member, err := baggage.NewMember(key, value)
	if err != nil {
		return ctx
	}

	bag, err := baggage.FromContext(ctx).SetMember(member)
	if err != nil {
		return ctx
	}

	return baggage.ContextWithBaggage(ctx, bag)
}

// GetBaggage retrieves a value from OTel Baggage in ctx.
// Returns an empty string if the key is absent or the Baggage is not set.
func GetBaggage(ctx context.Context, key string) string {
	return baggage.FromContext(ctx).Member(key).Value()
}

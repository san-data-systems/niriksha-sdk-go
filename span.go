package nirikshaai

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// RecordConversation records multi-turn conversation metadata on the current span.
// Call at the start of each LLM turn to correlate spans belonging to the same
// conversation and session.
func RecordConversation(ctx context.Context, conversationID, sessionID string, turnIndex int) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("llm.conversation.id", conversationID),
		attribute.String("llm.session.id", sessionID),
		attribute.Int("llm.turn.index", turnIndex),
	)
}

// RAGChunk holds metadata for a single retrieved document chunk.
type RAGChunk struct {
	ChunkID string
	Source  string
	Score   float64
	Content string // optional; empty string skips the attribute
}

// RecordRAGChunk attaches retrieved chunk metadata to the current span as a span
// event. Call once per chunk returned by your retrieval step.
func RecordRAGChunk(ctx context.Context, chunk RAGChunk) {
	attrs := []attribute.KeyValue{
		attribute.String("rag.chunk.id", chunk.ChunkID),
		attribute.String("rag.source", chunk.Source),
		attribute.Float64("rag.chunk.score", chunk.Score),
	}
	if chunk.Content != "" {
		attrs = append(attrs, attribute.String("rag.chunk.content", chunk.Content))
	}
	trace.SpanFromContext(ctx).AddEvent("rag.chunk.retrieved", trace.WithAttributes(attrs...))
}

// ToolCall holds metadata for an LLM tool/function invocation.
type ToolCall struct {
	ToolName string
	CallID   string
	Input    string // JSON string; optional
	Output   string // JSON string; optional
}

// RecordToolCall attaches tool call metadata to the current span as a span event.
// Call once per tool invocation made by the LLM during a generation step.
func RecordToolCall(ctx context.Context, call ToolCall) {
	attrs := []attribute.KeyValue{
		attribute.String("llm.tool.name", call.ToolName),
		attribute.String("llm.tool.call_id", call.CallID),
	}
	if call.Input != "" {
		attrs = append(attrs, attribute.String("llm.tool.input", call.Input))
	}
	if call.Output != "" {
		attrs = append(attrs, attribute.String("llm.tool.output", call.Output))
	}
	trace.SpanFromContext(ctx).AddEvent("llm.tool.call", trace.WithAttributes(attrs...))
}

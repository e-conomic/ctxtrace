package ctxtrace

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/openzipkin/zipkin-go/propagation/b3"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
)

const (
	dummyRequestID = "Foo"
)

func TestPackMetadata(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	ctx = context.WithValue(ctx, traceCtxMarker{}, TraceData{
		RequestID: dummyRequestID,
	})

	ctx = NewOutgoingContextWithData(ctx)
	outGoingMD, _ := metadata.FromOutgoingContext(ctx)
	if assert.NotNil(t, outGoingMD) {
		if assert.Contains(t, outGoingMD, headerRequestID) {
			if assert.Len(t, outGoingMD[headerRequestID], 1) {
				assert.Equal(t, dummyRequestID, outGoingMD[headerRequestID][0])
			}
		}
	}
}

func TestAddOtelSpanContextToContext_NotSmapled(t *testing.T) {
	r := httptest.NewRequest("GET", "/foo", nil)
	r.Header.Set(headerRequestID, dummyRequestID)
	r.Header.Set(b3.ParentSpanID, "0716f381a10c2a9b")
	r.Header.Set(b3.Sampled, "0")
	r.Header.Set(b3.SpanID, "b2f181687dd7ca60")
	r.Header.Set(b3.TraceID, "df1af326541277a75e451e1c03b7e893")

	ctx := context.Background()
	data, err := ExtractHTTP(r)
	assert.Nil(t, err)

	ctx = addOtelSpanContextToContext(ctx, data)

	spanContext := trace.SpanContextFromContext(ctx)

	assert.NotNil(t, spanContext)
	assert.Equal(t, spanContext.TraceFlags(), trace.TraceFlags(0))
	assert.Equal(t, spanContext.SpanID().String(), data.TraceSpan.ID.String())
	assert.Equal(t, spanContext.TraceID().String(), data.TraceSpan.TraceID.String())
}

func TestAddOtelSpanContextToContext_Smapled(t *testing.T) {
	r := httptest.NewRequest("GET", "/foo", nil)
	r.Header.Set(headerRequestID, dummyRequestID)
	r.Header.Set(b3.ParentSpanID, "0716f381a10c2a9b")
	r.Header.Set(b3.Sampled, "1")
	r.Header.Set(b3.SpanID, "b2f181687dd7ca60")
	r.Header.Set(b3.TraceID, "df1af326541277a75e451e1c03b7e893")

	ctx := context.Background()
	data, err := ExtractHTTP(r)
	assert.Nil(t, err)

	ctx = addOtelSpanContextToContext(ctx, data)

	spanContext := trace.SpanContextFromContext(ctx)

	assert.NotNil(t, spanContext)
	assert.Equal(t, spanContext.TraceFlags(), trace.TraceFlags(1))
	assert.Equal(t, spanContext.SpanID().String(), data.TraceSpan.ID.String())
	assert.Equal(t, spanContext.TraceID().String(), data.TraceSpan.TraceID.String())
}

func TestAddOtelSpanContextToContext_InvalidParent(t *testing.T) {
	r := httptest.NewRequest("GET", "/foo", nil)
	r.Header.Set(headerRequestID, dummyRequestID)
	r.Header.Set(b3.ParentSpanID, "")
	r.Header.Set(b3.Sampled, "1")
	r.Header.Set(b3.SpanID, "")
	r.Header.Set(b3.TraceID, "")

	ctx := context.Background()
	data, err := ExtractHTTP(r)
	assert.Nil(t, err)

	ctx = addOtelSpanContextToContext(ctx, data)

	// If there's no spanContext in the ctx, default EmptySpanContext is returned
	spanContext := trace.SpanContextFromContext(ctx)

	assert.NotNil(t, spanContext)
	assert.False(t, spanContext.IsValid())
	assert.Equal(t, spanContext, trace.SpanContext{})
}

func TestExtractHTTP(t *testing.T) {
	r := httptest.NewRequest("GET", "/foo", nil)
	r.Header.Set(headerRequestID, dummyRequestID)
	r.Header.Set(b3.ParentSpanID, "0716f381a10c2a9b")
	r.Header.Set(b3.Sampled, "0")
	r.Header.Set(b3.SpanID, "b2f181687dd7ca60")
	r.Header.Set(b3.TraceID, "df1af326541277a75e451e1c03b7e893")

	data, err := ExtractHTTP(r)
	assert.Nil(t, err)
	assert.NotNil(t, data.TraceSpan)
	assert.Equal(t, dummyRequestID, data.RequestID)

	ctx := context.Background()
	ctx = ExtractHTTPToContext(ctx, r)

	ctx = NewOutgoingContextWithData(ctx)
	outGoingMD, _ := metadata.FromOutgoingContext(ctx)
	if assert.NotNil(t, outGoingMD) {
		if assert.Contains(t, outGoingMD, headerRequestID) {
			if assert.Len(t, outGoingMD[headerRequestID], 1) {
				assert.Equal(t, dummyRequestID, outGoingMD[headerRequestID][0])
			}
		}
		assert.Contains(t, outGoingMD, b3.SpanID)
		assert.Contains(t, outGoingMD, b3.Sampled)
		assert.Contains(t, outGoingMD, b3.TraceID)
		assert.Contains(t, outGoingMD, b3.ParentSpanID)
	}
}

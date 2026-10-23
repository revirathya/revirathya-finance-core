package tracing

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/revirathya/revirathya-finance-core/internal/delivery/http/graphql/tracing"

// Extension traces GraphQL operations and resolvers. Unlike otelgqlgen it records neither the query text nor
// argument values, so inline literals (an idToken, a description) never reach the spans.
type Extension struct {
	tracer trace.Tracer
}

var (
	_ graphql.HandlerExtension    = Extension{}
	_ graphql.ResponseInterceptor = Extension{}
	_ graphql.FieldInterceptor    = Extension{}
)

func New() Extension {
	return Extension{
		tracer: otel.Tracer(tracerName),
	}
}

func (Extension) ExtensionName() string {
	return "Tracing"
}

func (Extension) Validate(graphql.ExecutableSchema) error {
	return nil
}

// InterceptResponse starts one span per operation, named like "mutation createExpense".
func (e Extension) InterceptResponse(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
	if !graphql.HasOperationContext(ctx) {
		return next(ctx)
	}

	oc := graphql.GetOperationContext(ctx)
	opType := "operation"
	if oc.Operation != nil {
		opType = string(oc.Operation.Operation)
	}

	opName := oc.OperationName
	if opName == "" && oc.Operation != nil {
		opName = oc.Operation.Name
	}

	name := opType
	if opName != "" {
		name += " " + opName
	}

	ctx, span := e.tracer.Start(
		ctx,
		name,
		trace.WithAttributes(
			attribute.String("graphql.operation.type", opType),
			attribute.String("graphql.operation.name", opName),
		),
	)
	defer span.End()

	response := next(ctx)
	if response != nil && len(response.Errors) > 0 {
		span.SetStatus(codes.Error, "") // no message: error texts may echo input
	}
	return response
}

// InterceptField start a span per resolver (not per plain field), named like "Mutation.createExpense".
func (e Extension) InterceptField(ctx context.Context, next graphql.Resolver) (any, error) {
	fc := graphql.GetFieldContext(ctx)
	if fc == nil || !fc.IsResolver {
		return next(ctx)
	}

	ctx, span := e.tracer.Start(ctx, fc.Object+"."+fc.Field.Name)
	defer span.End()

	res, err := next(ctx)
	if err != nil {
		span.SetStatus(codes.Error, "")
	}
	return res, err
}

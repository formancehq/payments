package v3

import (
	"fmt"
	"io"
	"net/http"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/query"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type TestData struct {
	Val string `json:"val"`
}

func oversizeRequestBody() []TestData {
	var data []TestData
	for i := 0; i < 1000000; i++ {
		data = append(data, TestData{
			Val: fmt.Sprintf("Rindfleischetikettierungsüberwachungsaufgabenübertragungsgesetz %d", i),
		})
	}
	return data
}

func getQueryBuilder(span trace.Span, r *http.Request) (query.Builder, error) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	if len(data) > 0 {
		span.SetAttributes(attribute.String("query", string(data)))
		return query.ParseJSON(string(data))
	} else {
		// In order to be backward compatible
		span.SetAttributes(attribute.String("query", r.URL.Query().Get("query")))
		return query.ParseJSON(r.URL.Query().Get("query"))
	}
}

func getPagination[T any](span trace.Span, r *http.Request, options T) (*bunpaginate.PaginatedQueryOptions[T], error) {
	return getPaginationWithBuilder[T](span, r, nil, options)
}

func getPaginationWithBuilder[T any](span trace.Span, r *http.Request, appendQuery query.Builder, options T) (*bunpaginate.PaginatedQueryOptions[T], error) {
	qb, err := getQueryBuilder(span, r)
	if err != nil {
		return nil, err
	}

	pageSize, err := bunpaginate.GetPageSize(r)
	if err != nil {
		return nil, err
	}
	span.SetAttributes(attribute.Int64("page_size", int64(pageSize)))

	queryBuilders := make([]query.Builder, 0, 2)
	if qb != nil {
		queryBuilders = append(queryBuilders, qb)
	}
	if appendQuery != nil {
		queryBuilders = append(queryBuilders, appendQuery)
	}

	if len(queryBuilders) == 0 {
		return pointer.For(bunpaginate.NewPaginatedQueryOptions(options).WithPageSize(pageSize)), nil
	}

	builder := query.And(queryBuilders...)
	return pointer.For(bunpaginate.NewPaginatedQueryOptions(options).WithQueryBuilder(builder).WithPageSize(pageSize)), nil
}

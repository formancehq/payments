package v3

import (
	"fmt"
	"io"
	"net/http"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/go-libs/v2/query"
	"github.com/formancehq/payments/internal/api/validation"
	"github.com/go-playground/validator/v10"
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

func WrapValidationError(w http.ResponseWriter, code string, rawErr error) {
	if errs, ok := rawErr.(validator.ValidationErrors); ok && len(errs) > 0 {
		err := fmt.Errorf("%s", errs[0].Translate(validation.Translator()))
		api.BadRequest(w, code, err)
		return
	}
	// fallback
	api.BadRequest(w, code, rawErr)
}

func getPagination[T any](span trace.Span, r *http.Request, options T) (*bunpaginate.PaginatedQueryOptions[T], error) {
	qb, err := getQueryBuilder(span, r)
	if err != nil {
		return nil, err
	}

	pageSize, err := bunpaginate.GetPageSize(r)
	if err != nil {
		return nil, err
	}
	span.SetAttributes(attribute.Int64("page_size", int64(pageSize)))

	return pointer.For(bunpaginate.NewPaginatedQueryOptions(options).WithQueryBuilder(qb).WithPageSize(pageSize)), nil
}

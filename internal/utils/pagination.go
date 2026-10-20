package utils

import (
	"github.com/revirathya/revirathya-finance-core/internal/domain/constant"
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
)

func SetPageSize(pageSize int) (int, error) {
	ps := pageSize
	switch {
	case ps < 0:
		return 0, &errDom.ValidationError{Field: "page_size", Reason: "must not be negative"}
	case ps == 0:
		ps = constant.DefaultPageSize
	case pageSize > constant.MaxPageSize:
		ps = constant.MaxPageSize
	}

	return ps, nil
}

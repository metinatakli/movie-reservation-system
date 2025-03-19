package domain

import "strings"

type Pagination struct {
	Page     int
	PageSize int
	Term     string
	Sort     string
}

func (f Pagination) SortColumn() string {
	return strings.TrimPrefix(f.Sort, "-")
}

func (f Pagination) SortDirection() string {
	if strings.HasPrefix(f.Sort, "-") {
		return "DESC"
	}

	return "ASC"
}

func (f Pagination) Limit() int {
	return f.PageSize
}

func (f Pagination) Offset() int {
	return (f.Page - 1) * f.PageSize
}

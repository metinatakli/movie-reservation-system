package domain

type Metadata struct {
	CurrentPage  int
	FirstPage    int
	LastPage     int
	PageSize     int
	TotalRecords int
}

func NewMetadata(totalRecords, page, pageSize int) *Metadata {
	return &Metadata{
		CurrentPage:  page,
		FirstPage:    1,
		LastPage:     (totalRecords + pageSize - 1) / pageSize,
		PageSize:     pageSize,
		TotalRecords: totalRecords,
	}
}

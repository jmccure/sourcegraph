package partitions

import (
	"fmt"
	"time"
)

type rangePartitionStrategy[T RangeBound] struct{}

func NewRangePartitionStrategy[T RangeBound]() PartitionStrategy[RangePartitionKey[T]] {
	return &rangePartitionStrategy[T]{}
}

func (m *rangePartitionStrategy[T]) FormatValuesClause(partitionKey RangePartitionKey[T]) string {
	return fmt.Sprintf(`FROM (%s) TO (%s)`, partitionKey.LowerBound.String(), partitionKey.UpperBound.String())
}

type RangeBound = interface {
	fmt.Stringer
	time.Time | string | int
}

type RangePartitionKey[T RangeBound] struct {
	LowerBound T
	UpperBound T
}

func (k RangePartitionKey[T]) Name() string {
	// TODO - sanitize
	return fmt.Sprintf("%s_%s", k.LowerBound, k.UpperBound)
}

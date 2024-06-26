package analyzer

import (
	"context"
	"errors"
	"fmt"
	"github.com/pirsch-analytics/pirsch/v6/pkg/db"
	"github.com/pirsch-analytics/pirsch/v6/pkg/model"
	"strings"
)

// Funnel aggregates funnels.
type Funnel struct {
	analyzer *Analyzer
	store    db.Store
}

// Steps returns the funnel steps for given filter list.
func (funnel *Funnel) Steps(ctx context.Context, filter []Filter) ([]model.FunnelStep, error) {
	if len(filter) < 2 {
		return nil, errors.New("not enough steps")
	}

	var query strings.Builder
	args := make([]any, 0)

	for i := range filter {
		f := funnel.analyzer.getFilter(&filter[i])
		f.funnelStep = i
		fields := []Field{
			FieldClientID,
			FieldVisitorID,
			FieldSessionID,
			FieldTime,
		}
		q, a := f.buildQuery(fields, nil, nil)
		args = append(args, a...)

		if i == 0 {
			query.WriteString(fmt.Sprintf("WITH step%d AS ( ", i))
		} else {
			query.WriteString(fmt.Sprintf("step%d AS ( ", i))
		}

		query.WriteString(q)
		query.WriteString(") ")

		if i != len(filter)-1 {
			query.WriteString(", ")
		}
	}

	query.WriteString("SELECT * FROM ( ")

	for i := 0; i < len(filter); i++ {
		query.WriteString(fmt.Sprintf("SELECT %d step, uniq(visitor_id) FROM step%d ", i, i))

		if i != len(filter)-1 {
			query.WriteString("UNION ALL ")
		}
	}

	query.WriteString(") ORDER BY step")
	stats, err := funnel.store.SelectFunnelSteps(ctx, query.String(), args...)

	if err != nil {
		return nil, err
	}

	return stats, nil
}

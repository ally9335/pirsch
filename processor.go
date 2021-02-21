package pirsch

import (
	"database/sql"
	"github.com/jmoiron/sqlx"
	"time"
)

type processFunc func(*sqlx.Tx, sql.NullInt64, time.Time) error
type processPathFunc func(*sqlx.Tx, sql.NullInt64, time.Time, string) error

// Processor processes hits to reduce them into meaningful statistics.
type Processor struct {
	store Store
}

// NewProcessor creates a new Processor for given Store.
func NewProcessor(store Store) *Processor {
	return &Processor{
		store: store,
	}
}

// Process processes all hits in database and deletes them afterwards.
// This does only apply the tenant ID is null, else ProcessTenant needs to be called for each tenant.
func (processor *Processor) Process() error {
	return processor.ProcessTenant(NullTenant)
}

// ProcessTenant processes all hits in database for given tenant and deletes them afterwards.
// The tenant can be set to nil if you don't split your data (which is usually the case).
func (processor *Processor) ProcessTenant(tenantID sql.NullInt64) error {
	// this explicitly excludes "today", because we might not have collected all visitors
	// and the hits will be deleted after the processor has finished reducing the data
	days, err := processor.store.HitDays(tenantID)

	if err != nil {
		return err
	}

	for _, day := range days {
		if err := processor.processDay(tenantID, day); err != nil {
			return err
		}
	}

	return nil
}

func (processor *Processor) processDay(tenantID sql.NullInt64, day time.Time) error {
	paths, err := processor.store.HitPaths(tenantID, day)

	if err != nil {
		return err
	}

	tx := processor.store.NewTx()

	for _, path := range paths {
		if err := processor.processPath(tx, tenantID, day, path); err != nil {
			processor.store.Rollback(tx)
			return err
		}
	}

	processFuncs := []processFunc{
		processor.visitorHours,
		processor.visitors,
		processor.languages,
		processor.referrer,
		processor.os,
		processor.browser,
		processor.screen,
		processor.country,
		processor.store.DeleteHitsByDay,
	}

	for _, f := range processFuncs {
		if err := f(tx, tenantID, day); err != nil {
			processor.store.Rollback(tx)
			return err
		}
	}

	processor.store.Commit(tx)
	return nil
}

func (processor *Processor) processPath(tx *sqlx.Tx, tenantID sql.NullInt64, day time.Time, path string) error {
	processFuncs := []processPathFunc{
		processor.pathVisitors,
		processor.pathLanguages,
		processor.pathReferrer,
		processor.pathOS,
		processor.pathBrowser,
	}

	for _, f := range processFuncs {
		if err := f(tx, tenantID, day, path); err != nil {
			return err
		}
	}

	return nil
}

func (processor *Processor) pathVisitors(tx *sqlx.Tx, tenantID sql.NullInt64, day time.Time, path string) error {
	visitors, err := processor.store.CountVisitorsByPath(tx, tenantID, day, path, true)

	if err != nil {
		return err
	}

	bounces := processor.store.CountVisitorsByPathAndMaxOneHit(tx, tenantID, day, path)

	for _, v := range visitors {
		v.Bounces = bounces

		if err := processor.store.SaveVisitorStats(tx, &v); err != nil {
			return err
		}
	}

	return nil
}

func (processor *Processor) pathLanguages(tx *sqlx.Tx, tenantID sql.NullInt64, day time.Time, path string) error {
	visitors, err := processor.store.CountVisitorsByPathAndLanguage(tx, tenantID, day, path)

	if err != nil {
		return err
	}

	for _, v := range visitors {
		if err := processor.store.SaveLanguageStats(tx, &v); err != nil {
			return err
		}
	}

	return nil
}

func (processor *Processor) pathReferrer(tx *sqlx.Tx, tenantID sql.NullInt64, day time.Time, path string) error {
	visitors, err := processor.store.CountVisitorsByPathAndReferrer(tx, tenantID, day, path)

	if err != nil {
		return err
	}

	for _, v := range visitors {
		v.Bounces = processor.store.CountVisitorsByPathAndReferrerAndMaxOneHit(tx, tenantID, day, path, v.Referrer.String)

		if err := processor.store.SaveReferrerStats(tx, &v); err != nil {
			return err
		}
	}

	return nil
}

func (processor *Processor) pathOS(tx *sqlx.Tx, tenantID sql.NullInt64, day time.Time, path string) error {
	visitors, err := processor.store.CountVisitorsByPathAndOS(tx, tenantID, day, path)

	if err != nil {
		return err
	}

	for _, v := range visitors {
		if err := processor.store.SaveOSStats(tx, &v); err != nil {
			return err
		}
	}

	return nil
}

func (processor *Processor) pathBrowser(tx *sqlx.Tx, tenantID sql.NullInt64, day time.Time, path string) error {
	visitors, err := processor.store.CountVisitorsByPathAndBrowser(tx, tenantID, day, path)

	if err != nil {
		return err
	}

	for _, v := range visitors {
		if err := processor.store.SaveBrowserStats(tx, &v); err != nil {
			return err
		}
	}

	return nil
}

func (processor *Processor) visitorHours(tx *sqlx.Tx, tenantID sql.NullInt64, day time.Time) error {
	visitors, err := processor.store.CountVisitorsByHour(tx, tenantID, day)

	if err != nil {
		return err
	}

	for _, v := range visitors {
		if err := processor.store.SaveVisitorTimeStats(tx, &v); err != nil {
			return err
		}
	}

	return nil
}

func (processor *Processor) visitors(tx *sqlx.Tx, tenantID sql.NullInt64, day time.Time) error {
	visitors := processor.store.CountVisitors(tx, tenantID, day)
	visitors.TenantID = tenantID
	visitors.Bounces = processor.store.CountVisitorsByPathAndMaxOneHit(tx, tenantID, day, "")
	visitors.AverageSessionDurationSeconds = processor.store.SessionDurationSum(tx, tenantID, day)
	platforms := processor.store.CountVisitorsByPlatform(tx, tenantID, day)
	platformSum := float64(platforms.PlatformDesktop + platforms.PlatformMobile + platforms.PlatformUnknown)
	v := &VisitorStats{
		Stats:                   *visitors,
		PlatformDesktop:         platforms.PlatformDesktop,
		PlatformMobile:          platforms.PlatformMobile,
		PlatformUnknown:         platforms.PlatformUnknown,
		RelativePlatformDesktop: float64(platforms.PlatformDesktop) / platformSum,
		RelativePlatformMobile:  float64(platforms.PlatformMobile) / platformSum,
		RelativePlatformUnknown: float64(platforms.PlatformUnknown) / platformSum,
	}

	if err := processor.store.SaveVisitorStats(tx, v); err != nil {
		return err
	}

	return nil
}

func (processor *Processor) languages(tx *sqlx.Tx, tenantID sql.NullInt64, day time.Time) error {
	visitors, err := processor.store.CountVisitorsByLanguage(tx, tenantID, day)

	if err != nil {
		return err
	}

	for _, v := range visitors {
		v.TenantID = tenantID

		if err := processor.store.SaveLanguageStats(tx, &v); err != nil {
			return err
		}
	}

	return nil
}

func (processor *Processor) referrer(tx *sqlx.Tx, tenantID sql.NullInt64, day time.Time) error {
	visitors, err := processor.store.CountVisitorsByReferrer(tx, tenantID, day)

	if err != nil {
		return err
	}

	for _, v := range visitors {
		v.TenantID = tenantID
		v.Bounces = processor.store.CountVisitorsByPathAndReferrerAndMaxOneHit(tx, tenantID, day, "", v.Referrer.String)

		if err := processor.store.SaveReferrerStats(tx, &v); err != nil {
			return err
		}
	}

	return nil
}

func (processor *Processor) os(tx *sqlx.Tx, tenantID sql.NullInt64, day time.Time) error {
	visitors, err := processor.store.CountVisitorsByOS(tx, tenantID, day)

	if err != nil {
		return err
	}

	for _, v := range visitors {
		v.TenantID = tenantID

		if err := processor.store.SaveOSStats(tx, &v); err != nil {
			return err
		}
	}

	return nil
}

func (processor *Processor) browser(tx *sqlx.Tx, tenantID sql.NullInt64, day time.Time) error {
	visitors, err := processor.store.CountVisitorsByBrowser(tx, tenantID, day)

	if err != nil {
		return err
	}

	for _, v := range visitors {
		v.TenantID = tenantID

		if err := processor.store.SaveBrowserStats(tx, &v); err != nil {
			return err
		}
	}

	return nil
}

func (processor *Processor) screen(tx *sqlx.Tx, tenantID sql.NullInt64, day time.Time) error {
	visitors, err := processor.store.CountVisitorsByScreenSize(tx, tenantID, day)

	if err != nil {
		return err
	}

	for _, v := range visitors {
		if err := processor.store.SaveScreenStats(tx, &v); err != nil {
			return err
		}
	}

	visitors, err = processor.store.CountVisitorsByScreenClass(tx, tenantID, day)

	if err != nil {
		return err
	}

	for _, v := range visitors {
		if err := processor.store.SaveScreenStats(tx, &v); err != nil {
			return err
		}
	}

	return nil
}

func (processor *Processor) country(tx *sqlx.Tx, tenantID sql.NullInt64, day time.Time) error {
	visitors, err := processor.store.CountVisitorsByCountryCode(tx, tenantID, day)

	if err != nil {
		return err
	}

	for _, v := range visitors {
		if err := processor.store.SaveCountryStats(tx, &v); err != nil {
			return err
		}
	}

	return nil
}

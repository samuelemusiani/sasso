package db

import (
	"errors"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gorm.io/gorm"
)

const (
	PromErrorTypeConnectionRefused = "connection_refused"
	PromErrorTypeConnectionError   = "connection_error"
	PromErrorTypeTimeout           = "timeout"
	PromErrorTypeDuplicateKey      = "duplicate_key"
	PromErrorTypeOther             = "other"
)

var (
	promAllErrorTypes = []string{
		PromErrorTypeConnectionRefused,
		PromErrorTypeConnectionError,
		PromErrorTypeTimeout,
		PromErrorTypeDuplicateKey,
		PromErrorTypeOther,
	}

	gormErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gorm_errors_total",
			Help: "Total number of GORM errors",
		},
		[]string{"error_type"},
	)
)

// ErrorMetricsPlugin is a GORM plugin that collects metrics on database errors
// and classifies them into categories for Prometheus.
type ErrorMetricsPlugin struct{}

func (*ErrorMetricsPlugin) Name() string {
	return "gorm:error_metrics"
}

func (p *ErrorMetricsPlugin) Initialize(db *gorm.DB) error {
	// Initialize metrics with 0 in order to have them available in Prometheus
	// even if no errors have occurred yet.
	for _, errType := range promAllErrorTypes {
		gormErrors.WithLabelValues(errType).Add(0)
	}

	err := db.Callback().Create().After("gorm:create").Register("metrics:after_create", p.after)
	if err != nil {
		return err
	}

	err = db.Callback().Query().After("gorm:query").Register("metrics:after_query", p.after)
	if err != nil {
		return err
	}

	err = db.Callback().Update().After("gorm:update").Register("metrics:after_update", p.after)
	if err != nil {
		return err
	}

	err = db.Callback().Delete().After("gorm:delete").Register("metrics:after_delete", p.after)
	if err != nil {
		return err
	}

	return nil
}

func (*ErrorMetricsPlugin) after(db *gorm.DB) {
	if db.Error != nil && !errors.Is(db.Error, gorm.ErrRecordNotFound) {
		var errorType string

		errStr := db.Error.Error()
		switch {
		case strings.Contains(errStr, "connection refused"):
			errorType = PromErrorTypeConnectionRefused
		case strings.Contains(errStr, "dial error"):
			errorType = PromErrorTypeConnectionError
		case strings.Contains(errStr, "timeout"):
			errorType = PromErrorTypeTimeout
		case strings.Contains(errStr, "duplicate key"):
			errorType = PromErrorTypeDuplicateKey
		default:
			errorType = PromErrorTypeOther
		}

		gormErrors.WithLabelValues(errorType).Inc()
	}
}

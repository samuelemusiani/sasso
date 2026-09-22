package notify

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"samuelemusiani/sasso/server/db"
)

var (
	// A counter would do the job, but as we set the number of notifications
	// periodically in the worker, a gauge is more suitable.
	notificationsTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sasso_notifications",
			Help: "Total number of notifications sent.",
		}, []string{"status", "email", "telegram"},
	)
)

func notificationsCounter() {
	for _, status := range []string{"pending", "sent"} {
		for _, email := range []string{"true", "false"} {
			for _, telegram := range []string{"true", "false"} {
				var count int64

				count, err := db.CountNotifications(status, email == "true", telegram == "true")
				if err != nil {
					logger.Error("Failed to count notifications for prometheus", "status", status, "email", email, "telegram", telegram, "error", err)

					continue
				}

				notificationsTotal.WithLabelValues(status, email, telegram).Set(float64(count))
			}
		}
	}
}

package adam

import (
	"fmt"
	"path"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

//getLogsRedisStream return logs stream for devUUID for load from redis
func (adam *Ctx) getLogsRedisStream(devUUID uuid.UUID) (dir string) {
	return fmt.Sprintf("%s%s", defaults.DefaultLogsRedisPrefix, devUUID.String())
}

//getInfoRedisStream return info stream for devUUID for load from redis
func (adam *Ctx) getInfoRedisStream(devUUID uuid.UUID) (dir string) {
	return fmt.Sprintf("%s%s", defaults.DefaultInfoRedisPrefix, devUUID.String())
}

//getMetricsRedisStream return metrics stream for devUUID for load from redis
func (adam *Ctx) getMetricsRedisStream(devUUID uuid.UUID) (dir string) {
	return fmt.Sprintf("%s%s", defaults.DefaultMetricsRedisPrefix, devUUID.String())
}

//getRequestRedisStream return request stream for devUUID for requests from redis
func (adam *Ctx) getRequestRedisStream(devUUID uuid.UUID) (dir string) {
	return fmt.Sprintf("%s%s", defaults.DefaultRequestsRedisPrefix, devUUID.String())
}

//getLogsRedisStreamCache return logs stream for devUUID for caching in redis
func (adam *Ctx) getLogsRedisStreamCache(devUUID uuid.UUID) (dir string) {
	if adam.AdamCachingPrefix == "" {
		return adam.getLogsRedisStream(devUUID)
	}
	return fmt.Sprintf("LOGS_EVE_%s_%s", adam.AdamCachingPrefix, devUUID.String())
}

//getInfoRedisStreamCache return info stream for devUUID for caching in redis
func (adam *Ctx) getInfoRedisStreamCache(devUUID uuid.UUID) (dir string) {
	if adam.AdamCachingPrefix == "" {
		return adam.getInfoRedisStream(devUUID)
	}
	return fmt.Sprintf("INFO_EVE_%s_%s", adam.AdamCachingPrefix, devUUID.String())
}

//getMetricsRedisStreamCache return metrics stream for devUUID for caching in redis
func (adam *Ctx) getMetricsRedisStreamCache(devUUID uuid.UUID) (dir string) {
	if adam.AdamCachingPrefix == "" {
		return adam.getMetricsRedisStream(devUUID)
	}
	return fmt.Sprintf("METRICS_EVE_%s_%s", adam.AdamCachingPrefix, devUUID.String())
}

//getRequestRedisStreamCache return request stream for devUUID for caching in redis
func (adam *Ctx) getRequestRedisStreamCache(devUUID uuid.UUID) (dir string) {
	if adam.AdamCachingPrefix == "" {
		return adam.getRequestRedisStream(devUUID)
	}
	return fmt.Sprintf("%s%s_%s", defaults.DefaultRequestsRedisPrefix, adam.AdamCachingPrefix, devUUID.String())
}

//getRedisStreamCache return logs stream for devUUID for caching in redis
func (adam *Ctx) getLogsDirCache(devUUID uuid.UUID) (dir string) {
	if adam.AdamCachingPrefix == "" {
		return adam.getLogsDir(devUUID)
	}
	return path.Join(adam.dir, adam.AdamCachingPrefix, devUUID.String(), "logs")
}

//getInfoDirCache return info directory for devUUID for caching
func (adam *Ctx) getInfoDirCache(devUUID uuid.UUID) (dir string) {
	if adam.AdamCachingPrefix == "" {
		return adam.getInfoDir(devUUID)
	}
	return path.Join(adam.dir, adam.AdamCachingPrefix, devUUID.String(), "info")
}

//getMetricsDirCache return metrics directory for devUUID for caching
func (adam *Ctx) getMetricsDirCache(devUUID uuid.UUID) (dir string) {
	if adam.AdamCachingPrefix == "" {
		return adam.getMetricsDir(devUUID)
	}
	return path.Join(adam.dir, adam.AdamCachingPrefix, devUUID.String(), "metrics")
}

//getMetricsDirCache return metrics directory for devUUID for caching
func (adam *Ctx) getRequestDirCache(devUUID uuid.UUID) (dir string) {
	if adam.AdamCachingPrefix == "" {
		return adam.getRequestDir(devUUID)
	}
	return path.Join(adam.dir, adam.AdamCachingPrefix, devUUID.String(), "requests")
}

//getLogsDir return logs directory for devUUID
func (adam *Ctx) getLogsDir(devUUID uuid.UUID) (dir string) {
	return path.Join(adam.dir, "run", "adam", "device", devUUID.String(), "logs")
}

//getInfoDir return info directory for devUUID
func (adam *Ctx) getInfoDir(devUUID uuid.UUID) (dir string) {
	return path.Join(adam.dir, "run", "adam", "device", devUUID.String(), "info")
}

//getMetricsDir return metrics directory for devUUID
func (adam *Ctx) getMetricsDir(devUUID uuid.UUID) (dir string) {
	return path.Join(adam.dir, "run", "adam", "device", devUUID.String(), "metrics")
}

//getRequestDir return request directory for devUUID
func (adam *Ctx) getRequestDir(devUUID uuid.UUID) (dir string) {
	return path.Join(adam.dir, "run", "adam", "device", devUUID.String(), "requests")
}

//getLogsUrl return logs url for devUUID
func (adam *Ctx) getLogsUrl(devUUID uuid.UUID) string {
	resUrl, err := utils.ResolveURL(adam.url, path.Join("/admin/device", devUUID.String(), "logs"))
	if err != nil {
		log.Fatalf("ResolveURL: %s", err)
	}
	return resUrl
}

//getLogsUrl return info url for devUUID
func (adam *Ctx) getInfoUrl(devUUID uuid.UUID) string {
	resUrl, err := utils.ResolveURL(adam.url, path.Join("/admin/device", devUUID.String(), "info"))
	if err != nil {
		log.Fatalf("ResolveURL: %s", err)
	}
	return resUrl
}

//getMetricsUrl return metrics url for devUUID
func (adam *Ctx) getMetricsUrl(devUUID uuid.UUID) string {
	resUrl, err := utils.ResolveURL(adam.url, path.Join("/admin/device", devUUID.String(), "metrics"))
	if err != nil {
		log.Fatalf("ResolveURL: %s", err)
	}
	return resUrl
}

//getRequestUrl return request url for devUUID
func (adam *Ctx) getRequestUrl(devUUID uuid.UUID) string {
	resUrl, err := utils.ResolveURL(adam.url, path.Join("/admin/device", devUUID.String(), "requests"))
	if err != nil {
		log.Fatalf("ResolveURL: %s", err)
	}
	return resUrl
}

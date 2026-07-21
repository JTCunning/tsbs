package clickhousepromql

import (
	"fmt"
	"strings"
	"time"

	"github.com/timescale/tsbs/cmd/tsbs_generate_queries/uses/devops"
	"github.com/timescale/tsbs/pkg/query"
)

// Devops produces PromQL queries for all the devops query types.
type Devops struct {
	*BaseGenerator
	*devops.Core
}

// mustGetRandomHosts is the form of GetRandomHosts that cannot error; if it does error,
// it causes a panic.
func (d *Devops) mustGetRandomHosts(nHosts int) []string {
	hosts, err := d.GetRandomHosts(nHosts)
	if err != nil {
		panic(err.Error())
	}
	return hosts
}

func (d *Devops) GroupByOrderByLimit(qi query.Query) {
	panic("GroupByOrderByLimit not supported in PromQL")
}

func (d *Devops) LastPointPerHost(qq query.Query) {
	panic("LastPointPerHost not supported in PromQL")
}

func (d *Devops) HighCPUForHosts(qi query.Query, nHosts int) {
	panic("HighCPUForHosts not supported in PromQL")
}

// GroupByTime selects the MAX for numMetrics metrics under 'cpu'
// per minute for nhosts hosts,
// e.g. in pseudo-PromQL:
// max(
// 	{__name__=~"metric1|metric2...|metricN",hostname=~"hostname1|hostname2...|hostnameN"}
// ) by (__name__)
// evaluated with a 60 second step.
func (d *Devops) GroupByTime(qq query.Query, nHosts, numMetrics int, timeRange time.Duration) {
	metrics := mustGetCPUMetricsSlice(numMetrics)
	hosts := d.mustGetRandomHosts(nHosts)
	selectClause := getSelectClause(metrics, hosts)
	qi := &queryInfo{
		// max_over_time is not yet implemented in ClickHouse PromQL.
		query:    fmt.Sprintf("max(%s) by (__name__)", selectClause),
		label:    fmt.Sprintf("ClickHouse PromQL %d cpu metric(s), random %4d hosts, random %s by 1m", numMetrics, nHosts, timeRange),
		interval: d.Interval.MustRandWindow(timeRange),
		step:     "60",
	}
	d.fillInQuery(qq, qi)
}

// GroupByTimeAndPrimaryTag selects the AVG of numMetrics metrics under 'cpu' per device per hour for a day,
// e.g. in pseudo-PromQL:
//
// avg(
// 	{__name__=~"metric1|metric2...|metricN"}
// ) by (__name__, hostname)
// evaluated with a 3600 second step.
//
// Resultsets:
// double-groupby-1
// double-groupby-5
// double-groupby-all
func (d *Devops) GroupByTimeAndPrimaryTag(qq query.Query, numMetrics int) {
	metrics := mustGetCPUMetricsSlice(numMetrics)
	selectClause := getSelectClause(metrics, nil)
	qi := &queryInfo{
		// avg_over_time is not yet implemented in ClickHouse PromQL.
		query:    fmt.Sprintf("avg(%s) by (__name__, hostname)", selectClause),
		label:    devops.GetDoubleGroupByLabel("ClickHouse PromQL", numMetrics),
		interval: d.Interval.MustRandWindow(devops.DoubleGroupByDuration),
		step:     "3600",
	}
	d.fillInQuery(qq, qi)
}

// MaxAllCPU selects the MAX of all metrics under 'cpu' per hour for nhosts hosts,
// e.g. in pseudo-PromQL:
//
// max(
// 	{hostname=~"hostname1|hostname2...|hostnameN"}
// ) by (__name__)
// evaluated with a 3600 second step.
func (d *Devops) MaxAllCPU(qq query.Query, nHosts int, duration time.Duration) {
	hosts := d.mustGetRandomHosts(nHosts)
	selectClause := getSelectClause(devops.GetAllCPUMetrics(), hosts)
	qi := &queryInfo{
		// max_over_time is not yet implemented in ClickHouse PromQL.
		query:    fmt.Sprintf("max(%s) by (__name__)", selectClause),
		label:    devops.GetMaxAllLabel("ClickHouse PromQL", nHosts),
		interval: d.Interval.MustRandWindow(duration),
		step:     "3600",
	}
	d.fillInQuery(qq, qi)
}

func getHostClause(hostnames []string) string {
	if len(hostnames) == 0 {
		return ""
	}
	if len(hostnames) == 1 {
		return fmt.Sprintf("hostname='%s'", hostnames[0])
	}
	return fmt.Sprintf("hostname=~'%s'", strings.Join(hostnames, "|"))
}

// getSelectClause builds a PromQL series selector. The prometheus remote-write
// serializer used for loading stores each field key as the metric name without
// a measurement prefix (e.g. 'usage_user', not 'cpu_usage_user').
func getSelectClause(metrics, hosts []string) string {
	if len(metrics) == 0 {
		panic("BUG: must be at least one metric name in clause")
	}

	hostsClause := getHostClause(hosts)
	if len(metrics) == 1 {
		return fmt.Sprintf("%s{%s}", metrics[0], hostsClause)
	}

	metricsClause := strings.Join(metrics, "|")
	if len(hosts) > 0 {
		return fmt.Sprintf("{__name__=~'(%s)', %s}", metricsClause, hostsClause)
	}
	return fmt.Sprintf("{__name__=~'(%s)'}", metricsClause)
}

// mustGetCPUMetricsSlice is the form of GetCPUMetricsSlice that cannot error; if it does error,
// it causes a panic.
func mustGetCPUMetricsSlice(numMetrics int) []string {
	metrics, err := devops.GetCPUMetricsSlice(numMetrics)
	if err != nil {
		panic(err.Error())
	}
	return metrics
}

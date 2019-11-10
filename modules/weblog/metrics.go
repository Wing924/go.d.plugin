package weblog

import (
	"github.com/netdata/go.d.plugin/pkg/metrics"
)

func newWebLogSummary() metrics.Summary {
	return &weblogSummary{metrics.NewSummary()}
}

type weblogSummary struct {
	metrics.Summary
}

// TODO: temporary workaround?
func (s weblogSummary) WriteTo(rv map[string]int64, key string, mul, div int) {
	s.Summary.WriteTo(rv, key, mul, div)
	if _, ok := rv[key+"_min"]; !ok {
		rv[key+"_min"] = 0
		rv[key+"_max"] = 0
		rv[key+"_avg"] = 0
	}
}

type (
	metricsData struct {
		Requests     metrics.Counter `stm:"requests"`
		ReqUnmatched metrics.Counter `stm:"req_unmatched"`

		RespCode metrics.CounterVec `stm:"resp_code"`
		Resp1xx  metrics.Counter    `stm:"resp_1xx"`
		Resp2xx  metrics.Counter    `stm:"resp_2xx"`
		Resp3xx  metrics.Counter    `stm:"resp_3xx"`
		Resp4xx  metrics.Counter    `stm:"resp_4xx"`
		Resp5xx  metrics.Counter    `stm:"resp_5xx"`

		ReqSuccess  metrics.Counter `stm:"req_success"`
		ReqRedirect metrics.Counter `stm:"req_redirect"`
		ReqBad      metrics.Counter `stm:"req_bad"`
		ReqError    metrics.Counter `stm:"req_error"`

		UniqueIPv4      metrics.UniqueCounter `stm:"uniq_ipv4"`
		UniqueIPv6      metrics.UniqueCounter `stm:"uniq_ipv6"`
		BytesSent       metrics.Counter       `stm:"bytes_sent"`
		BytesReceived   metrics.Counter       `stm:"bytes_received"`
		ReqProcTime     metrics.Summary       `stm:"req_proc_time"`
		ReqProcTimeHist metrics.Histogram     `stm:"req_proc_time_hist"`
		UpsRespTime     metrics.Summary       `stm:"upstream_resp_time"`
		UpsRespTimeHist metrics.Histogram     `stm:"upstream_resp_time_hist"`

		ReqVhost          metrics.CounterVec `stm:"req_vhost"`
		ReqPort           metrics.CounterVec `stm:"req_port"`
		ReqMethod         metrics.CounterVec `stm:"req_method"`
		ReqURLPattern     metrics.CounterVec `stm:"req_url_ptn"`
		ReqVersion        metrics.CounterVec `stm:"req_version"`
		ReqSSLProto       metrics.CounterVec `stm:"req_ssl_proto"`
		ReqSSLCipherSuite metrics.CounterVec `stm:"req_ssl_cipher_suite"`
		ReqHTTPScheme     metrics.Counter    `stm:"req_http_scheme"`
		ReqHTTPSScheme    metrics.Counter    `stm:"req_https_scheme"`
		ReqIPv4           metrics.Counter    `stm:"req_ipv4"`
		ReqIPv6           metrics.Counter    `stm:"req_ipv6"`

		ReqCustomField  map[string]metrics.CounterVec `stm:"custom_field"`
		URLPatternStats map[string]*patternMetrics    `stm:"url_ptn"`
	}
	patternMetrics struct {
		RespCode      metrics.CounterVec `stm:"resp_code"`
		BytesSent     metrics.Counter    `stm:"bytes_sent"`
		BytesReceived metrics.Counter    `stm:"bytes_received"`
		ReqProcTime   metrics.Summary    `stm:"req_proc_time"`
	}
)

func newMetricsData(config Config) *metricsData {
	return &metricsData{
		ReqVhost:          metrics.NewCounterVec(),
		ReqPort:           metrics.NewCounterVec(),
		ReqMethod:         metrics.NewCounterVec(),
		ReqVersion:        metrics.NewCounterVec(),
		RespCode:          metrics.NewCounterVec(),
		ReqSSLProto:       metrics.NewCounterVec(),
		ReqSSLCipherSuite: metrics.NewCounterVec(),
		ReqProcTime:       newWebLogSummary(),
		ReqProcTimeHist:   metrics.NewHistogram(config.Histogram),
		UpsRespTime:       newWebLogSummary(),
		UpsRespTimeHist:   metrics.NewHistogram(config.Histogram),
		UniqueIPv4:        metrics.NewUniqueCounter(true),
		UniqueIPv6:        metrics.NewUniqueCounter(true),
		ReqURLPattern:     newCounterVecFromPatterns(config.URLPatterns),
		ReqCustomField:    newReqCustomField(config.CustomFields),
		URLPatternStats:   newURLPatternStats(config.URLPatterns),
	}
}

func (m *metricsData) reset() {
	m.UniqueIPv4.Reset()
	m.UniqueIPv6.Reset()
	m.ReqProcTime.Reset()
	m.UpsRespTime.Reset()
	for _, v := range m.URLPatternStats {
		v.ReqProcTime.Reset()
	}
}

func newCounterVecFromPatterns(patterns []userPattern) metrics.CounterVec {
	c := metrics.NewCounterVec()
	for _, p := range patterns {
		_, _ = c.GetP(p.Name)
	}
	return c
}

func newURLPatternStats(patterns []userPattern) map[string]*patternMetrics {
	stats := make(map[string]*patternMetrics)
	for _, p := range patterns {
		stats[p.Name] = &patternMetrics{
			RespCode:    metrics.NewCounterVec(),
			ReqProcTime: newWebLogSummary(),
		}
	}
	return stats
}

func newReqCustomField(fields []customField) map[string]metrics.CounterVec {
	cf := make(map[string]metrics.CounterVec)
	for _, f := range fields {
		cf[f.Name] = newCounterVecFromPatterns(f.Patterns)
	}
	return cf
}

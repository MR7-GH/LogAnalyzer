package telegram

import (
	"fmt"
	"sort"
	"strings"

	"LogAnalyzer/internal/analyze"
)

// FormatStats formats an analysis result as a Telegram message.
func FormatStats(result analyze.StatsResponse, detailed bool) string {
	var builder strings.Builder

	for i, service := range result.Services {
		if i > 0 {
			builder.WriteString("\n\n")
		}

		writeService(&builder, service, detailed)
	}

	return strings.TrimSpace(builder.String())
}

// writeService writes one service analysis in the compact Telegram format.
func writeService(builder *strings.Builder, service analyze.StatusResult, detailed bool) {
	fmt.Fprintf(builder, "Service: %s\n", service.Service)
	fmt.Fprintf(builder, "Total Errors: %d\n", service.Total)

	statusCodes := sortedStatusCodes(service.Codes)

	for _, status := range statusCodes {
		total := service.Codes[status]

		correlation := service.Correlation.ByStatus[status]

		fmt.Fprintf(
			builder,
			"%d: %d    nginx_recieved: %d\n",
			status,
			total,
			correlation.Matched,
		)

		if detailed {
			writeUnmatchedRequestIDs(builder, service, status)
		}
	}
}

// writeUnmatchedRequestIDs writes unmatched request IDs for one status code in detailed mode.
func writeUnmatchedRequestIDs(builder *strings.Builder, service analyze.StatusResult, status int) {
	if service.CorrelationDetail == nil {
		return
	}

	statusDetail, exists := service.CorrelationDetail.ByStatus[status]
	if !exists || statusDetail == nil {
		return
	}

	if len(statusDetail.UnmatchedRequestIDs) == 0 {
		return
	}

	builder.WriteString("  unmatched_request_ids:\n")

	for _, requestID := range statusDetail.UnmatchedRequestIDs {
		fmt.Fprintf(builder, "    - %s\n", requestID)
	}
}

// sortedStatusCodes returns status codes in ascending order.
func sortedStatusCodes(codes map[int]int64) []int {
	statusCodes := make([]int, 0, len(codes))

	for status := range codes {
		statusCodes = append(statusCodes, status)
	}

	sort.Ints(statusCodes)

	return statusCodes
}

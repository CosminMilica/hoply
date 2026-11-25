package eval

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"cv/internal/cvdoc"
)

// Table is a run's results as a Markdown table, the means last. Style is
// findings per sentence (lower is better); Judge is the judge's mean, 1 to 5.
func Table(results []Result) string {
	var b strings.Builder
	b.WriteString("| Case | By | ATS | Lines | Refused | Style | Most found | Judge | Human | Seconds |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |\n")
	var style, judge, human, seconds []float64
	for _, r := range results {
		judged, humanly := "", ""
		if r.Judge != nil {
			judged, humanly = fmt.Sprintf("%.1f", r.Judge.Mean), fmt.Sprint(r.Judge.Human)
			judge = append(judge, r.Judge.Mean)
			human = append(human, float64(r.Judge.Human))
		}
		style = append(style, r.Style.Score)
		seconds = append(seconds, r.Seconds)
		fmt.Fprintf(&b, "| %s | %s | %d/%d | %d | %d | %.2f | %s | %s | %s | %.0f |\n", r.Case, r.By.WrittenBy, r.Written.Ats.Passed,
			r.Written.Ats.Total, cvdoc.EstimatedLines(r.Written.Tailored), r.By.Refused, r.Style.Score, mostFound(r.Style.ByRule), judged, humanly, r.Seconds)
	}
	fmt.Fprintf(&b, "| **mean** | | | | | **%.2f** | | **%s** | **%s** | %.0f |\n", mean(style), meanText(judge), meanText(human), mean(seconds))
	return b.String()
}

func mostFound(byRule map[string]int) string {
	rules := make([]string, 0, len(byRule))
	for rule := range byRule {
		rules = append(rules, rule)
	}
	slices.SortFunc(rules, func(a, b string) int {
		if byRule[a] != byRule[b] {
			return byRule[b] - byRule[a]
		}
		return strings.Compare(a, b)
	})
	parts := []string{}
	for _, rule := range rules[:min(3, len(rules))] {
		parts = append(parts, fmt.Sprintf("%s %d", rule, byRule[rule]))
	}
	return strings.Join(parts, ", ")
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return math.Round(sum/float64(len(values))*100) / 100
}

func meanText(values []float64) string {
	if len(values) == 0 {
		return ""
	}
	return fmt.Sprintf("%.2f", mean(values))
}

// CompareTable is the blind comparison of runs a and b, case by case, with the
// wins counted overall and per dimension.
func CompareTable(cases []string, verdicts map[string]Verdict) string {
	var b strings.Builder
	b.WriteString("| Case | Winner | " + strings.Join(Dimensions, " | ") + " |\n|" + strings.Repeat(" --- |", len(Dimensions)+2) + "\n")
	wins := map[string]map[string]int{"winner": {}}
	for _, dimension := range Dimensions {
		wins[dimension] = map[string]int{}
	}
	for _, name := range cases {
		verdict, ok := verdicts[name]
		if !ok {
			continue
		}
		picks := []string{}
		wins["winner"][verdict.Winner]++
		for _, dimension := range Dimensions {
			picks = append(picks, verdict.Dimensions[dimension])
			wins[dimension][verdict.Dimensions[dimension]]++
		}
		fmt.Fprintf(&b, "| %s | %s | %s |\n", name, verdict.Winner, strings.Join(picks, " | "))
	}
	count := func(key string) string {
		return fmt.Sprintf("a %d · b %d · tie %d", wins[key]["a"], wins[key]["b"], wins[key]["tie"])
	}
	totals := []string{}
	for _, dimension := range Dimensions {
		totals = append(totals, count(dimension))
	}
	fmt.Fprintf(&b, "| **total** | %s | %s |\n", count("winner"), strings.Join(totals, " | "))
	return b.String()
}

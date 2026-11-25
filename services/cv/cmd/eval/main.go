// Command eval measures the CV writer on the cases in services/cv/eval
// (services/cv/eval/README.md). Run it through scripts/cv-eval.sh.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"cv/internal/claude"
	"cv/internal/eval"
	"cv/internal/upstream"
	"cv/internal/writer"
)

const usage = `usage: eval <command> [flags]
  export  -ids <job ids> [-out eval/local]       cases from the running profile and jobs services
  write   -out <run dir> [-cases dirs] [-only names]  write and lint a CV per case
  judge   -run <run dir> [-cases dirs]            score each CV with the judge model
  show    -run <run dir>                          print a run's table
  compare -a <run dir> -b <run dir> [-cases dirs] blind comparison of two runs`

func main() {
	if len(os.Args) < 2 {
		fail(usage)
	}
	flags := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	casesDirs := flags.String("cases", "eval/cases,eval/local", "comma-separated case directories")
	out := flags.String("out", "", "directory to write to")
	run := flags.String("run", "", "run directory")
	a := flags.String("a", "", "first run directory")
	b := flags.String("b", "", "second run directory")
	only := flags.String("only", "", "comma-separated case names")
	ids := flags.String("ids", "", "comma-separated job ids")
	_ = flags.Parse(os.Args[2:])
	ctx := context.Background()

	switch os.Args[1] {
	case "export":
		target := *out
		if target == "" {
			target = "eval/local"
		}
		sources := upstream.FromEnv()
		profile, _, err := sources.Profile(ctx)
		check(err)
		for _, id := range list(*ids) {
			job, _, err := sources.Job(ctx, uuid.MustParse(id))
			check(err)
			c := eval.Case{Name: eval.Slug(job.Company, job.Title), Profile: profile, Job: job}
			check(eval.SaveCase(target, c))
			fmt.Println("saved", c.Name)
		}
	case "write":
		cases := loadCases(*casesDirs, *only)
		w := writer.New(claude.New(claude.ConfigFromEnv())).KeepAnalyses(eval.NewAnalyses())
		results := eval.Write(ctx, w, cases, time.Now())
		check(eval.SaveRun(*out, results))
		fmt.Print(eval.Table(results))
	case "judge":
		results, err := eval.LoadRun(*run)
		check(err)
		results = eval.JudgeRun(ctx, judge(), loadCases(*casesDirs, ""), results)
		check(eval.SaveRun(*run, results))
		fmt.Print(eval.Table(results))
	case "show":
		results, err := eval.LoadRun(*run)
		check(err)
		fmt.Print(eval.Table(results))
	case "compare":
		first, err := eval.LoadRun(*a)
		check(err)
		second, err := eval.LoadRun(*b)
		check(err)
		cases := loadCases(*casesDirs, "")
		names := []string{}
		for _, c := range cases {
			names = append(names, c.Name)
		}
		fmt.Print(eval.CompareTable(names, eval.CompareRuns(ctx, judge(), cases, first, second)))
	default:
		fail(usage)
	}
}

// judge is the judge model: EVAL_JUDGE_MODEL, Opus by default, so the writer is not its own judge.
func judge() *claude.Client {
	cfg := claude.ConfigFromEnv()
	cfg.Model = "claude-opus-5-5"
	if model := strings.TrimSpace(os.Getenv("EVAL_JUDGE_MODEL")); model != "" {
		cfg.Model = model
	}
	return claude.New(cfg)
}

func loadCases(dirs, only string) []eval.Case {
	cases, err := eval.LoadCases(list(dirs)...)
	check(err)
	if names := list(only); len(names) > 0 {
		kept := []eval.Case{}
		for _, c := range cases {
			for _, name := range names {
				if c.Name == name {
					kept = append(kept, c)
				}
			}
		}
		cases = kept
	}
	if len(cases) == 0 {
		fail("no cases")
	}
	return cases
}

func list(value string) []string {
	items := []string{}
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}
	return items
}

func check(err error) {
	if err != nil {
		fail(err.Error())
	}
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}

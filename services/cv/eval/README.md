# CV writer evaluation

Measures how the CV writer writes, so a change to the prompts is kept only when it reads better. Runs by hand with
the local Claude token; never a gate.

```text
eval/
  cases/   committed: fictional people in other fields, each a profile and a job (Case in internal/eval)
  local/   gitignored: your own profile with your saved jobs (eval export)
  runs/    gitignored: what each version of the writer wrote, with its measures
```

| Measure | What | Better |
| --- | --- | --- |
| Style | findings per sentence of `internal/style`: tool lists, trailing "-ing" clauses, stock words, bullets that start or run alike, the summary formula, a skill named twice | lower |
| Judge | the judge model's 1–5 marks on specific, human, evidence, relevance and readable (`internal/eval/prompts`) | higher |
| Compare | the judge picks between two runs' CVs, asked in both orders; a pick that flips with the order is a tie | more wins |
| ATS, lines, refused, seconds | as the service reports them | |
| Letters | `eval letters` writes the cover letter and messages from each CV of a run; the same lint (`style.LintLetter`), judge and blind comparison (`compare -letters`) apply | |

The judge is Opus (`EVAL_JUDGE_MODEL`), so the writer (Sonnet) does not grade itself.

From the repository root, with the local stack up for `export`:

```bash
scripts/cv-eval.sh export -ids <job id>,<job id>              # your profile and saved jobs, from the local stack
scripts/cv-eval.sh write -out eval/runs/baseline              # about a minute per case
scripts/cv-eval.sh judge -run eval/runs/baseline
scripts/cv-eval.sh write -out eval/runs/next                  # after changing a prompt
scripts/cv-eval.sh judge -run eval/runs/next
scripts/cv-eval.sh compare -a eval/runs/baseline -b eval/runs/next
scripts/cv-eval.sh letters -run eval/runs/next                # letters from that run's CVs, then judge and
scripts/cv-eval.sh compare -letters -a eval/runs/x -b eval/runs/next  # compare as above
```

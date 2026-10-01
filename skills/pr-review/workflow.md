# Workflow for one pull request

The Master Reviewer authors a workflow for each pull request from the two skeletons below and passes each inline to
the Workflow tool. Neither is ever saved under `.claude/workflows/`.

What you may change per pull request: the slices, their `focus`, and the lenses, all through `args`. What you never
change: the models, `EFFORT`, `CAP`, and the A/B agreement rules.

## Review script

`args`:

| Key | Value |
|---|---|
| `pr` | the pull request's URL |
| `dir` | absolute path of the checkout holding the pull request |
| `diffPath` | absolute path of `diff.patch` |
| `promptDir` | absolute path of this skill's `prompts/` directory |
| `lenses` | the mode's lenses in order, without any stub lens, for example `["ponytail", "general"]` |
| `slices` | `[{name, files, focus}]`, at most 10 |
| `existing` | the contents of `existing.json` |

```js
export const meta = {
  name: 'pr-review',
  description: 'Review one pull request: reviewers per slice, A/B judges, A/B auditors',
  phases: [
    { title: 'Review', detail: 'one reviewer per slice, lens by lens', model: 'sonnet' },
    { title: 'Judge', detail: 'A and B verify every candidate and existing comment', model: 'opus' },
    { title: 'Audit', detail: 'A and B rule on the findings document', model: 'fable' },
  ],
}

const MODEL = { reviewer: 'sonnet', judge: 'opus', auditor: 'fable' }
const EFFORT = 'medium'
const CAP = 10
const RANK = ['NIT', 'LOW', 'ASK', 'ISSUE']
const CATEGORY = { type: 'string', enum: RANK }
const FINDINGS = {
  type: 'object',
  properties: { findings: { type: 'array', items: {
    type: 'object',
    properties: {
      path: { type: 'string' }, line: { type: 'integer' }, category: CATEGORY,
      title: { type: 'string' }, detail: { type: 'string' }, fix: { type: 'string' },
    },
    required: ['path', 'line', 'category', 'title', 'detail', 'fix'],
  } } },
  required: ['findings'],
}
const VERDICTS = {
  type: 'object',
  properties: { verdicts: { type: 'array', items: {
    type: 'object',
    properties: {
      id: { type: 'string' }, verdict: { type: 'string', enum: ['confirmed', 'refuted', 'duplicate'] },
      category: CATEGORY, evidence: { type: 'string' },
    },
    required: ['id', 'verdict', 'category', 'evidence'],
  } } },
  required: ['verdicts'],
}
const RULINGS = {
  type: 'object',
  properties: {
    rulings: { type: 'array', items: {
      type: 'object',
      properties: {
        id: { type: 'string' }, action: { type: 'string', enum: ['keep', 'drop'] },
        category: CATEGORY, note: { type: 'string' },
      },
      required: ['id', 'action', 'category', 'note'],
    } },
    missed: { type: 'array', items: { type: 'string' } },
  },
  required: ['rulings', 'missed'],
}

const { pr, dir, diffPath, promptDir, lenses, slices, existing } = args
if (slices.length > CAP) throw new Error(`${slices.length} slices; merge them down to ${CAP}`)
const inputs = `Pull request: ${pr}\nCheckout (read-only): ${dir}\nDiff: ${diffPath}`
const pair = (role, phase, prompt, schema) => parallel(['A', 'B'].map(n => () =>
  agent(prompt, { label: `${role}:${n}`, phase, model: MODEL[role], effort: EFFORT, schema })))
const byId = (answer, key) => new Map((answer ? answer[key] : []).map(v => [v.id, v]))

// Lenses run one after another, so at most slices.length reviewers are ever running.
const candidates = []
for (const lens of lenses) {
  const answers = await parallel(slices.map(s => () => agent(
    `Read ${promptDir}/reviewer.md and ${promptDir}/lens-${lens}.md and follow them.\n${inputs}\n` +
    `Your slice: ${s.name}\nFiles:\n${s.files.join('\n')}${s.focus ? `\nFocus: ${s.focus}` : ''}`,
    { label: `review:${lens}:${s.name}`, phase: 'Review', model: MODEL.reviewer, effort: EFFORT, schema: FINDINGS })))
  answers.forEach((a, i) => {
    if (!a) return log(`reviewer ${lens}:${slices[i].name} returned nothing; that slice is unreviewed for this lens`)
    for (const f of a.findings) candidates.push({ ...f, id: `${lens}-${candidates.length + 1}`, lens, source: 'reviewer' })
  })
}
for (const e of existing) candidates.push({ ...e, source: 'existing' })
const empty = { final: [], needsMaster: [], dropped: [], existing: [], missed: [], document: '' }
if (!candidates.length) return empty

const [ja, jb] = (await pair('judge', 'Judge',
  `Read ${promptDir}/judge.md and follow it.\n${inputs}\nCandidates:\n${JSON.stringify(candidates, null, 1)}`,
  VERDICTS)).map(a => byId(a, 'verdicts'))
const judged = candidates.map(c => {
  const votes = [ja.get(c.id), jb.get(c.id)]
  const yes = votes.filter(v => v && v.verdict === 'confirmed')
  // Two judges who confirm but disagree on the category get the milder one; the auditors can raise it.
  const category = yes.length ? RANK[Math.min(...yes.map(v => RANK.indexOf(v.category)))] : c.category
  return { ...c, category, status: ['refuted', 'disputed', 'confirmed'][yes.length],
    judgeA: votes[0] ? `${votes[0].verdict}: ${votes[0].evidence}` : 'no verdict',
    judgeB: votes[1] ? `${votes[1].verdict}: ${votes[1].evidence}` : 'no verdict' }
})
const theirs = judged.filter(c => c.source === 'existing')
const ours = judged.filter(c => c.source === 'reviewer' && c.status !== 'refuted')
const dropped = judged.filter(c => c.source === 'reviewer' && c.status === 'refuted')

const document = [
  `# Findings: ${pr}`, '',
  ...ours.map(f => [
    `## ${f.id} [${f.status}] ${f.category}: ${f.title}`, `${f.path}:${f.line} (lens: ${f.lens})`, '',
    f.detail, '', `Fix: ${f.fix}`, '', `Judge A: ${f.judgeA}`, `Judge B: ${f.judgeB}`, '',
  ].join('\n')),
  '# Existing comments', '',
  ...theirs.map(e => `- ${e.id} [${e.status}] ${e.author}${e.path ? ` ${e.path}:${e.line}` : ''}: ` +
    `${String(e.body).slice(0, 300)}\n  Judge A: ${e.judgeA}\n  Judge B: ${e.judgeB}`),
].join('\n')
if (!ours.length) return { ...empty, dropped, existing: theirs, document }

const audits = await pair('auditor', 'Audit',
  `Read ${promptDir}/auditor.md and follow it.\n${inputs}\nFindings document:\n${document}`, RULINGS)
const [ra, rb] = audits.map(a => byId(a, 'rulings'))
const final = [], needsMaster = []
for (const f of ours) {
  const a = ra.get(f.id), b = rb.get(f.id)
  const row = { ...f, auditorA: a ? `${a.action} ${a.category}: ${a.note}` : 'no ruling',
    auditorB: b ? `${b.action} ${b.category}: ${b.note}` : 'no ruling' }
  if (a && b && a.action === 'drop' && b.action === 'drop') dropped.push(row)
  else if (a && b && a.action === 'keep' && b.action === 'keep' && a.category === b.category) final.push({ ...row, category: a.category })
  else needsMaster.push(row)
}
return { final, needsMaster, dropped, existing: theirs, missed: audits.filter(Boolean).flatMap(a => a.missed), document }
```

How A and B combine:

| Stage | Both agree to keep | Both agree to drop | They differ |
|---|---|---|---|
| Judges | `confirmed` | `refuted`, dropped | `disputed`, sent on to the auditors |
| Auditors | `final`, when the category matches too | dropped | `needsMaster`: you decide |

An agent that returns nothing counts as a disagreement. With one judge missing, nothing is `confirmed` and the
auditors settle every finding; with one auditor missing, every finding comes to you.

## Edit script

Run after `review.json` is written. `args`: `promptDir` and `reviewPath`, the absolute path of `review.json`.

```js
export const meta = {
  name: 'pr-review-edit',
  description: 'Check the drafted pull request review for formatting and tone',
  phases: [{ title: 'Edit', detail: 'one editor reads the draft review', model: 'opus' }],
}

const REVIEW = {
  type: 'object',
  properties: {
    body: { type: 'string' },
    comments: { type: 'array', items: {
      type: 'object',
      properties: { path: { type: 'string' }, line: { type: 'integer' }, body: { type: 'string' } },
      required: ['path', 'line', 'body'],
    } },
    notes: { type: 'array', items: { type: 'string' } },
  },
  required: ['body', 'comments', 'notes'],
}
return await agent(
  `Read ${args.promptDir}/editor.md and follow it.\nDraft review: ${args.reviewPath}`,
  { label: 'editor', phase: 'Edit', model: 'opus', effort: 'medium', schema: REVIEW })
```

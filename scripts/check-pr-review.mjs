// Dry-runs the workflow skeletons in skills/pr-review/workflow.md against stub agents. It proves the two rules of the
// skill (at most 10 agents at once, medium effort, alias models) and the A/B agreement table hold in the script itself.
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const md = readFileSync(new URL('../skills/pr-review/workflow.md', import.meta.url), 'utf8')
const scripts = [...md.matchAll(/```js\n([\s\S]*?)```/g)].map(m => m[1].replace('export const meta', 'const meta'))
assert.equal(scripts.length, 2, 'workflow.md holds the review script and the edit script')
const AsyncFunction = (async () => {}).constructor

async function run(script, args, answer) {
  const calls = []
  let running = 0, peak = 0
  const agent = async (prompt, opts) => {
    calls.push({ prompt, ...opts })
    peak = Math.max(peak, ++running)
    await new Promise(r => setTimeout(r, 1))
    running--
    return answer(opts.label, prompt)
  }
  const parallel = thunks => Promise.all(thunks.map(t => t().catch(() => null)))
  const result = await new AsyncFunction('args', 'agent', 'parallel', 'log', script)(args, agent, parallel, () => {})
  return { result, calls, peak }
}

const finding = n => ({ path: 'a.go', line: n, category: 'ISSUE', title: `t${n}`, detail: 'd', fix: 'f' })
const verdict = (id, verdict, category = 'ISSUE') => ({ id, verdict, category, evidence: 'e' })
const ruling = (id, action, category = 'ISSUE') => ({ id, action, category, note: 'n' })
const slices = Array.from({ length: 10 }, (_, i) => ({ name: `s${i}`, files: [`f${i}.go`] }))
const args = {
  pr: 'https://example.test/pr/1', dir: '/co', diffPath: '/d.patch', promptDir: '/p',
  lenses: ['ponytail', 'general'], slices, existing: [{ id: 'x-1', author: 'bot', path: null, line: null, body: 'b' }],
}

// One reviewer finds five things; the rest find nothing. Ids are ponytail-1 .. ponytail-5.
const answers = (label) => {
  if (label === 'review:ponytail:s0') return { findings: [1, 2, 3, 4, 5].map(finding) }
  if (label.startsWith('review:')) return { findings: [] }
  if (label === 'judge:A') return { verdicts: [
    verdict('ponytail-1', 'confirmed'), verdict('ponytail-2', 'confirmed'), verdict('ponytail-3', 'refuted'),
    verdict('ponytail-4', 'confirmed', 'ASK'), verdict('ponytail-5', 'confirmed'), verdict('x-1', 'refuted')] }
  if (label === 'judge:B') return { verdicts: [
    verdict('ponytail-1', 'confirmed'), verdict('ponytail-2', 'refuted'), verdict('ponytail-3', 'duplicate'),
    verdict('ponytail-4', 'confirmed', 'ISSUE'), verdict('ponytail-5', 'confirmed'), verdict('x-1', 'refuted')] }
  if (label === 'auditor:A') return { missed: ['m'], rulings: [
    ruling('ponytail-1', 'keep'), ruling('ponytail-2', 'keep'), ruling('ponytail-4', 'keep', 'ASK'), ruling('ponytail-5', 'drop')] }
  if (label === 'auditor:B') return { missed: [], rulings: [
    ruling('ponytail-1', 'keep'), ruling('ponytail-2', 'drop'), ruling('ponytail-4', 'keep', 'ASK'), ruling('ponytail-5', 'drop')] }
  throw new Error(`unexpected agent ${label}`)
}

const { result, calls, peak } = await run(scripts[0], args, answers)
const ids = list => list.map(f => f.id).sort()
assert.equal(calls.length, 24, '10 reviewers per lens for 2 lenses, 2 judges, 2 auditors')
assert.ok(peak <= 10, `at most 10 agents at once, saw ${peak}`)
for (const c of calls) {
  assert.equal(c.effort, 'medium', `${c.label} runs at medium effort`)
  assert.ok(['sonnet', 'opus', 'fable'].includes(c.model), `${c.label} uses a model alias, got ${c.model}`)
}
assert.deepEqual(ids(result.final), ['ponytail-1', 'ponytail-4'], 'kept by both judges and both auditors')
assert.equal(result.final.find(f => f.id === 'ponytail-4').category, 'ASK', 'judges split on category: the milder one')
assert.deepEqual(ids(result.needsMaster), ['ponytail-2'], 'a split goes to the Master Reviewer')
assert.deepEqual(ids(result.dropped), ['ponytail-3', 'ponytail-5'], 'refuted by both judges, or dropped by both auditors')
assert.deepEqual(result.existing.map(e => [e.id, e.status]), [['x-1', 'refuted']], 'existing comments are judged, not audited')
assert.deepEqual(result.missed, ['m'])
assert.ok(!result.document.includes('ponytail-3'), 'refuted findings stay out of the findings document')

// A judge that dies leaves every finding disputed for the auditors; an auditor that dies leaves them to the Master.
const oneJudge = await run(scripts[0], args, label => (label === 'judge:B' ? null : answers(label)))
assert.ok(!oneJudge.result.document.includes('[confirmed]'), 'nothing is confirmed on one judge alone')
assert.deepEqual(ids(oneJudge.result.dropped), ['ponytail-3', 'ponytail-5'], 'one judge alone drops only what it refuted')
const oneAuditor = await run(scripts[0], args, label => (label === 'auditor:B' ? null : answers(label)))
assert.deepEqual(ids(oneAuditor.result.final), [], 'nothing is final on one auditor alone')
assert.equal(oneAuditor.result.needsMaster.length, 4)

await assert.rejects(run(scripts[0], { ...args, slices: [...slices, slices[0]] }, answers), /merge them down to 10/)

const edit = await run(scripts[1], { promptDir: '/p', reviewPath: '/r.json' }, () => ({ body: 'b', comments: [], notes: [] }))
assert.deepEqual(edit.calls.map(c => [c.model, c.effort]), [['opus', 'medium']])
assert.equal(edit.result.body, 'b')
console.log('pr-review workflow skeletons: ok')

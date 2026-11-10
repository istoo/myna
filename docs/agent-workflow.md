# How a change is run

A unit of work starts with its check, not with its code. This page says
what that check is, what a unit is, and where a person stands. It is the
working shape behind [the code standards](code-standards.md) and
[LESSONS.md](../LESSONS.md).

## Before the first edit: write the GREEN condition

A unit of work starts with the check it must pass, written as a command
and the output that makes it green:

```
GREEN  go test ./internal/gateway -run TestRoutingWindow exits 0
GREEN  gofmt -l is empty
GREEN  the diff touches only internal/gateway/routing.go
```

Not a check: "the output looks good", "the model says it is confident",
"no errors were raised". Absence of an error is not evidence of
correctness, and a loop built on it repeats a mistake confidently.

## One unit is one bounded job

One input, one output, one lens, its own context. Write down the variable
that crosses from one step to the next; that is a dependency. "And then"
is not: two steps with nothing crossing between them run at the same time.
Finding the arrows that carry nothing is usually the largest speedup
available.

## Four kinds of node

- **splitter**: cuts the work into units, by blast radius rather than by
  folder, so two units do not read the same thing.
- **worker**: one unit, one lens, its own context. Workers sharing a window
  converge on one finding and pay for one opinion with echoes.
- **code node**: merging, ranking, deduplicating, diffing. If the step can
  be described without judge, decide, assess or summarize, it is code, not
  a model.
- **gate**: reads evidence and decides open, sent back, or held.

## Loop inside, graph between

Inside a unit: produce, check, correct, repeat until green.
Between units: split, fan out, merge, gate, send back. Neither list is
expressible from the other.

## A gate reads evidence in this order

1. deterministic results — build, vet, the GREEN commands;
2. this run's trajectory — what was tried and what was reverted;
3. how often this area has been rolled back before;
4. the model's assessment, last.

Lanes, by what a wrong result would cost to undo:

| Lane | What is in it | When it opens |
| --- | --- | --- |
| Reversible and contained | a copy change, a test, an isolated function with coverage | first |
| Reversible but wide | a shared utility, a schema addition, anything many callers touch | deterministic checks and a clean trajectory |
| Hard to reverse | migrations, deletions, writes to real data, releases | a person opens it; no score does |

## Return the unit, not the batch

A red unit goes back alone, with its verdict, its reason, its evidence and
its scope. Units that were green are not rewritten: their next version
would be different, not better. A returned unit that grows past its scope
is a diff nobody reviewed.

## Two return paths

- **correction edge** — the gate sends one unit back to the step that made
  it; it fixes this run.
- **learning edge** — an accepted result becomes a constraint where the
  work is cut, so every later run starts where this one ended. In this
  repository that place is [LESSONS.md](../LESSONS.md), under
  [What an accepted result teaches](../LESSONS.md#what-an-accepted-result-teaches).

A system with only the first is fast and never gets smarter.

## Where a person stands

At the highest consequence and the lowest reversibility: approving a
merge, choosing what ships, deleting data, publishing. Not reviewing
intermediate output. A person inside the graph is its slowest node, and
the graph then runs as fast as one person reading.

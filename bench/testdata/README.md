# Corpora

The fixed inputs the benchmarks run against.

A benchmark is a comparison between two runs, so the input has to be the same in both of them.
Every file here is versioned for that reason, and changing one invalidates every historical number for the benchmarks that read it.
That is not a rule about being careful with the files, it is what the numbers mean: a router benchmark against a different route table is measuring a different thing, and comparing the two says the framework moved when the input did.

When a corpus genuinely has to change, change it and reset the baseline in the same commit, and say so in the commit message so the gap in the history has a reason next to it.

Every file in this directory is listed below, and `benchrun lint` fails when one is not.
The list is here rather than in a header comment inside each file because a corpus is often a format with nowhere to put a comment, and one place to read beats two conventions.

| File | Read by | What it is |
|---|---|---|
| `hashes.txt` | `hash/argon2id/verify`, `hash/bcrypt/verify` | One encoded password hash per algorithm, all of the same password. The parameters written into each hash are what the verify costs, so they are pinned here rather than chosen at run time. |

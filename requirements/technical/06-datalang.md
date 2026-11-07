# DataLang integration

Design notes for REQ-DL-1 through REQ-DL-6. The public behavior is the feature file. This file records choices that are not stakeholder requirements. The in-process interpreter was accepted by the user on 2026-11-07 ("Der erste klingt gut").

## Isolation

The interpreter is a closed evaluator in the Keimling process. It has a fixed set of pure functions and no binding to files, the network, processes, or the environment. A deadline is checked on expression steps and grammar iterations. There is no step ceiling. A run stops when that deadline is reached.

A separate WebAssembly runtime would also keep host I/O away, and it would add a large module. That conflicts with REQ-DL-4, so it is not used. The security property is the absence of host operations in the evaluator, not a second process. A single regular expression runs to completion; the patterns are RE2, so their time grows with the input. The default timeout is 30 seconds. `timeout_ms` accepts any positive integer. A value whose duration would overflow the clock type waits for the longest duration that type can represent. There is no shorter cap.

## Tool names

The Responses API rejects a tool name that contains a dot. The tools are `data_processing` and `data_processing_help`.

## What the model sees

MCP, exec, and `data_processing` results are stored on the agent. A result larger than 4 KiB is replaced, before it is written into the model transcript, by a reference, a preview, and a schema hint. A smaller result keeps its text and gains a `ref:` line so a later `data_processing` can still name it. Nothing is truncated and then described as the whole result.

Exec in this harness already joins stdout and stderr. The stored CLI value therefore has the transcript in `stdout`, an empty `stderr`, and `exit_code`.

## Several inputs

`input_ref` is the value named `input`. `inputs` is a list of `{name, ref}`. Each name becomes a variable for that call. `lookup(rows, field, key)` returns the first object whose field equals the key, or null. Membership stays `contains`. A name must be an identifier and must not repeat. `input` is reserved for `input_ref`.

A `/regex/` written as an expression is a grammar rule. `extract` and `parse` accept it directly, so a filter does not need a `grammar` block. The tool card shows that program and the references, with no length cutoff.

`and`, `or`, and `not` are the same operators as `&&`, `||`, and `!`. The tool description names both forms.

Streaming a result that does not fit in memory is later work. Left recursion that consumes no input is rejected rather than rewritten. CSV and XML are included: CSV rows become objects from the header line, and XML is a tag tree with external entities refused.

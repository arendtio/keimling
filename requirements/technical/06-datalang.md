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

`outline` walks a value, or a string it can read as JSON, XML, or CSV, and writes names, types, and list lengths. It stops at 4 KiB and does not write string values. A whole-file read of a larger JSON, XML, or CSV file uses that shape as its preview. Other text keeps the byte preview. A binary file reports its size and a short run of readable characters.

A delegate `source` is a file path, a tool ref, or a history section id. The harness resolves it and sends the bytes only on the delegate call. Traces to REQ-DD-2. The budget is about 80 percent of the current model's context window, in bytes at four bytes per token, with no fixed byte ceiling. A missing window below 1,000 tokens is treated as 8,000, which is the usual configured minimum, not a ceiling. A larger source is packed by JSON elements, XML children, or lines. At most eight parts are asked the question, then one call combines the answers. A remainder is named in the tool result. A file source is read up to those eight parts, so the read grows with the window. Traces to REQ-DL-10. A DataLang file input has no size ceiling. A file that fits in 8 MiB is still the text, so a program can return it. A larger file is opened, not loaded: `csv`, `json`, `xml`, and `outline` scan it. `filter`, `map`, `take`, `first`, `count`, and `lookup` pull rows and can stop at the first hit. `outline` counts in constant memory and still stops its text at 4 KiB. `sort`, `group`, and `unique` hold the rows they reorder. Returning the large file itself asks for the rows instead. The result store stays 32 MiB.

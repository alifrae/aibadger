# Protocol

Badger bridges your local project and an AI chat in a three-step exchange: **Map → Extract → Apply**.

The goal editor may carry separate removable attachments, such as large pasted diffs or supporting notes. Those attachments stay outside the typed instruction surface and are assembled only when the goal is submitted.

Bare/default interactive startup selects Design focus. When an interactive
Design submission is empty, with no attachments, Badger uses this internal
exploratory task and proceeds through the normal Map → Extract flow:

```text
Explore this project with an open mind. Explain what stands out, how its main parts fit together, and surface any interesting opportunities, risks, or improvements worth investigating.
```

The exploratory task is not displayed in the editor beforehand. Code remains
explicit through `badger code` or `/code`; non-interactive API callers continue
to select `--focus <code|design>` explicitly. Typed goals and attachment-only
submissions retain their supplied content.

## Step 1: Map

**Prompt 1 (Map)** — the project topology — has this structure:

- **PROJECT TOPOLOGY** — languages, build stack, and module structure.
- **SOURCE TREE** — packages with file names and sizes, grouped by priority
  (docs, config, source code, assets).
- **EXTERNAL CONTEXT** — optional read-only context roots configured outside
  the normal project tree. P1 named sources can also expose a stable label,
  include scope, and current Git revision when available.
- **USER TAGGED FILES** — optional user-selected files you pin into the goal
  with `@path/to/file`; the section appears only when those references resolve.
  References inside fenced code, diffs, or attachment payloads are treated as
  literal context rather than file tags.
- **TASK** — your goal or question.
- **CONSTRAINT** — instructs the AI to reply with selectors only.

No source code is included.

Copy **Prompt 1 (Map)** and paste it into an AI chat.

## Step 2: Extract

The AI reads the topology and replies with selectors for the context it needs:

- `FILE:path` — extracts the entire file.
- `PREFIX:path#literal prefix` — finds the first line whose trimmed content starts with the prefix, then extracts a logical code block.
- `NEAR:path#literal string` — finds the first line containing the literal string, then extracts a logical code block.
- `SYMBOL:path#name` — requests a bounded span around a named/distinctive symbol in a **known file**. Today this resolves through the same local span machinery as `NEAR`; it is not an AST query.
- `REFERENCES:literal` — searches project-local text for the literal and returns bounded `NEAR` spans from at most 12 matching files.
- `TESTS:literal` — searches likely test files for the literal and returns bounded `NEAR` spans from at most 8 matching files.
- `SEARCH:literal text` — bounded project-local literal search returning `NEAR` spans from at most 12 matching files.

The discovery selectors are deliberately **literal and bounded**. They do not use an LSP, compiler symbol table, AST index, or semantic type system. `REFERENCES:FrameProvider` means “find a small set of text files containing `FrameProvider`,” not “return every type-aware reference to this symbol.”

Discovery searches inspect at most 5,000 eligible project files per request, skip known noisy/generated dependency directories, skip binary/assets, and skip files larger than 1 MiB. Normal Prompt 2 safety/egress filtering still runs on the resulting extracts.

Prefer the cheapest selector that already identifies the necessary context. A typical ordering is:

1. `FILE` / `PREFIX` / `NEAR` when the topology already tells you where to look;
2. `SYMBOL` when the file is known but a bounded declaration/implementation span is enough;
3. `TESTS`, `REFERENCES`, or `SEARCH` only when local discovery is actually needed.

Example:

```text
FILE:internal/scanner/scanner.go
SYMBOL:internal/scanner/scanner.go#Scan
TESTS:Scan
REFERENCES:NewRustDetector
SEARCH:Cargo.toml
```

### External context paths

Repo-local files are resolved first. If no repo-local file matches a file/span selector, Badger may resolve it against configured read-only external context.

Legacy `.badger-context` supports the existing relative/display/suffix resolution rules. P1 named sources configured in `.badger.toml` have explicit display identities such as:

```text
@algorithm-core
```

so selectors can be explicit:

```text
FILE:@algorithm-core/src/detection/energy.rs
SYMBOL:@algorithm-core/src/detection/energy.rs#compute_energy
```

Named-source `include` rules are enforced during extraction and tagged-file resolution. External sources remain read-only and cannot be patch targets. If the external root is a Git worktree, Prompt 1 policy metadata also shows the current revision so the AI can distinguish evidence from different checkouts.

Ambiguous external matches fail with a candidate list instead of guessing.

Extraction attempts to include relevant comments preceding the matched line.
Badger searches for structural blocks (balanced braces, indentation, or
declarations) within a lookahead limit; if structural detection fails, it
falls back to a 10-line window (3 before, 6 after the match).

Copy the AI's reply and paste it back into Badger. Badger extracts the
relevant code and produces **Prompt 2 (Code Context)** — the extracted files or
code blocks with their full contents, alongside the project topology and task.

Prompt 2 has this structure:

- **PROJECT TOPOLOGY** — languages, build stack, module structure, and active extraction count.
- **TASK** — your goal or question.
- **OUTPUT CONSTRAINT** — instructs the AI to answer using the provided context, not selector lines.
- **CONTEXT** — extracted file blocks such as `(Full File)`, `(Extracted Span)`, or `(Binary Summary)`.

Interactive Review and Deep Review integrations use the same selectors as an
optional continuation escape hatch. Their initial review prompt asks for
immediate findings when the supplied diff and supporting context are
sufficient. Ordinary findings end the review without another Badger call. A
selector-only response can be pasted into interactive Review or passed to
`api review-continuation`; the supplemental payload contains only current
requested file context and compact review framing, so it does not resend the
initial diff. Files reflect the filesystem at continuation time rather than a
persisted review snapshot.

Interactive Review composes generated review context into the normal Prompt 1
schema, so it includes `[PROJECT TOPOLOGY]` and `[SOURCE TREE]`. The stable
`api review-context` output is intentionally standalone and topology-free; it
contains the review instructions, authoritative diff, status, guidance, and
eligible supporting context only. Integrations must not assume the API and TUI
Prompt 1 are byte-for-byte equivalent.

## Step 3: Apply

Copy **Prompt 2 (Code Context)** back to the AI chat. Without a patch-only project policy, the legacy AI write protocol remains:

- `--- File: <path> ---` ... content ... `--- End File ---` — creates or updates a file.
- `--- Delete File: <path> ---` — deletes a file.

```text
--- File: cmd/main.go ---
package main

func main() {}
--- End File ---
```

When `.badger.toml` enables `write.patch_only = true`, the AI must instead return a guarded unified diff. See [Project Policy](project-policy.md) for patch validation, snapshot pinning, and post-apply review behavior.

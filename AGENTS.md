# Repository Instructions

## Environment And Editing

- Detect the current OS and shell before running repository commands.
- Treat repository text as UTF-8 and verify Korean text is not corrupted.
- Baton and repository workflows do not require a specific shell. On
  Windows, PowerShell, cmd, and Git Bash are supported; Git workflows require
  `git` to be available on `PATH`.
- Use Go 1.26 or newer for source builds and tests.
- Use `apply_patch` for repository edits. After changes, inspect the result and
  run `git diff --check`.

<baton-rules>

## Baton

This project follows Baton. Before doing any work here, read
`.baton/PROTOCOL.md`: it decides which role you hold and which role file to
read next. Until you have read it, do not run a `baton` command and do not
write under `.baton/`.

</baton-rules>

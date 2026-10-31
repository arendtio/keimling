# Feature: Interactive exec flag

## Intent

`interactive=true` means the command needs an interactive terminal. The command
text must not turn that flag off.

Origin: user, 2026-10-31. "the root cause is a wrong interactive=true. You are
masking that error with all those cases. That is stupid. So please build it
clean without all those exceptions."

## Requirements

### REQ-EX-1 Honor the flag

An `exec` call with `interactive=true` takes the interactive path. Keimling
does not reclassify it as a normal command because the text looks read-only.

Fit: `pwd` and `ls` with `interactive=true` both request a terminal. Without a
terminal, both return the same unavailable error and neither returns the
command's stdout.

### REQ-EX-2 Name the host system

The model is told the operating system and the shell `exec` actually runs. It is not told that it is a Linux-only assistant.

Origin: user, 2026-10-31. "I dont' like those Linux references. I mean, yes here is runs on Linux but since it is a Go binary it can run on Windows or MacOS similarly. Speaking of which maybe it is relevant for the AI to know on why system it is running since on Windows it might want to use powershell commands and not POSIX commands?"

Fit: the instructions contain the host operating system and the shell path. They do not contain "Linux CLI" or "Linux shell". On Windows, `exec` runs PowerShell.

# Feature: Live call

## Intent

A person can talk with the assistant in a live voice call, in a workspace
thread or an agent. The same answer is heard and written into that chat.
The call starts with the chat the next text turn would already see. Typing
during the call goes into the call. The separate text model stays quiet
until the call ends.

Screen sharing is a later step. The first call does not capture the display.

## REQ-RT-1 One live call

- **Statement:** Keimling shall open a live voice call on the provider and model chosen for that call, for the open workspace thread or the open agent.
- **Origin:** User, 2026-11-07 (paraphrase of using the OpenAI Realtime API, including through other providers that offer it, for agents and threads).
- **Fit criterion:** With a realtime provider and model saved, Call connects on the current thread and on an agent, in the browser and in the desktop window. The browser uses WebRTC. The desktop window uses a websocket and microphone PCM when that window has no WebRTC, and plays the assistant's spoken reply for the whole turn as one continuous stream. Interrupt stops that playback. Speaking while the reply is still playing stops it too, and that speech is one turn. After an interrupt, the next spoken turn works without starting the call again. Hang up ends it. The transcription microphone still records a message and does not place the call.
- **Priority:** shall
- **Status:** confirmed

## REQ-RT-2 History at the start

- **Statement:** The call shall start with the chat history the next text turn would see, as text.
- **Origin:** User, 2026-11-07 (quote: "Kriegt es die gesamte Chat-Historie zum Beispiel" and "Ich wäre erstmal zu allen Fragen … ja").
- **Fit criterion:** A thread that already contains a user line and an assistant line produces those lines as text items in the call setup. Raw audio is not required.
- **Priority:** shall
- **Status:** confirmed

## REQ-RT-3 Speech appears in the chat

- **Statement:** What is said in the call shall appear in the text window. The spoken reply and the written reply are the same answer.
- **Origin:** User, 2026-11-07 (quote: "gesagte soll direkt auch im Textfenster erscheinend" and "bietet die realtime API nicht die möglichkeit die textevents auch zu bekommen?").
- **Fit criterion:** Input transcription is on for the call. After a spoken turn, the thread shows the recognized user speech and the assistant transcript. Muting the microphone closes that turn, so the transcript appears without more microphone audio. Reloading the thread still shows both. A second model does not write a separate reply during the call.
- **Priority:** shall
- **Status:** confirmed

## REQ-RT-4 Typing during the call

- **Statement:** Text sent while the call is open shall reach the live call. The text model shall not start its own run until the call ends.
- **Origin:** User, 2026-11-07 (quote: "während das Gespräch läuft, kann man in der Zwischenzeit auch weiter mit der KI chatten, kriegt das Gespräch das mit als Events" and "ja").
- **Fit criterion:** Sending a line during a call leaves the agent idle, stores the line in the thread, and the call setup can accept that line as a text event. A wake that arrives during the call is delivered to the call the same way.
- **Priority:** shall
- **Status:** confirmed

## REQ-RT-5 Same tools and approval

- **Statement:** A tool the call asks for shall run with the same grant and the same approval as that thread or agent.
- **Origin:** User, 2026-11-07, agreeing that the call works with agents and threads, which already carry those rights.
- **Fit criterion:** A granted tool, including an MCP tool, runs as that function and its result is stored on the thread. The call does not run an `mcp__` name as a shell command. A tool the agent is not granted does not run. Ask mode still waits for Approve or Deny.
- **Priority:** shall
- **Status:** confirmed

## REQ-RT-6 Its own setting

- **Statement:** The live call shall have its own provider, model, and voice in Settings. Transcription shall keep its own provider and model.
- **Origin:** User, 2026-11-07 (paraphrase of needing something in Settings, and the earlier rule that voice transcription chooses its own service).
- **Fit criterion:** Saving a realtime provider, model, and voice survives a restart. Clearing the provider turns the call off and leaves the transcription model unchanged.
- **Priority:** shall
- **Status:** confirmed

## Out of scope

Sending a screen share into the call. The live share itself is
[REQ-SS-1](20-screen-share.md). A later requirement can hand that picture to
the model.

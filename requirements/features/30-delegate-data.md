# Feature: Ask a delegate about data

## REQ-DD-1 A delegate can read a named source

- **Bin:** explicit requirement
- **Statement:** The user shall be able to ask a question about a file, a stored tool result, or a section of this chat through a delegate. The bytes shall be sent only on that delegate call. The main conversation shall keep the question and the answer. A source larger than one delegate call shall be split by its records, the same question asked of each part, and the answers combined. The main conversation shall still be one question and one answer.
- **Origin:** User, 2026-11-13 (quote: "ich hab beispielsweise irgend 'ne unbekannte Datei, ich schmeiß die gegen ein LLM mit irgend 'ner Fragestellung und ich krieg irgend 'ne sinnvolle Antwort zurück" and "ich kann damit einen relativ großen Datei relativ schnell analysieren" and "in meinem Hauptthread, also da, wo ich meine Unterhaltung führe, diesen ganzen Kontextmüll gar nicht drin habe"). Same day, confirming the shape: "klingt gut, bitte umsetzen" after the proposal that the harness loads a path, a reference, or a section, and splits a larger source by records.
- **Rationale:** Pasting the file into the conversation spends the main context on bytes the question does not need to keep, and a file the model cannot see the shape of cannot be queried exactly.
- **Fit criterion:** A delegate call that names a file path sends that file's text on the delegate request and the tool result is the answer, not the file. A list larger than one delegate call is sent as more than one part, each part smaller than the call budget, and the tool result is one combined answer. The tool arguments do not contain the file text.
- **Priority:** shall
- **Status:** confirmed

## REQ-DD-2 One call follows the current model's context

- **Bin:** explicit constraint
- **Statement:** The amount of source text placed in one delegate call shall follow the context size of the current model. There shall be no fixed byte ceiling on that amount.
- **Origin:** User, 2026-11-13 (quote: "ich find das gut, das an die Kontextgröße des aktuellen Modells zu hängen" and "ein hartes Limit will ich eigentlich gar nicht. Weil das ist ja auch nicht besonders zukunftsträchtig, wenn's demnächst Modelle gibt, die vielleicht noch größere Kontexte verarbeiten können").
- **Rationale:** A fixed ceiling stays small after models with larger windows exist, so a file that would fit in the current model is split or dropped for a number that no longer matches the model.
- **Fit criterion:** With the current model configured for 1,000,000 tokens, one delegate call can carry a source larger than 400 KiB. With the current model configured for a smaller window, a source larger than that window is split, and each part stays within the window. Raising the configured window raises the bytes one call can carry, without a second constant cap.
- **Priority:** shall
- **Status:** confirmed

# Skills in the prompt

Keimling lists installed Agent Skills, including ones discovered from another harness, in the model request.

## REQ-SK-1 Turn a skill off

- **Bin:** explicit requirement
- **Statement:** The user shall be able to turn each discovered skill on or off. A skill that is off shall not be listed in the model request and shall not be loaded when the model asks for it.
- **Origin:** User, 2026-10-31 (quote: "we also make use of the SKILLs from other harnesses like Cursor. I would like to be able to activate and deactivate those skill" and "for my lokal model the pollute the context very much and do not match my use-cases").
- **Rationale:** A local model pays for every skill name and description in the prompt. Skills written for another harness are not always wanted.
- **Fit criterion:** A discovered skill can be turned off. The next model request does not list it. Turning it on lists it again. Asking to load a skill that is off does not put its text into the request.
- **Priority:** shall
- **Status:** confirmed

## REQ-SK-2 Choose whether a new skill starts on

- **Bin:** explicit requirement
- **Statement:** The user shall choose whether a skill that appears later starts included in the prompt or stays out until the user turns it on. The choice shall be changeable later.
- **Origin:** User, 2026-10-31 (quote: "I would like to have an option if new skill must be enabled or are enabled automatically, because currently I want to enabled them manuelly but in the future the opposite might be the case").
- **Rationale:** The same person may want new skills held back now and included automatically later. Changing the choice must not silently turn back on a skill they already turned off.
- **Fit criterion:** With the option off, a skill that appears after that stays out of the prompt until the user turns it on. With the option on, a skill that appears after that is in the prompt. A skill the user has turned off stays off when the option changes.
- **Priority:** shall
- **Status:** confirmed

## Design notes (not requirements)

Skills already present when this choice is first saved stay as they are, which today means included, until the user turns one off. The option applies to a skill the first time it is seen after that. `auto_enable_new` defaults off.

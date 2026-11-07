# Vision through delegate

Design for REQ-CF-8 and REQ-CF-9. The public behavior is the feature file.

## Where the bytes go

`expandChatFiles` never adds an `input_image` part. The user message keeps the path. When the chat model can see, that message tells the model to call `delegate` with `image` set to the path and `task` set to the question. A sound is still included when the chat model can hear.

The journal stores the path trailer, as before. The delegate tool result stores the text answer. Image bytes are not written into either.

## Which model looks

Seeing is a switch on the chat model. Delegate profiles stay on the Settings endpoint and are not that switch. A delegate call with an empty `image` still uses the profile model on Settings.

A delegate call with a path uses the chat model that has seeing on, on that model's provider, in one isolated request: no tools, no conversation, the question plus the image. `profile` is unused on that call. The reply is text. The same path can be sent again with a new question.

When seeing is off, the call returns an error and sends no bytes. The path stays available for tools.

## Limits

The picture must be a readable image file within the existing 4 MiB image limit. The isolated call waits up to 180 seconds. A text delegate is unchanged.

## The question

The isolated call's task is the latest user message, with the file trailer removed. A task the chat model wrote is not substituted for that message. Earlier user and assistant text from the conversation is passed as context, along with the context argument, so a fact the question depends on is present. The delegate is told to answer at the level of that question.

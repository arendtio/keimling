# Feature: File preview

## REQ-RF-1 A large whole-file read is a preview and the path

- **Bin:** explicit requirement
- **Statement:** Reading a whole file that is larger than the inline limit shall give the model the path, the size, and a short preview. It shall not give the model the rest of the file. A line range the model names shall still be the exact bytes of that range.
- **Origin:** User, 2026-11-12, confirming the reference shape (quote: "ok, dann bitte umsetzen" after the proposal that a large file arrives as a preview plus its path, and a requested line range stays exact).
- **Rationale:** The path is the durable address of a file. Putting a large file into the chat spends the context on bytes the model can ask for by range, or hand to a transform by path.
- **Fit criterion:** A whole-file read of a file larger than 4 KiB contains the path and a preview, and does not contain a marker that sits past that preview. A read of the line that holds the marker contains the marker and a hash of those exact bytes. A whole-file read of a file of 4 KiB or less still contains the file text.
- **Priority:** shall
- **Status:** confirmed

## REQ-RF-2 A large structured file shows its shape

- **Bin:** explicit requirement
- **Statement:** The preview of a large JSON, XML, or CSV file shall be the shape of that file. A large file that is not text of those kinds shall give the path, the size, and a short sample of readable text when it has any.
- **Origin:** User, 2026-11-13, confirming the shape as the first look (quote: "klingt gut, bitte umsetzen" after the proposal that a large structured file shows its anatomy instead of its first bytes, and a binary file shows its size, a recognized kind, and a short readable sample).
- **Rationale:** The first bytes of a long list are often one record and do not show the shape.
- **Fit criterion:** A whole-file read of a JSON file larger than 4 KiB contains a field name and the length of its list, and does not contain a value that sits only inside that list. A whole-file read of a binary file contains the size and does not contain the file's raw bytes as the preview text.
- **Priority:** shall
- **Status:** confirmed

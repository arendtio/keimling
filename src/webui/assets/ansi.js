// Minimal SGR ANSI → HTML (no deps). Escapes text; strips unknown CSI.
(function (global) {
  var FG = {
    30: "#1c1c1c", 31: "#e07a7a", 32: "#7dcea0", 33: "#e0c07a",
    34: "#7aa2e0", 35: "#c07ae0", 36: "#7ad4e0", 37: "#e8eaef",
    90: "#9aa3b5", 91: "#ff9a9a", 92: "#a8e6c3", 93: "#f0d78a",
    94: "#9abcff", 95: "#d9a8ff", 96: "#9eecf5", 97: "#ffffff"
  };

  function esc(s) {
    return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
  }

  function styleAttr(st) {
    var parts = [];
    if (st.fg) parts.push("color:" + st.fg);
    if (st.bold) parts.push("font-weight:600");
    if (st.dim) parts.push("opacity:0.65");
    return parts.length ? ' style="' + parts.join(";") + '"' : "";
  }

  function applySgr(st, params) {
    if (!params.length || (params.length === 1 && params[0] === "")) {
      st.fg = null; st.bold = false; st.dim = false;
      return;
    }
    for (var i = 0; i < params.length; i++) {
      var n = parseInt(params[i], 10);
      if (isNaN(n) || n === 0) {
        st.fg = null; st.bold = false; st.dim = false;
      } else if (n === 1) st.bold = true;
      else if (n === 2) st.dim = true;
      else if (n === 22) { st.bold = false; st.dim = false; }
      else if (n === 39) st.fg = null;
      else if (FG[n]) st.fg = FG[n];
    }
  }

  function ansiToHtml(input) {
    if (!input) return "";
    var st = { fg: null, bold: false, dim: false };
    var out = "";
    var i = 0;
    while (i < input.length) {
      if (input.charCodeAt(i) === 0x1b && input.charAt(i + 1) === "[") {
        var j = i + 2;
        while (j < input.length) {
          var c = input.charAt(j);
          if ((c >= "a" && c <= "z") || (c >= "A" && c <= "Z")) break;
          j++;
        }
        if (j >= input.length) break;
        var final = input.charAt(j);
        var body = input.slice(i + 2, j);
        i = j + 1;
        if (final === "m") applySgr(st, body.split(";"));
        // drop cursor/erase sequences (e.g. 2K)
        continue;
      }
      if (input.charAt(i) === "\r") {
        i++;
        continue;
      }
      var start = i;
      while (i < input.length && input.charCodeAt(i) !== 0x1b && input.charAt(i) !== "\r") i++;
      var chunk = esc(input.slice(start, i));
      if (!chunk) continue;
      out += "<span" + styleAttr(st) + ">" + chunk + "</span>";
    }
    return out;
  }

  global.ansiToHtml = ansiToHtml;
})(typeof window !== "undefined" ? window : this);

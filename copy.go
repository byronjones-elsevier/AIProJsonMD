package main

import (
	"crypto/sha256"
	"encoding/base64"
)

// baseCSP is the Content-Security-Policy of every generated page. It blocks
// scripts and every external resource.
const baseCSP = "default-src 'none'; style-src 'unsafe-inline'; img-src 'self' data:; base-uri 'none'"

// contentSecurityPolicy returns the policy for a page. When the page carries a
// script, the policy allows exactly that script by its SHA-256 hash and nothing
// else: no other inline script, no external script, and no eval.
func contentSecurityPolicy(script string) string {
	if script == "" {
		return baseCSP
	}
	sum := sha256.Sum256([]byte(script))
	return baseCSP + "; script-src 'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
}

// copyScript adds a Copy button to every code block. It is only included on
// pages that have a code block, and the page's policy allows it by hash.
//
// The buttons are added by script rather than written into the HTML, so a page
// opened with scripts disabled shows plain code blocks and no dead buttons.
// Code is read with textContent, which is exactly the text of the block. The
// asynchronous Clipboard API is used when the page is a secure context; the
// textarea and execCommand fallback covers browsers and file:// pages where it
// is not available. The result is announced to screen readers through a status
// region. Keep the script free of comments, regular expression literals, and
// angle brackets in strings: the page template inserts it verbatim, and its
// hash must match the bytes served.
const copyScript = `(function () {
  var blocks = document.querySelectorAll("pre > code");
  if (!blocks.length) { return; }

  function legacyCopy(text) {
    return new Promise(function (resolve, reject) {
      var area = document.createElement("textarea");
      area.value = text;
      area.setAttribute("readonly", "");
      area.style.position = "fixed";
      area.style.opacity = "0";
      document.body.appendChild(area);
      area.select();
      var ok = false;
      try { ok = document.execCommand("copy"); } catch (err) { ok = false; }
      document.body.removeChild(area);
      if (ok) { resolve(); } else { reject(new Error("copy failed")); }
    });
  }

  function copyText(text) {
    if (navigator.clipboard && window.isSecureContext) {
      return navigator.clipboard.writeText(text).catch(function () { return legacyCopy(text); });
    }
    return legacyCopy(text);
  }

  Array.prototype.forEach.call(blocks, function (code) {
    var pre = code.parentNode;
    var wrap = document.createElement("div");
    wrap.className = "code-block";
    pre.parentNode.insertBefore(wrap, pre);
    wrap.appendChild(pre);

    var button = document.createElement("button");
    button.type = "button";
    button.className = "copy-button";
    button.textContent = "Copy";
    button.setAttribute("aria-label", "Copy code to clipboard");

    var status = document.createElement("span");
    status.className = "sr-only";
    status.setAttribute("role", "status");

    var timer;
    function report(label, message) {
      button.textContent = label;
      status.textContent = message;
      clearTimeout(timer);
      timer = setTimeout(function () {
        button.textContent = "Copy";
        status.textContent = "";
      }, 2000);
    }

    button.addEventListener("click", function () {
      copyText(code.textContent).then(
        function () { report("Copied!", "Code copied to clipboard"); },
        function () { report("Failed", "Copy failed"); }
      );
    });

    wrap.appendChild(button);
    wrap.appendChild(status);
  });
})();`

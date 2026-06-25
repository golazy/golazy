<section class="glz-error" role="alert" aria-labelledby="glz-error-title" data-glz-open-editor-path="{{.openEditorPath}}">
  <style>
    .glz-error {
      --glz-error-bg: #fff1f2;
      --glz-error-border: #fecdd3;
      --glz-error-text: #4c0519;
      --glz-error-muted: #881337;
      --glz-error-code-bg: #ffe4e6;

      background: var(--glz-error-bg);
      border: 1px solid var(--glz-error-border);
      border-radius: 0.5rem;
      color: var(--glz-error-text);
      margin: clamp(1rem, 8vh, 4rem) auto;
      padding: clamp(1.25rem, 5vw, 2rem);
      width: min(calc(100% - 2rem), 56rem);
    }

    .glz-error__eyebrow {
      color: var(--glz-error-muted);
      font-size: 0.78rem;
      font-weight: 800;
      letter-spacing: 0;
      margin: 0 0 0.8rem;
      text-transform: uppercase;
    }

    .glz-error h1 {
      font-size: clamp(1.75rem, 7vw, 3.25rem);
      line-height: 1.05;
      margin: 0;
    }

    .glz-error__message {
      color: var(--glz-error-muted);
      font-size: clamp(1rem, 2.4vw, 1.15rem);
      margin: 1.2rem 0 0;
      overflow-wrap: anywhere;
    }

    .glz-error__backtrace {
      border-top: 1px solid var(--glz-error-border);
      margin-top: 1.5rem;
      padding-top: 1rem;
    }

    .glz-error__backtrace summary {
      color: var(--glz-error-text);
      cursor: pointer;
      font-weight: 700;
    }

    .glz-error__frames {
      display: grid;
      gap: 0.65rem;
      margin: 0.9rem 0 0;
      max-height: min(24rem, 45vh);
      overflow: auto;
      padding: 0;
    }

    .glz-error__frame {
      background: var(--glz-error-code-bg);
      border: 1px solid var(--glz-error-border);
      border-radius: 0.5rem;
      list-style-position: inside;
      padding: 0.7rem 0.8rem;
    }

    .glz-error__frame-button {
      appearance: none;
      background: transparent;
      border: 0;
      color: inherit;
      cursor: pointer;
      display: block;
      margin: 0;
      padding: 0;
      text-align: left;
      width: 100%;
    }

    .glz-error__frame-button:hover .glz-error__location,
    .glz-error__frame-button:focus-visible .glz-error__location {
      text-decoration: underline;
    }

    .glz-error__frame-button:focus-visible {
      outline: 2px solid var(--glz-error-text);
      outline-offset: 3px;
    }

    .glz-error__function,
    .glz-error__location {
      display: block;
      font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
      overflow-wrap: anywhere;
      white-space: pre-wrap;
    }

    .glz-error__function {
      color: var(--glz-error-text);
      font-size: 0.86rem;
      font-weight: 700;
    }

    .glz-error__location {
      color: var(--glz-error-muted);
      font-size: 0.78rem;
      margin-top: 0.3rem;
    }

    @media (prefers-color-scheme: dark) {
      .glz-error {
        --glz-error-bg: #3f121d;
        --glz-error-border: #7f1d1d;
        --glz-error-text: #ffe4e6;
        --glz-error-muted: #fecdd3;
        --glz-error-code-bg: #5f1723;
      }
    }
  </style>

  <p class="glz-error__eyebrow">Request error</p>
  <h1 id="glz-error-title">{{.status}} {{.statusText}}</h1>
  {{if .error}}
    <p class="glz-error__message">{{.error}}</p>
  {{else}}
    <p class="glz-error__message">The request could not be completed. Please try again later.</p>
  {{end}}
  {{if .backtrace}}
    <details class="glz-error__backtrace" open>
      <summary>Backtrace</summary>
      <ol class="glz-error__frames">
        {{range .backtrace}}
          <li class="glz-error__frame">
            {{if $.openEditorPath}}
              <button type="button" class="glz-error__frame-button" data-glz-open-editor data-glz-file="{{.AbsoluteFile}}" data-glz-line="{{.Line}}">
                {{if .Function}}<code class="glz-error__function">{{.Function}}</code>{{end}}
                {{if .File}}<code class="glz-error__location">{{.File}}{{if .Line}}:{{.Line}}{{end}}</code>{{end}}
              </button>
            {{else}}
              {{if .Function}}<code class="glz-error__function">{{.Function}}</code>{{end}}
              {{if .File}}<code class="glz-error__location">{{.File}}{{if .Line}}:{{.Line}}{{end}}</code>{{end}}
            {{end}}
          </li>
        {{end}}
      </ol>
    </details>
  {{end}}
  {{if .openEditorPath}}
    <script>
      (() => {
        const root = document.currentScript.closest(".glz-error");
        const endpoint = root?.dataset.glzOpenEditorPath;
        if (!endpoint) return;

        root.addEventListener("click", (event) => {
          const trigger = event.target.closest("[data-glz-open-editor]");
          if (!trigger) return;

          event.preventDefault();
          const file = trigger.dataset.glzFile;
          const line = Number(trigger.dataset.glzLine || "0");
          if (!file) return;

          fetch(endpoint, {
            method: "POST",
            headers: {"Content-Type": "application/json"},
            body: JSON.stringify({file, line}),
            keepalive: true
          }).catch(() => {});
        });
      })();
    </script>
  {{end}}
</section>

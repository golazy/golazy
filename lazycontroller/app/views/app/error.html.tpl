<section class="glz-error" role="alert" aria-labelledby="glz-error-title">
  <style>
    .glz-error {
      background: var(--glz-panel);
      border: 1px solid var(--glz-border);
      border-radius: 1rem;
      box-shadow: var(--glz-shadow);
      padding: clamp(1.25rem, 5vw, 3rem);
    }

    .glz-error__eyebrow {
      color: var(--glz-accent);
      font-size: 0.78rem;
      font-weight: 800;
      letter-spacing: 0.12em;
      margin: 0 0 0.8rem;
      text-transform: uppercase;
    }

    .glz-error h1 {
      font-size: clamp(2rem, 8vw, 4.5rem);
      line-height: 0.98;
      margin: 0;
    }

    .glz-error__message {
      color: var(--glz-muted);
      font-size: clamp(1rem, 2.4vw, 1.15rem);
      margin: 1.2rem 0 0;
      overflow-wrap: anywhere;
    }
  </style>

  <p class="glz-error__eyebrow">Request error</p>
  <h1 id="glz-error-title">{{.status}} {{.statusText}}</h1>
  {{if .error}}
    <p class="glz-error__message">{{.error}}</p>
  {{else}}
    <p class="glz-error__message">The request could not be completed. Please try again later.</p>
  {{end}}
</section>

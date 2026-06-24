<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>GoLazy</title>
    <style>
      :root {
        color-scheme: light dark;
        --glz-bg: #f6f7fb;
        --glz-panel: #ffffff;
        --glz-text: #16181f;
        --glz-muted: #5f6677;
        --glz-border: #d8deea;
        --glz-accent: #0a7cff;
        --glz-shadow: 0 24px 80px rgba(20, 28, 45, 0.16);
      }

      * {
        box-sizing: border-box;
      }

      body {
        margin: 0;
        min-height: 100vh;
        background:
          radial-gradient(circle at top left, rgba(10, 124, 255, 0.14), transparent 34rem),
          var(--glz-bg);
        color: var(--glz-text);
        font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
        line-height: 1.5;
      }

      .glz-page {
        display: grid;
        min-height: 100vh;
        place-items: center;
        padding: clamp(1rem, 4vw, 3rem);
      }

      .glz-panel {
        width: min(100%, 46rem);
      }

      @media (prefers-color-scheme: dark) {
        :root {
          --glz-bg: #11141b;
          --glz-panel: #181d27;
          --glz-text: #eef2ff;
          --glz-muted: #a9b1c3;
          --glz-border: #2d3545;
          --glz-accent: #70b7ff;
          --glz-shadow: 0 24px 80px rgba(0, 0, 0, 0.38);
        }
      }
    </style>
  </head>
  <body>
    <main class="glz-page">
      <div class="glz-panel">
        {{.content}}
      </div>
    </main>
  </body>
</html>

import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import "./styles/globals.css";

// Render a visible error message in case React fails to mount or
// throws during the first render. Without this you get a silently
// blank window — easy to confuse with "the app is hung".
class BootErrorBoundary extends React.Component<
  { children: React.ReactNode },
  { error: Error | null }
> {
  state = { error: null as Error | null };
  static getDerivedStateFromError(error: Error) {
    return { error };
  }
  componentDidCatch(error: Error, info: React.ErrorInfo) {
    // eslint-disable-next-line no-console
    console.error("Portal boot error:", error, info);
  }
  render() {
    if (this.state.error) {
      return (
        <div
          style={{
            color: "#fca5a5",
            padding: 24,
            fontFamily: "JetBrains Mono, ui-monospace, monospace",
            fontSize: 13,
            whiteSpace: "pre-wrap",
            lineHeight: 1.6,
          }}
        >
          <div style={{ color: "#f87171", fontWeight: 600, marginBottom: 12 }}>
            Portal boot failed
          </div>
          {this.state.error.message}
          {this.state.error.stack ? "\n\n" + this.state.error.stack : ""}
        </div>
      );
    }
    return <>{this.props.children}</>;
  }
}

const rootEl = document.getElementById("root");
if (!rootEl) {
  document.body.innerHTML =
    '<div style="color:red;padding:24px;font-family:monospace;">root element missing in DOM</div>';
} else {
  try {
    ReactDOM.createRoot(rootEl).render(
      <React.StrictMode>
        <BootErrorBoundary>
          <App />
        </BootErrorBoundary>
      </React.StrictMode>
    );
  } catch (e: any) {
    rootEl.innerHTML =
      '<div style="color:red;padding:24px;font-family:monospace;white-space:pre-wrap;">' +
      "React.createRoot threw:\n" +
      String(e?.stack || e?.message || e) +
      "</div>";
  }
}

// Surface uncaught runtime errors that escape React's tree. Shows in
// the bottom corner so we can read them without DevTools.
window.addEventListener("error", (e) => showError(e.error || e.message));
window.addEventListener("unhandledrejection", (e) => showError(e.reason));

function showError(err: any) {
  // eslint-disable-next-line no-console
  console.error("Portal global error:", err);
  let div = document.getElementById("portal-global-err");
  if (!div) {
    div = document.createElement("div");
    div.id = "portal-global-err";
    div.style.cssText =
      "position:fixed;left:12px;bottom:12px;max-width:60%;background:rgba(244,63,94,.18);border:1px solid #f87171;color:#fecaca;padding:10px 14px;border-radius:8px;font-family:JetBrains Mono,monospace;font-size:11px;white-space:pre-wrap;z-index:9999;";
    document.body.appendChild(div);
  }
  div.textContent = (err?.stack || err?.message || String(err)).slice(0, 1500);
}

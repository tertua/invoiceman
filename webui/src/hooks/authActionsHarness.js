import { register } from "node:module";
import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { QueryClientProvider } from "@tanstack/react-query";

// Shared harness for the useAuthActions tests: a Vite-less resolver for "@/"
// and extensionless imports, plus SSR helpers to render the hook once.
export const resolveHook = `import { existsSync } from "node:fs";
import { fileURLToPath } from "node:url";
const base = ${JSON.stringify(new URL("../", import.meta.url).href)};
export async function resolve(specifier, context, nextResolve) {
  if (specifier.startsWith("@/")) {
    let rel = specifier.slice(2);
    if (!/\\.[a-z]+$/.test(rel)) rel += ".js";
    return nextResolve(new URL(rel, base).href, context);
  }
  if (/^\\.\\.?\\//.test(specifier) && !/\\.[a-z]+$/.test(specifier) && context.parentURL) {
    const url = new URL(specifier + ".js", context.parentURL).href;
    if (existsSync(fileURLToPath(url))) return nextResolve(url, context);
  }
  return nextResolve(specifier, context);
}`;

export function installResolver() {
  register("data:text/javascript," + encodeURIComponent(resolveHook));
}

// renderActions renders the hook once via SSR and returns the captured actions.
export function renderActions(client, setters, useAuthActions) {
  let captured;
  function Probe() {
    captured = useAuthActions(setters);
    return null;
  }
  renderToStaticMarkup(createElement(QueryClientProvider, { client }, createElement(Probe)));
  return captured;
}

export function fakeSetters() {
  const calls = { user: [], loading: [], expired: [] };
  return {
    calls,
    setUser: (v) => calls.user.push(v),
    setLoading: (v) => calls.loading.push(v),
    setSessionExpired: (v) => calls.expired.push(v),
  };
}

export async function withAuthStubs(authApi, stubs, run) {
  const original = {};
  for (const [name, fn] of Object.entries(stubs)) {
    original[name] = authApi[name];
    authApi[name] = fn;
  }
  try {
    return await run();
  } finally {
    Object.assign(authApi, original);
  }
}

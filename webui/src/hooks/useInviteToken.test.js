import { test } from "node:test";
import assert from "node:assert/strict";
import { register } from "node:module";
import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { MemoryRouter } from "react-router-dom";

// node --test lacks Vite's resolution, so register a tiny resolver for "@/" and extensionless imports.
const resolveHook = `import { existsSync } from "node:fs";
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
register("data:text/javascript," + encodeURIComponent(resolveHook));

const { useInviteToken } = await import("./useInviteToken.js");

// useInviteToken reads the query string via react-router, so render it under a MemoryRouter.
function readToken(entry) {
  let captured;
  function Probe() {
    captured = useInviteToken();
    return null;
  }
  renderToStaticMarkup(
    createElement(MemoryRouter, { initialEntries: [entry] }, createElement(Probe)),
  );
  return captured;
}

test("useInviteToken reads invite_token from the query string", () => {
  assert.equal(readToken("/register?invite_token=abc123"), "abc123");
});

test("useInviteToken decodes a URL-encoded token", () => {
  assert.equal(readToken("/register?invite_token=a%2Fb%2Bc"), "a/b+c");
});

test("useInviteToken is empty when the parameter is absent", () => {
  assert.equal(readToken("/register"), "");
});

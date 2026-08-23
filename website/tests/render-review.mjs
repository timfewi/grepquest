import { chromium } from "playwright";
import AxeBuilder from "@axe-core/playwright";
import { mkdir } from "node:fs/promises";

const baseURL = process.env.REVIEW_URL ?? "http://127.0.0.1:4321/";
const outputDirectory = new URL("../review/", import.meta.url);
await mkdir(outputDirectory, { recursive: true });

const browser = await chromium.launch({
  headless: true,
  executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE,
});
const results = [];

async function inspect(name, viewport, options = {}) {
  const context = await browser.newContext({
    viewport,
    reducedMotion: options.reducedMotion ? "reduce" : "no-preference",
  });
  const page = await context.newPage();
  const externalRequests = [];
  page.on("request", (request) => {
    const url = new URL(request.url());
    if (url.hostname !== "127.0.0.1") externalRequests.push(request.url());
  });
  await page.goto(baseURL, { waitUntil: "networkidle" });
  if (name === "desktop-1440" || name === "mobile-390") {
    await page.screenshot({ path: new URL(`${name}-viewport.png`, outputDirectory).pathname });
  }
  await page.screenshot({ path: new URL(`${name}.png`, outputDirectory).pathname, fullPage: true });
  if (name === "desktop-1440") {
    await page.locator(".core-loop").screenshot({ path: new URL("desktop-1440-core-loop.png", outputDirectory).pathname });
  }
  if (name === "desktop-1440" || name === "mobile-390") {
    await page.locator(".ranger-section").scrollIntoViewIfNeeded();
    await page.locator(".ranger-portrait img").waitFor({ state: "visible" });
    await page.waitForFunction(() => {
      const image = document.querySelector(".ranger-portrait img");
      return image instanceof HTMLImageElement && image.complete && image.naturalWidth > 0;
    });
    await page.locator(".ranger-section").screenshot({ path: new URL(`${name}-ranger.png`, outputDirectory).pathname });
  }

  const layout = await page.evaluate(() => ({
    viewport: document.documentElement.clientWidth,
    documentWidth: document.documentElement.scrollWidth,
    headingCount: document.querySelectorAll("h1").length,
    h1: document.querySelector("h1")?.textContent?.replace(/\s+/g, " ").trim(),
    activeTargetMin: Math.min(
      ...Array.from(document.querySelectorAll("a, button, summary"), (node) => {
        const rect = node.getBoundingClientRect();
        if (rect.width === 0 || rect.height === 0) return Number.POSITIVE_INFINITY;
        return Math.min(rect.width, rect.height);
      }),
    ),
    overflowing: Array.from(document.querySelectorAll("body *"))
      .filter((node) => {
        const rect = node.getBoundingClientRect();
        return rect.right > document.documentElement.clientWidth + 1 || rect.left < -1;
      })
      .slice(0, 12)
      .map((node) => ({
        tag: node.tagName.toLowerCase(),
        className: node.className,
        text: node.textContent?.replace(/\s+/g, " ").trim().slice(0, 60),
        left: Math.round(node.getBoundingClientRect().left),
        right: Math.round(node.getBoundingClientRect().right),
      })),
    brokenAnchors: Array.from(document.querySelectorAll('a[href^="#"]'))
      .map((link) => link.getAttribute("href"))
      .filter((href) => href && !document.querySelector(href)),
  }));

  const accessibility = await new AxeBuilder({ page }).analyze();
  results.push({
    name,
    layout,
    axeViolations: accessibility.violations.map(({ id, impact, nodes }) => ({
      id,
      impact,
      nodes: nodes.map((node) => node.target),
    })),
    externalRequests,
  });
  await context.close();
}

await inspect("desktop-1440", { width: 1440, height: 1000 });
await inspect("tablet-820", { width: 820, height: 1180 });
await inspect("mobile-390", { width: 390, height: 844 });
await inspect("mobile-320", { width: 320, height: 780 });
await inspect("reduced-motion-390", { width: 390, height: 844 }, { reducedMotion: true });

const interactionContext = await browser.newContext({ viewport: { width: 390, height: 844 }, permissions: ["clipboard-read", "clipboard-write"] });
const interactionPage = await interactionContext.newPage();
await interactionPage.goto(baseURL, { waitUntil: "networkidle" });
await interactionPage.locator(".mobile-menu summary").click();
const menuOpen = await interactionPage.locator(".mobile-menu").evaluate((node) => node.open);
await interactionPage.locator("[data-pause]").click();
const paused = await interactionPage.locator("[data-pause]").getAttribute("aria-pressed");
await interactionPage.locator("[data-pause]").click();
await interactionPage.locator("[data-replay]").click();
await interactionPage.locator("#install").scrollIntoViewIfNeeded();
await interactionPage.locator("[data-copy]").click();
await interactionPage.waitForTimeout(100);
const copyLabel = await interactionPage.locator("[data-copy-label]").textContent();
await interactionPage.goto(baseURL, { waitUntil: "networkidle" });
await interactionPage.keyboard.press("Tab");
const firstFocus = await interactionPage.evaluate(() => ({
  text: document.activeElement?.textContent?.trim(),
  visible: document.activeElement instanceof HTMLElement && document.activeElement.getBoundingClientRect().top >= 0,
}));
results.push({ name: "interactions", menuOpen, paused, copyLabel, firstFocus });
await interactionContext.close();

const zoomContext = await browser.newContext({ viewport: { width: 1280, height: 900 } });
const zoomPage = await zoomContext.newPage();
await zoomPage.goto(baseURL, { waitUntil: "networkidle" });
await zoomPage.evaluate(() => { document.documentElement.style.fontSize = "32px"; });
await zoomPage.screenshot({ path: new URL("text-zoom-200.png", outputDirectory).pathname, fullPage: true });
results.push({
  name: "text-zoom-200",
  layout: await zoomPage.evaluate(() => ({
    viewport: document.documentElement.clientWidth,
    documentWidth: document.documentElement.scrollWidth,
  })),
});
await zoomContext.close();

await browser.close();
console.log(JSON.stringify(results, null, 2));

import { chromium } from "playwright";

const baseURL = process.env.REVIEW_URL ?? "http://127.0.0.1:4321/";
const browser = await chromium.launch({
  headless: true,
  executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE,
});
const page = await browser.newPage({ viewport: { width: 1200, height: 630 } });

await page.goto(baseURL, { waitUntil: "networkidle" });
await page.waitForTimeout(6100);
await page.screenshot({ path: new URL("../public/images/social-card.png", import.meta.url).pathname });

await browser.close();

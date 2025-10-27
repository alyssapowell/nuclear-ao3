import { test, expect } from "@playwright/test";

test("minimal connectivity test", async ({ page }) => {
  test.setTimeout(15000);
  
  // Test that server responds at all
  await page.goto("http://localhost:3002/");
  
  // Wait for any content to appear
  await page.waitForSelector("body", { timeout: 10000 });
  
  // Check if page has any content
  const content = await page.textContent("body");
  expect(content).toBeTruthy();
  
  console.log("Page content:", content?.substring(0, 200));
});

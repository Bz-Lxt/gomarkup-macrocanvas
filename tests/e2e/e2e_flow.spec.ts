import { test, expect } from "@playwright/test";

test("login, canvas, run P10, emergency stop", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "HID 宏编排控制台" })).toBeVisible();
  await page.getByRole("button", { name: "进入控制台" }).click();
  await expect(page.getByTestId("canvas")).toBeVisible({ timeout: 20000 });
  await expect(page.getByTestId("tier-badge")).toContainText(/T-[ABC]/);
  await expect(page.getByTestId("canvas").getByText(/P10/)).toBeVisible();
  await page.getByTestId("btn-run").click();
  await expect(page.getByText(/执行 succeeded|执行 stopped/)).toBeVisible({ timeout: 20000 });
  await page.getByTestId("btn-estop").click();
  await expect(page.getByText("紧急停止已触发")).toBeVisible();
  await page.getByRole("button", { name: "事件墙" }).click();
  await page.getByTestId("btn-capture").click();
  await page.getByTestId("confirm-ok").click();
  await expect(page.getByTestId("event-wall")).toBeVisible();
});

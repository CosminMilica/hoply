import { test as base } from '@playwright/test';
import { Navigation } from '../actions/Navigation';
import { ApiHelper } from '../actions/ApiHelper';
import { Assertions } from '../actions/Assertions';


export type AppFixtures = {
  app: {
    nav: Navigation;
    api: ApiHelper;
    assertions: Assertions;
    page: import('@playwright/test').Page;

  };
};

export const test = base.extend<AppFixtures>({
  app: async ({ page }, use) => {
    await use({
      nav: new Navigation(page),
      api: new ApiHelper(page),
      assertions: new Assertions(page),
      page,

    });
  },
});

export { expect } from '@playwright/test';

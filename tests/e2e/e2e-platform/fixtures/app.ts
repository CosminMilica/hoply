import { test as base } from '@playwright/test';
import { Navigation } from '../actions/Navigation';
import { ApiHelper } from '../actions/ApiHelper';
import { Assertions } from '../actions/Assertions';

import { AuthManager } from '../auth/AuthManager';


export type AppFixtures = {
  app: {
    nav: Navigation;
    api: ApiHelper;
    assertions: Assertions;
    page: import('@playwright/test').Page;

    auth: AuthManager;

  };
};

export const test = base.extend<AppFixtures>({
  app: async ({ page, context }, use) => {
    await use({
      nav: new Navigation(page),
      api: new ApiHelper(page),
      assertions: new Assertions(page),
      page,

      auth: new AuthManager(page, context),

    });
  },
});

export { expect } from '@playwright/test';

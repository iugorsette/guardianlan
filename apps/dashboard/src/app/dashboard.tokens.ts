import { InjectionToken } from '@angular/core';

export const DASHBOARD_AUTO_REFRESH_MS = new InjectionToken<number>(
  'DASHBOARD_AUTO_REFRESH_MS',
  {
    factory: () => 15000
  }
);


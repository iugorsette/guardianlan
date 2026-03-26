import {
  ApplicationConfig,
  provideBrowserGlobalErrorListeners,
  provideZonelessChangeDetection
} from '@angular/core';
import { provideHttpClient } from '@angular/common/http';

import { DASHBOARD_AUTO_REFRESH_MS } from './dashboard.tokens';

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    provideZonelessChangeDetection(),
    provideHttpClient(),
    { provide: DASHBOARD_AUTO_REFRESH_MS, useValue: 15000 }
  ]
};

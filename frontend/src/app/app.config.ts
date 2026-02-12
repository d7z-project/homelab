import { ApplicationConfig, provideBrowserGlobalErrorListeners } from '@angular/core';
import {provideRouter, withHashLocation} from '@angular/router';
import { provideHttpClient } from '@angular/common/http';
import { provideApi } from './generated';

import { routes } from './app.routes';

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    provideRouter(routes,withHashLocation()),
    provideHttpClient(),
    provideApi(`${window.location.origin}/api/v1`)
  ]
};
    
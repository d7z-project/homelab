import { ApplicationConfig, provideBrowserGlobalErrorListeners } from '@angular/core';
import { provideRouter, withHashLocation } from '@angular/router';
import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { provideApi } from './generated';

import { routes } from './app.routes';
import { registerLocaleData } from '@angular/common';
import zh from '@angular/common/locales/zh';
import { authInterceptor } from './auth.interceptor';

registerLocaleData(zh);

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    provideRouter(routes, withHashLocation()),
    provideHttpClient(withInterceptors([authInterceptor])),
    provideApi(`${window.location.origin}/api/v1`),
  ],
};

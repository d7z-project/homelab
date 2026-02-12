import { ApplicationConfig, provideBrowserGlobalErrorListeners } from '@angular/core';
import { provideRouter, withHashLocation } from '@angular/router';
import { provideHttpClient } from '@angular/common/http';
import { provideApi } from './generated';

import { routes } from './app.routes';
import { zh_CN, provideNzI18n } from 'ng-zorro-antd/i18n';
import { registerLocaleData } from '@angular/common';
import zh from '@angular/common/locales/zh';
import { provideAnimations } from '@angular/platform-browser/animations';
import { provideNzIcons } from 'ng-zorro-antd/icon';
import {
  UserOutline,
  LockOutline,
  DashboardOutline,
  MenuUnfoldOutline,
  MenuFoldOutline,
  DatabaseOutline,
  ApiOutline,
  ContainerOutline,
  SafetyOutline,
} from '@ant-design/icons-angular/icons';

registerLocaleData(zh);

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    provideRouter(routes, withHashLocation()),
    provideHttpClient(),
    provideApi(`${window.location.origin}/api/v1`),
    provideNzI18n(zh_CN),
    provideAnimations(),
    provideNzIcons([
      LockOutline,
      DashboardOutline,
      MenuUnfoldOutline,
      MenuFoldOutline,
      DatabaseOutline,
      ApiOutline,
      ContainerOutline,
      SafetyOutline,
    ]),
  ],
};

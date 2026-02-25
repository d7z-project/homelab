import { ApplicationConfig, provideBrowserGlobalErrorListeners } from '@angular/core';
import { provideRouter, withHashLocation } from '@angular/router';
import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { provideApi } from './generated';

import { routes } from './app.routes';
import { registerLocaleData } from '@angular/common';
import zh from '@angular/common/locales/zh';
import { authInterceptor } from './auth.interceptor';
import { MatPaginatorIntl } from '@angular/material/paginator';

registerLocaleData(zh);

export function getZhPaginatorIntl() {
  const paginatorIntl = new MatPaginatorIntl();
  paginatorIntl.itemsPerPageLabel = '每页条数:';
  paginatorIntl.nextPageLabel = '下一页';
  paginatorIntl.previousPageLabel = '上一页';
  paginatorIntl.firstPageLabel = '首页';
  paginatorIntl.lastPageLabel = '尾页';
  paginatorIntl.getRangeLabel = (page: number, pageSize: number, length: number) => {
    if (length === 0 || pageSize === 0) {
      return `0 / ${length}`;
    }
    length = Math.max(length, 0);
    const startIndex = page * pageSize;
    const endIndex =
      startIndex < length ? Math.min(startIndex + pageSize, length) : startIndex + pageSize;
    return `${startIndex + 1} – ${endIndex} / ${length}`;
  };
  return paginatorIntl;
}

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    provideRouter(routes),
    provideHttpClient(withInterceptors([authInterceptor])),
    provideApi(`${window.location.origin}/api/v1`),
    { provide: MatPaginatorIntl, useValue: getZhPaginatorIntl() },
  ],
};

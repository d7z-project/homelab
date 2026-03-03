import {
  HttpInterceptorFn,
  HttpRequest,
  HttpHandlerFn,
  HttpEvent,
  HttpErrorResponse,
} from '@angular/common/http';
import { Observable, tap } from 'rxjs';
import { inject } from '@angular/core';
import { Router } from '@angular/router';

export const authInterceptor: HttpInterceptorFn = (
  req: HttpRequest<unknown>,
  next: HttpHandlerFn,
): Observable<HttpEvent<unknown>> => {
  const sessionId = localStorage.getItem('session_id');
  const router = inject(Router);

  let authReq = req;
  if (sessionId) {
    authReq = req.clone({
      setHeaders: {
        Authorization: `Bearer ${sessionId}`,
      },
    });
  }

  return next(authReq).pipe(
    tap({
      error: (err: any) => {
        if (err instanceof HttpErrorResponse && err.status === 401) {
          // If the error code is 10001 (TOTP required), let the login component handle it
          if (err.error && err.error.code === 10001) {
            return;
          }
          localStorage.clear();
          router.navigate(['/login']);
        }
      },
    }),
  );
};

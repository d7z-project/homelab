import {
  Component,
  OnInit,
  OnDestroy,
  inject,
  ChangeDetectorRef,
  ViewChild,
  ElementRef,
} from '@angular/core';
import { FormBuilder, FormGroup, Validators, ReactiveFormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { CommonModule } from '@angular/common';
import { AuthService } from '../../generated';
import { MatSnackBar } from '@angular/material/snack-bar';
import { firstValueFrom } from 'rxjs';

import { HttpErrorResponse } from '@angular/common/http';

@Component({
  selector: 'app-login',

  standalone: true,

  imports: [
    CommonModule,

    ReactiveFormsModule,

    MatFormFieldModule,

    MatInputModule,

    MatButtonModule,

    MatCardModule,

    MatIconModule,

    MatProgressBarModule,
  ],

  templateUrl: './login.component.html',
  styles: [
    `
      .brand-panel {
        background-color: #0f172a; /* Slate 900 */
        position: relative;
      }

      .brand-content {
        max-width: 400px;
      }

      .brand-logo-icon {
        color: #6366f1; /* Indigo 500 */
        filter: drop-shadow(0 0 8px rgba(99, 102, 241, 0.2));
      }
    `,
  ],
})
export class LoginComponent implements OnInit, OnDestroy {
  @ViewChild('totpInput') totpInput?: ElementRef;

  validateForm!: FormGroup;

  loading = false;

  showTotp = false;

  totpProgress = 0;

  private progressInterval: any;

  private fb = inject(FormBuilder);

  private router = inject(Router);

  private authService = inject(AuthService);

  private snackBar = inject(MatSnackBar);

  private cdr = inject(ChangeDetectorRef);

  ngOnInit(): void {
    this.validateForm = this.fb.group({
      password: ['', [Validators.required]],

      totp: [''],
    });
  }

  async submitForm(): Promise<void> {
    if (this.validateForm.valid) {
      requestAnimationFrame(() => {
        this.loading = true;
        this.cdr.detectChanges();
      });

      try {
        const res = await firstValueFrom(
          this.authService.authLoginPost({
            password: this.validateForm.value.password,
            totp: this.validateForm.value.totp,
          }),
        );

        localStorage.setItem('session_id', res.session_id!);
        this.snackBar.open('登录成功', '关闭', { duration: 3000 });
        this.router.navigate(['/']);
      } catch (err) {
        let errorMsg = '登录失败，请检查密码或验证码';

        if (err instanceof HttpErrorResponse && err.error) {
          const apiError = err.error;

          if (apiError.code === 10001) {
            requestAnimationFrame(() => {
              this.showTotp = true;
              this.startProgressTimer();
              this.validateForm.get('totp')?.setValidators([Validators.required]);
              this.validateForm.get('totp')?.updateValueAndValidity();
              this.cdr.detectChanges();
            });

            // 更明确的安全校验提示
            this.snackBar.open('身份核验通过，请输入 2FA 动态验证码', '了解', {
              duration: 5000,
              panelClass: ['security-snack'],
            });

            setTimeout(() => {
              this.totpInput?.nativeElement.focus();
            }, 100);

            return;
          } else if (apiError.code === 10000) {
            errorMsg = '密码或验证码错误';
          } else if (apiError.message) {
            errorMsg = apiError.message;
          }
        }

        this.snackBar.open(errorMsg, '关闭', { duration: 3000 });
      } finally {
        requestAnimationFrame(() => {
          this.loading = false;
          this.cdr.detectChanges();
        });
      }
    } else {
      Object.values(this.validateForm.controls).forEach((control) => {
        if (control.invalid) {
          control.markAsDirty();

          control.updateValueAndValidity({ onlySelf: true });
        }
      });
    }
  }

  resetLogin(): void {
    requestAnimationFrame(() => {
      this.showTotp = false;
      this.stopProgressTimer();
      this.validateForm.get('totp')?.clearValidators();
      this.validateForm.get('totp')?.setValue('');
      this.validateForm.get('totp')?.updateValueAndValidity();
      this.cdr.detectChanges();
    });
  }

  private startProgressTimer(): void {
    this.stopProgressTimer();
    this.updateProgress();
    this.progressInterval = setInterval(() => {
      this.updateProgress();
    }, 100);
  }

  private stopProgressTimer(): void {
    if (this.progressInterval) {
      clearInterval(this.progressInterval);
      this.progressInterval = null;
    }
  }

  private updateProgress(): void {
    requestAnimationFrame(() => {
      const now = new Date();
      const seconds = now.getSeconds() % 30;
      const milliseconds = now.getMilliseconds();
      // 30s = 30000ms
      this.totpProgress = ((seconds * 1000 + milliseconds) / 30000) * 100;
      this.cdr.detectChanges();
    });
  }

  ngOnDestroy(): void {
    this.stopProgressTimer();
  }
}
